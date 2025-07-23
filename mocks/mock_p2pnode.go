// Package mocks provides mock implementations for P2P node interfaces used in testing.
package mocks

import (
	"context"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/mock"
)

// PeerInfo represents peer information for mocking
type PeerInfo struct {
	ID            peer.ID
	Addrs         []string
	CurrentHeight int32
	ConnTime      *time.Time
}

// MockP2PNode is a mock implementation of P2PNodeI interface
type MockP2PNode struct {
	mock.Mock
}

// Start mocks the Start method
func (m *MockP2PNode) Start(ctx context.Context, streamHandler func(network.Stream), topicNames ...string) error {
	args := m.Called(ctx, streamHandler, topicNames)
	return args.Error(0)
}

// Stop mocks the Stop method
func (m *MockP2PNode) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// SetTopicHandler mocks the SetTopicHandler method
func (m *MockP2PNode) SetTopicHandler(ctx context.Context, topicName string, handler func(context.Context, []byte, string)) error {
	args := m.Called(ctx, topicName, handler)
	return args.Error(0)
}

// GetTopic mocks the GetTopic method
func (m *MockP2PNode) GetTopic(topicName string) *pubsub.Topic {
	args := m.Called(topicName)
	if topic := args.Get(0); topic != nil {
		return topic.(*pubsub.Topic)
	}
	return nil
}

// Publish mocks the Publish method
func (m *MockP2PNode) Publish(ctx context.Context, topicName string, msgBytes []byte) error {
	args := m.Called(ctx, topicName, msgBytes)
	return args.Error(0)
}

// HostID mocks the HostID method
func (m *MockP2PNode) HostID() peer.ID {
	args := m.Called()
	return args.Get(0).(peer.ID)
}

// ConnectedPeers mocks the ConnectedPeers method
func (m *MockP2PNode) ConnectedPeers() []PeerInfo {
	args := m.Called()
	if peers := args.Get(0); peers != nil {
		return peers.([]PeerInfo)
	}
	return nil
}

// CurrentlyConnectedPeers mocks the CurrentlyConnectedPeers method
func (m *MockP2PNode) CurrentlyConnectedPeers() []PeerInfo {
	args := m.Called()
	if peers := args.Get(0); peers != nil {
		return peers.([]PeerInfo)
	}
	return nil
}

// ConnectToPeer mocks the ConnectToPeer method
func (m *MockP2PNode) ConnectToPeer(ctx context.Context, peer string) error {
	args := m.Called(ctx, peer)
	return args.Error(0)
}

// DisconnectPeer mocks the DisconnectPeer method
func (m *MockP2PNode) DisconnectPeer(ctx context.Context, peerID peer.ID) error {
	args := m.Called(ctx, peerID)
	return args.Error(0)
}

// SendToPeer mocks the SendToPeer method
func (m *MockP2PNode) SendToPeer(ctx context.Context, pid peer.ID, msg []byte) error {
	args := m.Called(ctx, pid, msg)
	return args.Error(0)
}

// SetPeerConnectedCallback mocks the SetPeerConnectedCallback method
func (m *MockP2PNode) SetPeerConnectedCallback(callback func(context.Context, peer.ID)) {
	m.Called(callback)
}

// UpdatePeerHeight mocks the UpdatePeerHeight method
func (m *MockP2PNode) UpdatePeerHeight(peerID peer.ID, height int32) {
	m.Called(peerID, height)
}

// LastSend mocks the LastSend method
func (m *MockP2PNode) LastSend() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// LastRecv mocks the LastRecv method
func (m *MockP2PNode) LastRecv() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// BytesSent mocks the BytesSent method
func (m *MockP2PNode) BytesSent() uint64 {
	args := m.Called()
	return args.Get(0).(uint64)
}

// BytesReceived mocks the BytesReceived method
func (m *MockP2PNode) BytesReceived() uint64 {
	args := m.Called()
	return args.Get(0).(uint64)
}

// GetProcessName mocks the GetProcessName method
func (m *MockP2PNode) GetProcessName() string {
	args := m.Called()
	return args.String(0)
}

// UpdateBytesReceived mocks the UpdateBytesReceived method
func (m *MockP2PNode) UpdateBytesReceived(bytesCount uint64) {
	m.Called(bytesCount)
}

// UpdateLastReceived mocks the UpdateLastReceived method
func (m *MockP2PNode) UpdateLastReceived() {
	m.Called()
}

// GetPeerIPs mocks the GetPeerIPs method
func (m *MockP2PNode) GetPeerIPs(peerID peer.ID) []string {
	args := m.Called(peerID)
	if ips := args.Get(0); ips != nil {
		return ips.([]string)
	}
	return nil
}

// NewMockP2PNode creates a new mock P2P node instance
func NewMockP2PNode() *MockP2PNode {
	return &MockP2PNode{}
}
