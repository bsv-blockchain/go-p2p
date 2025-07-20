package p2p

import (
	"context"
	"github.com/libp2p/go-libp2p/core/network"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewP2PNode(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name     string
		config   Config
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, node *Node)
	}{
		{
			name: "basic node creation",
			config: Config{
				ProcessName:     "test-node",
				ListenAddresses: []string{"127.0.0.1"},
				Port:            0, // Use 0 for random port
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				assert.Equal(t, "test-node", node.config.ProcessName)
				assert.NotEmpty(t, node.host.ID())
			},
		},
		{
			name: "node with private key",
			config: Config{
				ProcessName:     "key-node",
				ListenAddresses: []string{"127.0.0.1"},
				Port:            0,
				PrivateKey:      generateValidHexKey(t),
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				// Verify the host ID is deterministic with the provided key
				expectedID := node.host.ID()
				assert.NotEmpty(t, expectedID)
			},
		},
		{
			name: "node with invalid private key",
			config: Config{
				ProcessName:     "bad-key-node",
				ListenAddresses: []string{"127.0.0.1"},
				Port:            0,
				PrivateKey:      "invalid-hex-key",
			},
			wantErr: true,
			errMsg:  "error decoding private key",
		},
		{
			name: "node with advertise addresses",
			config: Config{
				ProcessName:        "advertise-node",
				ListenAddresses:    []string{"0.0.0.0"},
				AdvertiseAddresses: []string{"1.2.3.4", "example.com"},
				Port:               0,
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				// Host should be created successfully with advertise addresses
			},
		},
		{
			name: "private network node",
			config: Config{
				ProcessName:     "private-node",
				ListenAddresses: []string{"127.0.0.1"},
				Port:            0,
				UsePrivateDHT:   true,
				SharedKey:       "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				assert.True(t, node.config.UsePrivateDHT)
			},
		},
		{
			name: "private network with invalid shared key",
			config: Config{
				ProcessName:     "bad-private-node",
				ListenAddresses: []string{"127.0.0.1"},
				Port:            0,
				UsePrivateDHT:   true,
				SharedKey:       "invalid-key",
			},
			wantErr: true,
			errMsg:  "error setting up private network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			node, err := NewP2PNode(ctx, logger, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, node)
			} else {
				require.NoError(t, err)
				require.NotNil(t, node)

				// Common validations
				assert.Equal(t, tt.config, node.config)
				assert.Equal(t, logger, node.logger)
				assert.NotNil(t, node.handlerByTopic)
				assert.NotNil(t, node.host)
				assert.Equal(t, "teranode/bitcoin/1.0.0", node.bitcoinProtocolID)

				// Custom validations
				if tt.validate != nil {
					tt.validate(t, node)
				}

				// Cleanup
				node.host.Close()
			}
		})
	}
}

func TestP2PNode_StartStop(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	t.Run("normal start and stop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := Config{
			ProcessName:     "start-stop-test",
			ListenAddresses: []string{"127.0.0.1"},
			Port:            0,
		}

		node, err := NewP2PNode(ctx, logger, config)
		require.NoError(t, err)
		require.NotNil(t, node)

		// Start the node
		err = node.Start(ctx, nil, "test-topic")
		require.NoError(t, err)

		// Verify node is running
		assert.NotNil(t, node.pubSub)
		assert.NotNil(t, node.topics)
		assert.Contains(t, node.topics, "test-topic")

		// Stop the node
		err = node.Stop(ctx)
		require.NoError(t, err)
	})

	t.Run("start with stream handler", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := Config{
			ProcessName:     "stream-handler-test",
			ListenAddresses: []string{"127.0.0.1"},
			Port:            0,
		}

		node, err := NewP2PNode(ctx, logger, config)
		require.NoError(t, err)

		streamHandler := func(stream network.Stream) {
			stream.Close()
		}

		err = node.Start(ctx, streamHandler, "test-topic")
		require.NoError(t, err)

		// Verify stream handler was set (we can't easily test it's called without another node)
		// Just ensure no panic and proper setup

		err = node.Stop(ctx)
		assert.NoError(t, err)
	})

	t.Run("multiple topics", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config := Config{
			ProcessName:     "multi-topic-test",
			ListenAddresses: []string{"127.0.0.1"},
			Port:            0,
		}

		node, err := NewP2PNode(ctx, logger, config)
		require.NoError(t, err)

		topics := []string{"topic1", "topic2", "topic3"}
		err = node.Start(ctx, nil, topics...)
		require.NoError(t, err)

		// Verify all topics were created
		for _, topic := range topics {
			assert.Contains(t, node.topics, topic)
			assert.NotNil(t, node.topics[topic])
		}

		err = node.Stop(ctx)
		assert.NoError(t, err)
	})
}

func TestP2PNode_BasicGetters(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := Config{
		ProcessName:     "getter-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	t.Run("HostID", func(t *testing.T) {
		hostID := node.HostID()
		assert.NotEmpty(t, hostID)
		assert.Equal(t, node.host.ID(), hostID)
	})

	t.Run("GetProcessName", func(t *testing.T) {
		name := node.GetProcessName()
		assert.Equal(t, "getter-test", name)
	})

	t.Run("GetTopic before start", func(t *testing.T) {
		topic := node.GetTopic("non-existent")
		assert.Nil(t, topic)
	})

	// Start node to test topic retrieval
	err = node.Start(ctx, nil, "test-topic")
	require.NoError(t, err)

	t.Run("GetTopic after start", func(t *testing.T) {
		topic := node.GetTopic("test-topic")
		assert.NotNil(t, topic)

		nonExistent := node.GetTopic("non-existent")
		assert.Nil(t, nonExistent)
	})
}

func TestP2PNode_Metrics(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := Config{
		ProcessName:     "metrics-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	t.Run("initial metrics", func(t *testing.T) {
		assert.Equal(t, uint64(0), node.BytesSent())
		assert.Equal(t, uint64(0), node.BytesReceived())
		assert.Equal(t, time.Unix(0, 0), node.LastSend())
		assert.Equal(t, time.Unix(0, 0), node.LastRecv())
	})

	t.Run("update metrics", func(t *testing.T) {
		// Update bytes received
		node.UpdateBytesReceived(100)
		assert.Equal(t, uint64(100), node.BytesReceived())

		node.UpdateBytesReceived(50)
		assert.Equal(t, uint64(150), node.BytesReceived())

		// Update last received
		beforeUpdate := time.Now()
		node.UpdateLastReceived()
		afterUpdate := time.Now()

		lastRecv := node.LastRecv()
		assert.True(t, lastRecv.After(beforeUpdate) || lastRecv.Equal(beforeUpdate))
		assert.True(t, lastRecv.Before(afterUpdate) || lastRecv.Equal(afterUpdate))
	})
}

func TestP2PNode_PeerManagement(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := Config{
		ProcessName:     "peer-mgmt-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	// Test peer ID parsing
	peerIDStr := "12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F"
	peerID, err := peer.Decode(peerIDStr)
	require.NoError(t, err)

	t.Run("UpdatePeerHeight", func(t *testing.T) {
		node.UpdatePeerHeight(peerID, 12345)

		// Verify it was stored
		height, ok := node.peerHeights.Load(peerID)
		assert.True(t, ok)
		assert.Equal(t, int32(12345), height.(int32))

		// Update to new height
		node.UpdatePeerHeight(peerID, 12346)
		height, ok = node.peerHeights.Load(peerID)
		assert.True(t, ok)
		assert.Equal(t, int32(12346), height.(int32))
	})

	t.Run("ConnectedPeers empty", func(t *testing.T) {
		peers := node.ConnectedPeers()
		assert.NotNil(t, peers)
		// Will include self
		assert.GreaterOrEqual(t, len(peers), 1)
	})

	t.Run("CurrentlyConnectedPeers empty", func(t *testing.T) {
		peers := node.CurrentlyConnectedPeers()
		assert.NotNil(t, peers)
		assert.Empty(t, peers) // No actual connections
	})

	t.Run("GetPeerIPs", func(t *testing.T) {
		ips := node.GetPeerIPs(peerID)
		assert.NotNil(t, ips)
		// Won't have IPs for non-connected peer
		assert.Empty(t, ips)
	})

	t.Run("DisconnectPeer non-connected", func(t *testing.T) {
		err := node.DisconnectPeer(ctx, peerID)
		// Should not error even if peer not connected
		assert.NoError(t, err)
	})
}

func TestP2PNode_Callbacks(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := Config{
		ProcessName:     "callback-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	t.Run("SetPeerConnectedCallback", func(t *testing.T) {
		callbackCalled := false
		var receivedPeerID peer.ID

		callback := func(_ context.Context, peerID peer.ID) {
			callbackCalled = true
			receivedPeerID = peerID
		}

		node.SetPeerConnectedCallback(callback)

		// Simulate a peer connection by calling the internal method
		testPeerID, _ := peer.Decode("12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F")
		node.callPeerConnected(ctx, testPeerID)

		// Give the goroutine time to execute
		time.Sleep(10 * time.Millisecond)

		assert.True(t, callbackCalled)
		assert.Equal(t, testPeerID, receivedPeerID)
	})

	t.Run("callback not set", func(t *testing.T) {
		// Create new node without callback
		node2, err := NewP2PNode(ctx, logger, config)
		require.NoError(t, err)
		defer node2.host.Close()

		// Should not panic when calling without callback
		testPeerID, _ := peer.Decode("12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F")
		assert.NotPanics(t, func() {
			node2.callPeerConnected(ctx, testPeerID)
		})
	})
}

func TestP2PNode_ThreadSafety(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := Config{
		ProcessName:     "thread-safety-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	// Test concurrent access to callbacks
	t.Run("concurrent callback operations", func(_ *testing.T) {
		done := make(chan bool)

		// Writer goroutine
		go func() {
			for i := 0; i < 100; i++ {
				callback := func(_ context.Context, _ peer.ID) {}
				node.SetPeerConnectedCallback(callback)
			}
			done <- true
		}()

		// Reader goroutine
		go func() {
			for i := 0; i < 100; i++ {
				testPeerID, _ := peer.Decode("12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F")
				node.callPeerConnected(ctx, testPeerID)
			}
			done <- true
		}()

		// Wait for both to complete
		<-done
		<-done
	})
}
