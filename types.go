package p2p

import (
	"context"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	errorCreatingDhtMessage = "[Node] error creating DHT"
	multiAddrIPTemplate     = "/ip4/%s/tcp/%d"
)

// Node implements the NodeI interface and provides the core functionality
// for peer-to-peer networking in BSV blockchain applications using libp2p.
// It manages peer connections, topic subscriptions, message routing, and network discovery.
//
// The Node encapsulates several critical components:
// - libp2p host for network transport and connection management
// - PubSub for topic-based message distribution
// - Peer height tracking for blockchain synchronization
// - Bandwidth and activity metrics for monitoring
//
// Thread safety is maintained for all concurrent operations across multiple goroutines.
type Node struct {
	config            Config                         // Configuration parameters for the node
	host              host.Host                      // libp2p host for network communication
	pubSub            *pubsub.PubSub                 // Publish-subscribe system for topic-based messaging
	topics            map[string]*pubsub.Topic       // Map of topic names to topic objects
	logger            *logrus.Logger                 // Logger for P2P operations
	bitcoinProtocolID string                         // Protocol identifier for Bitcoin-specific streams
	handlerByTopic    map[string]Handler             // Map of topic handlers for message processing
	startTime         time.Time                      // Time when the node was started
	onPeerConnected   func(context.Context, peer.ID) // Callback for peer connection events
	callbackMutex     sync.RWMutex                   // Mutex for thread-safe callback access

	// IMPORTANT: The following variables must only be used atomically.
	bytesReceived uint64   // Counter for bytes received over the network
	bytesSent     uint64   // Counter for bytes sent over the network
	lastRecv      int64    // Timestamp of last message received
	lastSend      int64    // Timestamp of last message sent
	peerHeights   sync.Map // Thread-safe map tracking peer blockchain heights
	peerConnTimes sync.Map // Thread-safe map tracking peer connection times (peer.ID -> time.Time)
}

// Handler defines the function signature for topic message handlers.
// Each topic in the P2P network can have a dedicated handler that processes incoming messages.
//
// Parameters:
//   - ctx: Context for the handler execution, allowing for cancellation and timeouts
//   - msg: Raw message bytes received from the network
//   - from: Identifier of the peer that sent the message
//
// Handlers should process messages efficiently as they may be called frequently
// in high-traffic scenarios. Any long-running operations should be delegated to separate goroutines.
type Handler func(ctx context.Context, msg []byte, from string)

// Config defines the configuration parameters for a P2P node.
// It encapsulates all settings needed to establish and maintain
// a functional peer-to-peer network presence.
type Config struct {
	ProcessName        string   // Identifier for this node in logs and metrics
	BootstrapAddresses []string // Initial peer addresses to connect to for network discovery
	ListenAddresses    []string // Network addresses to listen on for incoming connections
	AdvertiseAddresses []string // Addresses to advertise to other peers (may differ from listen addresses)
	Port               int      // Port number for P2P communication
	DHTProtocolID      string   // Protocol ID for the DHT used by this node
	PrivateKey         string   // Node's private key for secure communication
	SharedKey          string   // Shared key for private network communication
	UsePrivateDHT      bool     // Whether to use a private DHT instead of the public IPFS DHT
	OptimiseRetries    bool     // Whether to optimize connection retry behavior
	Advertise          bool     // Whether to advertise this node's presence on the network
	StaticPeers        []string // List of peer addresses to always attempt to connect to
}
