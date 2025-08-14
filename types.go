package p2p

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	errorCreatingDhtMessage = "[Node] error creating DHT: %w"
	multiAddrIPTemplate     = "/ip4/%s/tcp/%d"

	// ListenModeFull defines the node should operate in full mode
	ListenModeFull = "full"
	// ListenModeListenOnly defines the node should operate in listen-only mode
	ListenModeListenOnly = "listen_only"
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
	logger            Logger                         // Logger for P2P operations
	bitcoinProtocolID string                         // Protocol identifier for Bitcoin-specific streams
	handlerByTopic    map[string]Handler             // Map of topic handlers for message processing
	startTime         time.Time                      // Time when the node was started
	onPeerConnected   func(context.Context, peer.ID) // Callback for peer connection events
	callbackMutex     sync.RWMutex                   // Mutex for thread-safe callback access
	peerCache         *PeerCache                     // Peer cache for persistence across restarts

	// IMPORTANT: The following variables must only be used atomically.
	bytesReceived       uint64   // Counter for bytes received over the network
	bytesSent           uint64   // Counter for bytes sent over the network
	lastRecv            int64    // Timestamp of last message received
	lastSend            int64    // Timestamp of last message sent
	peerHeights         sync.Map // Thread-safe map tracking peer blockchain heights
	peerStartingHeights sync.Map // Thread-safe map for initial peer heights at connection time
	peerConnTimes       sync.Map // Thread-safe map tracking peer connection times (peer.ID -> time.Time)
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
	ProcessName        string        // Identifier for this node in logs and metrics
	BootstrapAddresses []string      // Initial peer addresses to connect to for network discovery
	ListenAddresses    []string      // Network addresses to listen on for incoming connections
	AdvertiseAddresses []string      // Addresses to advertise to other peers (may differ from listen addresses)
	Port               int           // Port number for P2P communication
	DHTProtocolID      string        // Protocol ID for the DHT used by this node
	PrivateKey         string        // Node's private key for secure communication
	SharedKey          string        // Shared key for private network communication
	UsePrivateDHT      bool          // Whether to use a private DHT instead of the public IPFS DHT
	OptimiseRetries    bool          // Whether to optimize connection retry behavior
	Advertise          bool          // Whether to advertise this node's presence on the network
	StaticPeers        []string      // List of peer addresses to always attempt to connect to
	ListenMode         string        // Mode of operation: "full" for active participation, "listen_only" for passive listening
	EnableNATService   bool          // Whether to enable NAT service for peer connectivity
	EnableHolePunching bool          // Whether to enable NAT hole punching
	EnableRelay        bool          // Whether to enable relay functionality
	EnableNATPortMap   bool          // Whether to enable NAT port mapping
	EnableAutoNATv2    bool          // Whether to enable AutoNAT v2 for better address discovery
	ForceReachability  string        // Force reachability: "public", "private", or "" (auto-detect)
	EnableRelayService bool          // Whether to act as a relay for other nodes (requires EnableRelay)
	// Peer persistence configuration
	EnablePeerCache    bool          // Whether to enable peer caching for persistence across restarts
	PeerCacheFile      string        // Path to the peer cache file (default: "~/.p2p/peers.json")
	MaxCachedPeers     int           // Maximum number of peers to cache (default: 100)
	PeerCacheTTL       time.Duration // How long to keep cached peers (default: 30 days)
	// Connection management configuration
	EnableConnManager  bool          // Whether to enable connection manager with high/low water marks
	ConnLowWater       int           // Minimum number of connections to maintain (default: 200)
	ConnHighWater      int           // Maximum number of connections before pruning (default: 400)
	ConnGracePeriod    time.Duration // Grace period before pruning new connections (default: 60s)
	EnableConnGater    bool          // Whether to enable connection gater for fine-grained control
	MaxConnsPerPeer    int           // Maximum connections allowed per peer (default: 3)
}

// Logger defines the interface for logging within the P2P node.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}
