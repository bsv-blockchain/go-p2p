package p2p

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	alertP2P "github.com/bitcoin-sv/alert-system/app/p2p"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/libp2p/go-libp2p/core/protocol"
	dRouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dUtil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/sync/errgroup"
)

// NewNode creates and initializes a new P2P network node with the provided configuration.
// This constructor performs the core setup of the libp2p networking stack, including:
//   - Setting up the node's cryptographic identity (private key)
//   - Configuring network transports and listeners
//   - Initializing the DHT (Distributed Hash Table) for peer discovery
//   - Preparing topic handlers and message routing systems
//
// Parameters:
//   - ctx: Context for controlling the initialization process
//   - logger: Logger for recording initialization and operational events
//   - config: P2P-specific configuration parameters defining network behavior
//
// Returns a fully initialized P2P node ready for starting, or an error if initialization fails.
func NewNode(ctx context.Context, logger Logger, config Config) (*Node, error) {
	logger.Infof("[Node] Creating node")

	var (
		err error
		h   host.Host       // the libp2p host for the node
		pk  *crypto.PrivKey // the private key for the node's identity
	)

	if config.PrivateKey == "" {
		pk, err = generatePrivateKey(ctx)
		if err != nil {
			return nil, fmt.Errorf("[Node] error generating private key: %w", err)
		}
	} else {
		// Decode the provided private key from hex format
		pk, err = decodeHexEd25519PrivateKey(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("[Node] error decoding private key: %w", err)
		}
	}

	// If a private DHT is configured, set up the private network
	if config.UsePrivateDHT {
		h, err = setUpPrivateNetwork(logger, config, pk)
		if err != nil {
			return nil, fmt.Errorf("[Node] error setting up private network: %w", err)
		}
	} else {
		// If no private DHT is configured, create a standard libp2p host
		var listenMultiAddresses []string
		for _, addr := range config.ListenAddresses {
			listenMultiAddresses = append(listenMultiAddresses, fmt.Sprintf("/ip4/%s/tcp/%d", addr, config.Port))
		}

		opts := []libp2p.Option{
			libp2p.ListenAddrStrings(listenMultiAddresses...),
			libp2p.Identity(*pk),
		}

		// If advertise addresses are specified, add them to the options
		addrsToAdvertise := buildAdvertiseMultiAddrs(logger, config.AdvertiseAddresses, config.Port)
		if len(addrsToAdvertise) > 0 {
			opts = append(opts, libp2p.AddrsFactory(func(_ []multiaddr.Multiaddr) []multiaddr.Multiaddr {
				return addrsToAdvertise
			}))
		} else {
			// User has not specified any broadcast addresses in their config, and we are not using a private DHT
			// define address factory to remove all private IPs from being advertised
			opts = append(opts, libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
				var publicAddrs []multiaddr.Multiaddr

				for _, addr := range addrs {
					// if IP is not private add it to the list
					if !isPrivateIP(addr) {
						publicAddrs = append(publicAddrs, addr)
					}
				}

				// If we still don't have any advertisable addresses then attempt to grab it from `https://ifconfig.me/ip`
				if len(publicAddrs) > 0 {
					return publicAddrs
				}

				// If no public addresses are set, let's attempt to grab it publicly
				// Ignore errors because we don't care if we can't find it
				var ifconfig string
				ifconfig, localErr := alertP2P.GetPublicIP(context.Background())
				if localErr != nil {
					logger.Debugf("[Node] error getting public IP: %v", localErr)
				}

				if len(ifconfig) == 0 {
					return publicAddrs
				}

				var addr multiaddr.Multiaddr
				addr, localErr = multiaddr.NewMultiaddr(fmt.Sprintf(multiAddrIPTemplate, ifconfig, config.Port))
				if localErr != nil {
					logger.Debugf("[Node] error creating public multiaddr: %v", localErr)
				}

				if addr != nil {
					publicAddrs = append(publicAddrs, addr)
				}

				return publicAddrs
			}))
		}

		h, err = libp2p.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("[Node] error creating libp2p host: %w", err)
		}
	}

	logger.Infof("[Node] peer ID: %s", h.ID().String())
	logger.Infof("[Node] Connect to me on:")

	for _, addr := range h.Addrs() {
		logger.Infof("[Node]   %s/p2p/%s", addr, h.ID().String())
	}

	node := &Node{
		config:            config,
		logger:            logger,
		host:              h,
		bitcoinProtocolID: "teranode/bitcoin/1.0.0",
		handlerByTopic:    make(map[string]Handler),
		startTime:         time.Now(),
		peerHeights:       sync.Map{},
		peerConnTimes:     sync.Map{},
	}

	// Set up connection notifications
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(_ network.Network, conn network.Conn) {
			peerID := conn.RemotePeer()
			node.logger.Debugf("[Node] Peer connected: %s", peerID.String())

			// Store connection time
			node.peerConnTimes.Store(peerID, time.Now())

			// Notify any connection handlers about the new peer
			node.callPeerConnected(context.Background(), peerID)
		},
		DisconnectedF: func(_ network.Network, conn network.Conn) {
			peerID := conn.RemotePeer()
			node.logger.Debugf("[Node] Peer disconnected: %s", peerID.String())

			// Remove connection time when peer disconnects
			node.peerConnTimes.Delete(peerID)
		},
	})

	return node, nil
}

// setUpPrivateNetwork creates a libp2p host configured for a private network using a pre-shared key.
// This function establishes a secure, isolated P2P network that only allows connections between
// nodes that possess the same shared key. It's used for creating private blockchain networks
// or testing environments where public network access should be restricted.
//
// The function constructs a pre-shared key (PSK) from the provided shared key string and
// configures the libp2p host to use this PSK for all network communications. Only peers
// with the matching PSK can establish connections and participate in the network.
//
// Parameters:
//   - config: P2P configuration containing the shared key and other network parameters
//   - pk: Private key for the node's cryptographic identity
//
// Returns:
//   - A configured libp2p host ready for private network operation
//   - Error if the shared key is invalid or host creation fails
func setUpPrivateNetwork(logger Logger, config Config, pk *crypto.PrivKey) (host.Host, error) {
	var h host.Host

	s := ""
	s += fmt.Sprintln("/key/swarm/psk/1.0.0/")
	s += fmt.Sprintln("/base16/")
	s += config.SharedKey

	buf := bytes.NewBufferString(s)
	psk, err := pnet.DecodeV1PSK(buf)
	if err != nil {
		return nil, fmt.Errorf("[Node] error decoding shared key: %w", err)
	}

	listenMultiAddresses := make([]string, 0, len(config.ListenAddresses))
	for _, addr := range config.ListenAddresses {
		listenMultiAddresses = append(listenMultiAddresses, fmt.Sprintf(multiAddrIPTemplate, addr, config.Port))
	}

	// Set up libp2p options
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listenMultiAddresses...),
		libp2p.Identity(*pk),
		libp2p.PrivateNetwork(psk),
	}

	// If advertise addresses are specified, add them to the options
	addrsToAdvertise := buildAdvertiseMultiAddrs(logger, config.AdvertiseAddresses, config.Port)
	if len(addrsToAdvertise) > 0 {
		opts = append(opts, libp2p.AddrsFactory(func(_ []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return addrsToAdvertise
		}))
	}

	h, err = libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("[Node] error creating libp2p node: %w", err)
	}

	return h, nil
}

// buildAdvertiseMultiAddrs constructs multiaddrs from host strings with optional ports.
// Logs warnings via fmt.Printf for invalid addresses.
func buildAdvertiseMultiAddrs(log Logger, addrs []string, defaultPort int) []multiaddr.Multiaddr {
	result := make([]multiaddr.Multiaddr, 0, len(addrs))

	for _, addr := range addrs {
		hostStr := addr
		portNum := defaultPort

		if h, p, err := net.SplitHostPort(addr); err == nil {
			hostStr = h

			var pi int
			pi, err = strconv.Atoi(p)
			if err != nil {
				log.Debugf("invalid port in advertise address: %s, error: %v\n", addr, err)
				continue
			}
			portNum = pi
		}

		var maddr multiaddr.Multiaddr

		var err error

		if net.ParseIP(hostStr) != nil {
			maddr, err = multiaddr.NewMultiaddr(fmt.Sprintf(multiAddrIPTemplate, hostStr, portNum))
		} else {
			// If the host is not an IP address, assume it's a DNS name
			// Validate that the host is a valid DNS name
			if strings.Contains(hostStr, ":") {
				log.Debugf("invalid DNS name in advertise address: %s, error: %v\n", addr, err)
				continue
			}
			maddr, err = multiaddr.NewMultiaddr(fmt.Sprintf("/dns4/%s/tcp/%d", hostStr, portNum))
		}

		if err != nil {
			log.Debugf("invalid advertise address: %s, error: %v\n", addr, err)
			continue
		}

		result = append(result, maddr)
	}

	return result
}

func (s *Node) startStaticPeerConnector(ctx context.Context) {
	if len(s.config.StaticPeers) == 0 {
		s.logger.Infof("[Node] no static peers to connect to - skipping connection attempt")
		return
	}

	go func() {
		logged := false

		delay := 0 * time.Second

		for {
			// Use a ticker with context to handle cancellation during sleep
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				// Timer completed, continue as normal
			case <-ctx.Done():
				// Context was canceled during wait, clean up and return
				if !timer.Stop() {
					<-timer.C
				}

				s.logger.Infof("[Node] shutting down")

				return
			}

			allConnected := s.connectToStaticPeers(ctx, s.config.StaticPeers)

			select {
			case <-ctx.Done():
				return
			default:
			}

			if allConnected {
				if !logged {
					s.logger.Infof("[Node] all static peers connected")
				}

				logged = true
				delay = 30 * time.Second // it is possible that a peer disconnects, so we need to keep checking
			} else {
				s.logger.Infof("[Node] all static peers NOT connected")

				logged = false
				delay = 5 * time.Second
			}
		}
	}()
}

func (s *Node) initGossipSub(ctx context.Context, topicNames []string) error {
	ps, err := pubsub.NewGossipSub(ctx, s.host,
		pubsub.WithMessageSignaturePolicy(pubsub.StrictSign)) // Ensure messages are signed and verified
	if err != nil {
		return err
	}

	topics, shouldReturn, err := subscribeToTopics(topicNames, ps, s)
	if shouldReturn {
		return err
	}

	s.pubSub = ps
	s.topics = topics

	return nil
}

// Start activates the P2P node and begins network operations.
// This method initializes peer discovery, topic subscriptions, and stream handlers.
// It performs several key operations:
// - Launches static peer connector to maintain connections with configured static peers
// - Starts peer discovery in a background goroutine to find and connect to network peers
// - Initializes the GossipSub protocol for pub/sub messaging
// - Sets up stream handlers for direct peer-to-peer communication
//
// Parameters:
// - ctx: Context for controlling the start process and subsequent operations
// - streamHandler: Handler for incoming protocol streams (can be nil if not using direct streams)
// - topicNames: List of topic names to subscribe to for pub/sub messaging
//
// The method is non-blocking for peer discovery but waits for GossipSub initialization to complete.
// Returns an error if any critical component fails to initialize.
func (s *Node) Start(ctx context.Context, streamHandler func(network.Stream), topicNames ...string) error {
	s.logger.Infof("[%s] starting", s.config.ProcessName)

	s.startStaticPeerConnector(ctx)

	go func() {
		if err := s.discoverPeers(ctx, topicNames); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Errorf("[Node] error discovering peers: %v", err)
		}
	}()

	if err := s.initGossipSub(ctx, topicNames); err != nil {
		return err
	}

	if streamHandler != nil {
		s.host.SetStreamHandler(protocol.ID(s.bitcoinProtocolID), streamHandler)
	}

	return nil
}

// subscribeToTopics joins the P2P node to multiple pubsub topics for message distribution.
// This function iterates through the provided topic names and joins each one using the
// libp2p pubsub system. It creates a mapping of topic names to topic objects that can
// be used for publishing and subscribing to messages.
//
// The function is used during node initialization to establish participation in
// various communication channels used by the network, such as block propagation,
// transaction distribution, and control message topics.
//
// Parameters:
//   - topicNames: Array of topic names to join
//   - ps: The pubsub instance to use for joining topics
//   - s: The P2P node instance for logging purposes
//
// Returns:
//   - Map of topic names to topic objects for successful subscriptions
//   - Boolean indicating if an error occurred (true if error, false if success)
//   - Error if any topic join operation fails
func subscribeToTopics(topicNames []string, ps *pubsub.PubSub, s *Node) (map[string]*pubsub.Topic, bool, error) {
	topics := map[string]*pubsub.Topic{}

	var topic *pubsub.Topic

	var err error

	for _, topicName := range topicNames {

		topic, err = ps.Join(topicName)
		if err != nil {
			return nil, true, err
		}

		s.logger.Infof("[Node] joined topic: %s", topicName)

		topics[topicName] = topic
	}

	return topics, false, nil
}

// Stop gracefully shuts down the P2P node and closes all connections.
func (s *Node) Stop(_ context.Context) error {
	s.logger.Infof("[Node] stopping")

	// Close the underlying libp2p host
	if s.host != nil {
		if err := s.host.Close(); err != nil {
			s.logger.Errorf("[Node] error closing host: %v", err)
			return err // Return the error if closing fails
		}

		s.logger.Infof("[Node] host closed")
	}

	return nil
}

// SetTopicHandler sets a message handler for the specified topic.
func (s *Node) SetTopicHandler(ctx context.Context, topicName string, handler Handler) error {
	_, ok := s.handlerByTopic[topicName]
	if ok {
		return fmt.Errorf("[Node][SetTopicHandler] handler already exists for topic: %s", topicName)
	}

	topic := s.topics[topicName]

	if topic == nil {
		return fmt.Errorf("[Node][SetTopicHandler] topic not found: %s", topicName)
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}

	s.handlerByTopic[topicName] = handler

	go func() {
		s.logger.Infof("[Node][SetTopicHandler] starting handler for topic: %s", topicName)

		for {
			select {
			case <-ctx.Done():
				s.logger.Infof("[Node][SetTopicHandler] shutting down")
				return
			default:
				m, err := sub.Next(ctx)
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						s.logger.Errorf("[Node][SetTopicHandler] error getting msg from %s topic: %v", topicName, err)
					}

					continue
				}

				s.logger.Debugf("[Node][SetTopicHandler]: topic: %s - from: %s - message: %s\n", *m.Message.Topic, m.ReceivedFrom.ShortString(), strings.TrimSpace(string(m.Message.Data)))
				handler(ctx, m.Data, m.ReceivedFrom.String())
			}
		}
	}()

	return nil
}

// HostID returns the peer ID of this node.
func (s *Node) HostID() peer.ID {
	return s.host.ID()
}

// GetTopic returns the topic instance for the given topic name.
func (s *Node) GetTopic(topicName string) *pubsub.Topic {
	return s.topics[topicName]
}

// Publish sends a message to all subscribers of the specified topic.
func (s *Node) Publish(ctx context.Context, topicName string, msgBytes []byte) error {
	if len(s.topics) == 0 {
		return fmt.Errorf("[Node][Publish] topics not initialized")
	}

	if _, ok := s.topics[topicName]; !ok {
		return fmt.Errorf("[Node][Publish] topic not found: %s", topicName)
	}

	if err := s.topics[topicName].Publish(ctx, msgBytes); err != nil {
		return fmt.Errorf("[Node][Publish] publish error: %w", err)
	}

	s.logger.Debugf("[Node][Publish] topic: %s - message: %s\n", topicName, strings.TrimSpace(string(msgBytes)))

	// Increment bytesSent using atomic operations
	atomic.AddUint64(&s.bytesSent, uint64(len(msgBytes)))

	// Update lastSend timestamp
	atomic.StoreInt64(&s.lastSend, time.Now().Unix())

	return nil
}

// SendToPeer sends a message to a peer. It will attempt to connect to the peer if not already connected.
func (s *Node) SendToPeer(ctx context.Context, peerID peer.ID, msg []byte) (err error) {
	h2pi := s.host.Peerstore().PeerInfo(peerID)
	s.logger.Infof("[Node][SendToPeer] dialing %s", h2pi.Addrs)

	if err = s.host.Connect(ctx, h2pi); err != nil {
		s.logger.Errorf("[Node][SendToPeer] failed to connect: %+v", err)
		return err
	}

	var st network.Stream

	st, err = s.host.NewStream(
		ctx,
		peerID,
		protocol.ID(s.bitcoinProtocolID),
	)
	if err != nil {
		return err
	}

	defer func() {
		err = st.Close()
		if err != nil {
			s.logger.Errorf("[Node][SendToPeer] error closing stream: %s", err)
		}
	}()

	_, err = st.Write(msg)
	if err != nil {
		return err
	}

	s.logger.Debugf("[Node][SendToPeer] sent %v bytes to %s", strings.TrimSpace(string(msg)), peerID.String())
	// Increment bytesSent using atomic operations
	atomic.AddUint64(&s.bytesSent, uint64(len(msg)))

	// Update lastSend timestamp
	atomic.StoreInt64(&s.lastSend, time.Now().Unix())

	return nil
}

// generatePrivateKey creates a new Ed25519 private key for P2P node identity.
// This function generates a cryptographically secure key pair using the Ed25519 algorithm,
// which provides the node's unique identity in the P2P network.
//
// The generated key serves as the node's cryptographic identity for:
//   - Peer authentication and verification
//   - Message signing and validation
//   - Secure communication establishment
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - Pointer to the generated private key ready for libp2p use
//   - Error if key generation fails
func generatePrivateKey(_ context.Context) (*crypto.PrivKey, error) {
	// Generate a new key pair
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &priv, nil
}

func decodeHexEd25519PrivateKey(hexEncodedPrivateKey string) (*crypto.PrivKey, error) {
	privKeyBytes, err := hex.DecodeString(hexEncodedPrivateKey)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.UnmarshalEd25519PrivateKey(privKeyBytes)
	if err != nil {
		return nil, err
	}

	return &privKey, nil
}

func (s *Node) connectToStaticPeers(ctx context.Context, staticPeers []string) bool {
	i := len(staticPeers)

	for _, peerAddr := range staticPeers {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		// check if peerAddr is valid
		peerMaddr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			s.logger.Errorf("[Node] invalid static peer address: %s", peerAddr)
			continue
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(peerMaddr)
		if err != nil {
			s.logger.Errorf("[Node] failed to get peerInfo from  %s: %v", peerAddr, err)
			continue
		}

		if s.host.Network().Connectedness(peerInfo.ID) == network.Connected {
			i--
			continue
		}

		err = s.host.Connect(ctx, *peerInfo)
		if err != nil {
			s.logger.Debugf("[Node] failed to connect to static peer %s: %v", peerAddr, err)
		} else {
			i--

			s.logger.Infof("[Node] connected to static peer: %s", peerAddr)
		}
	}

	return i == 0
}

func (s *Node) discoverPeers(ctx context.Context, topicNames []string) error {
	var (
		kademliaDHT *dht.IpfsDHT
		err         error
	)

	if s.config.UsePrivateDHT {
		kademliaDHT, err = s.initPrivateDHT(ctx, s.host)
	} else {
		kademliaDHT, err = s.initDHT(ctx, s.host)
	}

	if err != nil {
		return fmt.Errorf(errorCreatingDhtMessage, err)
	}

	if kademliaDHT == nil {
		return nil
	}

	routingDiscovery := dRouting.NewRoutingDiscovery(kademliaDHT)

	if s.config.Advertise {
		for _, topicName := range topicNames {
			s.logger.Infof("[Node] advertising topic: %s", topicName)
			dUtil.Advertise(ctx, routingDiscovery, topicName)
		}
	}

	// Log peer store info
	peerCount := len(s.host.Peerstore().Peers())
	s.logger.Debugf("[Node] %d peers in peerstore", peerCount)

	// Use simultaneous connect for hole punching
	ctx = network.WithSimultaneousConnect(ctx, true, "hole punching")
	peerAddrErrorMap := sync.Map{}

	// Look for others who have announced and attempt to connect to them
	for {
		select {
		case <-ctx.Done():
			// Exit immediately if context is done
			s.logger.Infof("[Node] shutting down")
			return nil
		default:
			// Create a copy of the map to avoid concurrent modifications
			peerAddrMap := sync.Map{}

			eg := errgroup.Group{}

			start := time.Now()

			// Start all peer finding goroutines
			for _, topicName := range topicNames {
				// We need to create a copy of the topic name for each goroutine
				// to avoid data races on the loop variable
				topicNameCopy := topicName

				eg.Go(func() error {
					return s.findPeers(ctx, topicNameCopy, routingDiscovery, &peerAddrMap, &peerAddrErrorMap)
				})
			}

			if err := eg.Wait(); err != nil {
				return err
			}

			duration := time.Since(start)
			if duration > 0 { // Avoid logging negative durations due to clock skew
				s.logger.Debugf("[Node] Completed discovery process in %v", duration)
			}

			// Using a timer with context to handle cancellation during sleep
			sleepTimer := time.NewTimer(5 * time.Second)
			select {
			case <-sleepTimer.C:
				// Timer completed normally, continue the loop
			case <-ctx.Done():
				// Context was canceled, clean up and return
				if !sleepTimer.Stop() {
					select {
					case <-sleepTimer.C:
					default:
					}
				}

				return ctx.Err()
			}
		}
	}
}

func (s *Node) findPeers(ctx context.Context, topicName string, routingDiscovery *dRouting.RoutingDiscovery, peerAddrMap *sync.Map, peerAddrErrorMap *sync.Map) error {
	// Find peers subscribed to the topic
	addrChan, err := routingDiscovery.FindPeers(ctx, topicName)
	if err != nil {
		s.logger.Errorf("[Node] error finding peers: %+v", err)

		return err
	}

	wg := &sync.WaitGroup{}

	// Process each peer address discovered
	for addr := range addrChan {
		// Check if context is done before processing each peer
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip peers we shouldn't connect to
		if s.shouldSkipPeer(addr, peerAddrErrorMap) {
			continue
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			s.attemptConnection(ctx, addr, peerAddrMap, peerAddrErrorMap)
		}()
	}

	wg.Wait()

	return nil
}

// shouldSkipPeer determines if a peer should be skipped based on filtering criteria
func (s *Node) shouldSkipPeer(addr peer.AddrInfo, peerAddrErrorMap *sync.Map) bool {
	// Skip self connection
	if addr.ID == s.host.ID() {
		return true
	}

	// Skip already connected peers
	if s.host.Network().Connectedness(addr.ID) == network.Connected {
		return true
	}

	// Skip peers with no addresses
	if len(addr.Addrs) == 0 {
		return true
	}

	// Skip based on previous errors if optimizing retries
	if s.config.OptimiseRetries {
		return s.shouldSkipBasedOnErrors(addr, peerAddrErrorMap)
	}

	return false
}

// shouldSkipBasedOnErrors determines if a peer should be skipped based on previous errors
func (s *Node) shouldSkipBasedOnErrors(addr peer.AddrInfo, peerAddrErrorMap *sync.Map) bool {
	peerConnectionErrorString, ok := peerAddrErrorMap.Load(addr.ID.String())
	if !ok {
		return false
	}

	errorStr := peerConnectionErrorString.(string)

	// Check for "no good addresses" error
	if strings.Contains(errorStr, "no good addresses") {
		return s.shouldSkipNoGoodAddresses(addr)
	}

	// Check for "peer id mismatch" error
	if strings.Contains(errorStr, "peer id mismatch") {
		// "peer id mismatch" is where the node has started using a new private key
		// No point trying to connect to it
		return true
	}

	return false
}

// shouldSkipNoGoodAddresses determines if a peer with "no good addresses" error should be skipped
func (s *Node) shouldSkipNoGoodAddresses(addr peer.AddrInfo) bool {
	numAddresses := len(addr.Addrs)

	switch numAddresses {
	case 0:
		// peer has no addresses, no point trying to connect to it
		return true
	case 1:
		address := addr.Addrs[0].String()
		if strings.Contains(address, "127.0.0.1") {
			// Peer has a single localhost address, and it failed on first attempt
			// You aren't allowed to dial 'yourself' and there are no other addresses available
			return true
		}
	}

	return false
}

// attemptConnection tries to connect to a peer if it hasn't been attempted already
func (s *Node) attemptConnection(ctx context.Context, peerAddr peer.AddrInfo, peerAddrMap *sync.Map, peerAddrErrorMap *sync.Map) {
	if _, ok := peerAddrMap.Load(peerAddr.ID.String()); ok {
		return
	}

	peerAddrMap.Store(peerAddr.ID.String(), true)

	err := s.host.Connect(ctx, peerAddr)
	if err != nil {
		peerAddrErrorMap.Store(peerAddr.ID.String(), true)
		s.logger.Debugf("[Node][%s] Failed to connect: %v", peerAddr.String(), err)
	} else {
		s.logger.Infof("[Node][%s] Connected in %s", peerAddr.String(), time.Since(s.startTime))
	}
}

func (s *Node) initDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	var options []dht.Option

	options = append(options, dht.Mode(dht.ModeAutoServer))

	kademliaDHT, err := dht.New(ctx, h, options...)
	if err != nil {
		return nil, fmt.Errorf(errorCreatingDhtMessage, err)
	} else if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("[Node] error bootstrapping DHT: %w", err)
	}

	var wg sync.WaitGroup

	// Create a context with timeout to ensure bootstrap connections don't hang
	connectCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Create a synchronization channel for handling connection errors
	errorChan := make(chan error, len(dht.DefaultBootstrapPeers))

	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)

		wg.Add(1)

		go func(pi *peer.AddrInfo) {
			defer wg.Done()

			if err := h.Connect(connectCtx, *pi); err != nil {
				errorChan <- err
			}
		}(peerinfo)
	}

	// Launch a separate goroutine to collect and log errors
	var wgLogging sync.WaitGroup

	wgLogging.Add(1)

	// Create a done channel to signal when to stop receiving from errorChan
	doneChan := make(chan struct{})

	go func() {
		defer wgLogging.Done()

		for {
			select {
			case err, ok := <-errorChan:
				if !ok {
					// Channel closed, exit
					return
				}
				// Check context before logging
				select {
				case <-ctx.Done():
					// Context canceled, stop logging
					return
				default:
					s.logger.Debugf("DHT Bootstrap warning: %v", err)
				}
			case <-ctx.Done():
				// Context canceled, stop logging
				return
			case <-doneChan:
				// Signal to stop, exit
				return
			}
		}
	}()

	// Wait for all connection attempts to complete
	wg.Wait()

	// Signal the logging goroutine to exit and close the error channel
	close(doneChan)
	close(errorChan)

	// Wait for logging to complete
	wgLogging.Wait()

	return kademliaDHT, nil
}

func (s *Node) initPrivateDHT(ctx context.Context, host host.Host) (*dht.IpfsDHT, error) {
	bootstrapAddresses := s.config.BootstrapAddresses
	s.logger.Infof("[Node] bootstrapAddresses: %v", bootstrapAddresses)

	if len(bootstrapAddresses) == 0 {
		return nil, fmt.Errorf("[Node] bootstrapAddresses not set in config")
	}

	// Ensure the DHT protocol ID is set in the config
	dhtProtocolIDStr := s.config.DHTProtocolID
	if dhtProtocolIDStr == "" {
		return nil, errors.New("[Node] error getting p2p_dht_protocol_id")
	}

	dhtProtocolID := protocol.ID(dhtProtocolIDStr)

	// Track if we successfully connected to at least one bootstrap address
	connectedToBootstrap := false

	for _, ba := range bootstrapAddresses {
		bootstrapAddr, err := multiaddr.NewMultiaddr(ba)
		if err != nil {
			s.logger.Warnf("[Node] failed to create bootstrap multiaddress %s: %v", ba, err)

			continue // Try the next bootstrap address
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(bootstrapAddr)
		if err != nil {
			s.logger.Warnf("[Node] failed to get peerInfo from %s: %v", ba, err)

			continue // Try the next bootstrap address
		}

		// get the IP from the multiaddress
		ip, err := getIPFromMultiaddr(peerInfo.Addrs[0])
		if err != nil {
			s.logger.Warnf("[Node] failed to get IP from multiaddress %s: %v", ba, err)
			s.logger.Warnf("peerInfo: %+v\n", peerInfo)
		}

		s.logger.Infof("[Node] bootstrap address %s has IP %s", ba, ip)

		err = host.Connect(ctx, *peerInfo)
		if err != nil {
			s.logger.Warnf("[Node] failed to connect to bootstrap address %s: %v", ba, err)

			continue // Try the next bootstrap address
		}

		// Successfully connected to this bootstrap address
		connectedToBootstrap = true

		s.logger.Infof("[Node] successfully connected to bootstrap address %s", ba)
	}

	// Only return an error if we couldn't connect to any bootstrap addresses
	if !connectedToBootstrap {
		return nil, fmt.Errorf("[Node] failed to connect to any bootstrap addresses")
	}

	var options []dht.Option
	options = append(options, dht.ProtocolPrefix(dhtProtocolID))
	options = append(options, dht.Mode(dht.ModeAuto))

	kademliaDHT, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, fmt.Errorf(errorCreatingDhtMessage, err)
	}

	err = kademliaDHT.Bootstrap(ctx)
	if err != nil {
		return nil, fmt.Errorf("[Node] error bootstrapping DHT: %w", err)
	}

	return kademliaDHT, nil
}

// LastSend returns the timestamp of the last message sent.
func (s *Node) LastSend() time.Time {
	return time.Unix(atomic.LoadInt64(&s.lastSend), 0)
}

// LastRecv returns the timestamp of the last message received.
func (s *Node) LastRecv() time.Time {
	return time.Unix(atomic.LoadInt64(&s.lastRecv), 0)
}

// BytesSent returns the total number of bytes sent by this node.
func (s *Node) BytesSent() uint64 {
	return atomic.LoadUint64(&s.bytesSent)
}

// BytesReceived returns the total number of bytes received by this node.
func (s *Node) BytesReceived() uint64 {
	return atomic.LoadUint64(&s.bytesReceived)
}

// PeerInfo contains information about a connected peer.
type PeerInfo struct {
	ID            peer.ID
	Addrs         []multiaddr.Multiaddr
	CurrentHeight int32
	ConnTime      *time.Time // Connection time (nil if not connected)
}

// ConnectedPeers returns information about all connected peers.
func (s *Node) ConnectedPeers() []PeerInfo {
	// Get all connected peers from the network
	peerIDs := s.host.Network().Peerstore().Peers()

	// Create a slice with zero initial length but with capacity for all peers
	peers := make([]PeerInfo, 0, len(peerIDs))

	// Add each peer to the slice
	for _, peerID := range peerIDs {
		var height int32
		if h, ok := s.peerHeights.Load(peerID); ok {
			height = h.(int32)
		}

		var connTime *time.Time
		if ct, ok := s.peerConnTimes.Load(peerID); ok {
			t := ct.(time.Time)
			connTime = &t
		}

		peers = append(peers, PeerInfo{
			ID:            peerID,
			Addrs:         s.host.Network().Peerstore().PeerInfo(peerID).Addrs,
			CurrentHeight: height,
			ConnTime:      connTime,
		})
	}

	s.logger.Debugf("[Node] %d peers in peerstore\n", len(peers))

	return peers
}

// CurrentlyConnectedPeers returns information about currently connected peers.
func (s *Node) CurrentlyConnectedPeers() []PeerInfo {
	// Get all connected peers from the network
	peerIDs := s.host.Network().Peers()

	// Create a slice with zero initial length but with capacity for all peers
	peers := make([]PeerInfo, 0, len(peerIDs))

	// Add each peer to the slice
	for _, peerID := range peerIDs {
		var height int32
		if h, ok := s.peerHeights.Load(peerID); ok {
			height = h.(int32)
		}

		var connTime *time.Time
		if ct, ok := s.peerConnTimes.Load(peerID); ok {
			t := ct.(time.Time)
			connTime = &t
		}

		peers = append(peers, PeerInfo{
			ID:            peerID,
			Addrs:         s.host.Network().Peerstore().PeerInfo(peerID).Addrs,
			CurrentHeight: height,
			ConnTime:      connTime,
		})
	}

	return peers
}

// DisconnectPeer disconnects from the specified peer.
func (s *Node) DisconnectPeer(_ context.Context, peerID peer.ID) error {
	// Close all connections to this peer to ensure disconnection events are triggered
	conns := s.host.Network().ConnsToPeer(peerID)
	for _, conn := range conns {
		err := conn.Close()
		if err != nil {
			s.logger.Debugf("[Node] Error closing connection to %s: %v", peerID.String(), err)
		}
	}

	// Clean up connection time immediately as a fallback
	s.peerConnTimes.Delete(peerID)

	return s.host.Network().ClosePeer(peerID)
}

// UpdatePeerHeight updates the blockchain height for a specific peer.
func (s *Node) UpdatePeerHeight(peerID peer.ID, height int32) {
	s.logger.Debugf("[Node] UpdatePeerHeight: %s %d\n", peerID.String(), height)
	s.peerHeights.Store(peerID, height)
}

// TODO: remove - helper function for extracting IP from multiaddr

func getIPFromMultiaddr(addr multiaddr.Multiaddr) (string, error) {
	// First try to get DNS component
	if value, err := addr.ValueForProtocol(multiaddr.P_DNS4); err == nil {
		return value, nil
	}

	if value, err := addr.ValueForProtocol(multiaddr.P_DNS6); err == nil {
		return value, nil
	}

	// If no DNS, try IP
	if value, err := addr.ValueForProtocol(multiaddr.P_IP4); err == nil {
		return value, nil
	}

	if value, err := addr.ValueForProtocol(multiaddr.P_IP6); err == nil {
		return value, nil
	}

	return "", fmt.Errorf("no IP or DNS component found in multiaddr")
}

// GetProcessName returns the name of the current process.
func (s *Node) GetProcessName() string {
	return s.config.ProcessName
}

// UpdateBytesReceived updates the count of bytes received.
func (s *Node) UpdateBytesReceived(bytesCount uint64) {
	atomic.AddUint64(&s.bytesReceived, bytesCount)
}

// UpdateLastReceived updates the timestamp of the last received message.
func (s *Node) UpdateLastReceived() {
	atomic.StoreInt64(&s.lastRecv, time.Now().Unix())
}

// GetPeerIPs returns the IP addresses for a specific peer.
func (s *Node) GetPeerIPs(peerID peer.ID) []string {
	addrs := s.host.Network().Peerstore().PeerInfo(peerID).Addrs
	ips := make([]string, 0, len(addrs))

	for _, addr := range addrs {
		ip := extractIPFromMultiaddr(addr)
		if ip != "" {
			ips = append(ips, ip)
		}
	}

	return ips
}

// SetPeerConnectedCallback sets a callback function to be called when a new peer connects
func (s *Node) SetPeerConnectedCallback(callback func(context.Context, peer.ID)) {
	s.callbackMutex.Lock()
	defer s.callbackMutex.Unlock()
	s.onPeerConnected = callback
}

// callPeerConnected safely calls the peer connected callback if it exists
func (s *Node) callPeerConnected(ctx context.Context, peerID peer.ID) {
	s.callbackMutex.RLock()
	callback := s.onPeerConnected
	s.callbackMutex.RUnlock()

	if callback != nil {
		go callback(ctx, peerID)
	}
}

func extractIPFromMultiaddr(multiaddr multiaddr.Multiaddr) string {
	str := multiaddr.String()

	parts := strings.Split(str, "/")
	for i, part := range parts {
		if part == "ip4" || part == "ip6" {
			if i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}

	return ""
}

// isPrivateIP checks if an IP address is private according to RFC 1918 and RFC 3927
func isPrivateIP(addr multiaddr.Multiaddr) bool {
	ipStr := extractIPFromMultiaddr(addr)
	if ipStr == "" {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() == nil {
		return false
	}

	// Define private IPv4 ranges according to RFC 1918 and RFC 3927
	// These are standard private network ranges and are safe to use
	privateRanges := []*net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},     // RFC 1918: Class A private network
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},  // RFC 1918: Class B private network
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)}, // RFC 1918: Class C private network
		{IP: net.ParseIP("127.0.0.0"), Mask: net.CIDRMask(8, 32)},    // RFC 3927: Loopback addresses
	}

	// Check if the IP falls into any of the private ranges
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}

	return false
}
