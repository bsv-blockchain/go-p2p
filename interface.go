package p2p

import (
	"context"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// NodeI defines the interface for P2P node functionality.
// This interface abstracts the concrete implementation to allow for better testability.
// It provides methods for managing core peer-to-peer networking operations, including
// node lifecycle management, topic subscription, peer discovery, and message propagation.
//
// The interface is designed to be robust for both standard network operation and
// specialized testing scenarios. It encapsulates all libp2p functionality behind
// a clean API that integrates with BSV blockchain applications.
type NodeI interface {
	// Core lifecycle methods

	// Start initializes the P2P node, setting up the network and subscribing to specified topics.
	Start(ctx context.Context, streamHandler func(network.Stream), topicNames ...string) error

	// Stop gracefully shuts down the P2P node, closing all connections and cleaning up resources.
	Stop(ctx context.Context) error

	// Topic-related methods

	// SetTopicHandler registers a handler for a specific topic, allowing the node to process messages
	SetTopicHandler(ctx context.Context, topicName string, handler Handler) error

	// GetTopic retrieves a pubsub topic by its name, allowing for message publication and subscription.
	GetTopic(topicName string) *pubsub.Topic

	// Publish sends a message to a specified topic, broadcasting it to all subscribers.
	Publish(ctx context.Context, topicName string, msgBytes []byte) error

	// Peer management methods

	// HostID returns the unique identifier of the P2P node.
	HostID() peer.ID

	// ConnectedPeers returns a list of all currently connected peers.
	ConnectedPeers() []PeerInfo

	// CurrentlyConnectedPeers returns a list of currently connected peers with their information.
	CurrentlyConnectedPeers() []PeerInfo

	// ConnectToPeer connects to a peer identified by its ID, establishing a P2P connection.
	ConnectToPeer(ctx context.Context, peer string) error

	// DisconnectPeer disconnects a peer from the P2P network, removing it from the list of connected peers.
	DisconnectPeer(ctx context.Context, peerID peer.ID) error

	// SendToPeer sends a raw byte message to a specific peer, allowing for direct communication.
	SendToPeer(ctx context.Context, pid peer.ID, msg []byte) error

	// SetPeerConnectedCallback sets a callback function that is invoked when a peer connects to the node.
	SetPeerConnectedCallback(callback func(context.Context, peer.ID))

	// UpdatePeerHeight updates the height of a peer, which is used to track the blockchain state of that peer.
	UpdatePeerHeight(peerID peer.ID, height int32)

	// GetPeerStartingHeight retrieves the initial height of a peer at the time of connection.
	GetPeerStartingHeight(peerID peer.ID) (int32, bool)

	// SetPeerStartingHeight sets the initial height of a peer at the time of connection.
	SetPeerStartingHeight(peerID peer.ID, height int32)

	// Stats methods

	// LastSend returns the timestamp of the last message sent by the node.
	LastSend() time.Time

	// LastRecv returns the timestamp of the last message received by the node.
	LastRecv() time.Time

	// BytesSent returns the total number of bytes sent by the node.
	BytesSent() uint64

	// BytesReceived returns the total number of bytes received by the node.
	BytesReceived() uint64

	// Additional accessors needed by Server

	// GetProcessName returns the name of the process running the P2P node.
	GetProcessName() string

	// UpdateBytesReceived updates the total number of bytes received by the node.
	UpdateBytesReceived(bytesCount uint64)

	// UpdateLastReceived updates the timestamp of the last message received by the node.
	UpdateLastReceived()

	// GetPeerIPs retrieves the IP addresses of a specific peer by its ID.
	GetPeerIPs(peerID peer.ID) []string
}
