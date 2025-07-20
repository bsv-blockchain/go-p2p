package p2p

import (
	"context"
	"github.com/libp2p/go-libp2p/core/network"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
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
	Start(ctx context.Context, streamHandler func(network.Stream), topicNames ...string) error
	Stop(ctx context.Context) error

	// Topic-related methods
	SetTopicHandler(ctx context.Context, topicName string, handler Handler) error
	GetTopic(topicName string) *pubsub.Topic
	Publish(ctx context.Context, topicName string, msgBytes []byte) error

	// Peer management methods
	HostID() peer.ID
	ConnectedPeers() []PeerInfo
	CurrentlyConnectedPeers() []PeerInfo
	DisconnectPeer(ctx context.Context, peerID peer.ID) error
	SendToPeer(ctx context.Context, pid peer.ID, msg []byte) error
	SetPeerConnectedCallback(callback func(context.Context, peer.ID))

	UpdatePeerHeight(peerID peer.ID, height int32)

	// Stats methods
	LastSend() time.Time
	LastRecv() time.Time
	BytesSent() uint64
	BytesReceived() uint64

	// Additional accessors needed by Server
	GetProcessName() string
	UpdateBytesReceived(bytesCount uint64)
	UpdateLastReceived()

	GetPeerIPs(peerID peer.ID) []string
}
