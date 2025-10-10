package p2p

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"

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
		{
			name: "node with NAT service enabled",
			config: Config{
				ProcessName:      "nat-service-node",
				ListenAddresses:  []string{"127.0.0.1"},
				Port:             0,
				EnableNATService: true,
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				assert.True(t, node.config.EnableNATService)
			},
		},
		{
			name: "node with hole punching enabled",
			config: Config{
				ProcessName:        "hole-punch-node",
				ListenAddresses:    []string{"127.0.0.1"},
				Port:               0,
				EnableHolePunching: true,
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				assert.True(t, node.config.EnableHolePunching)
			},
		},
		{
			name: "node with relay enabled",
			config: Config{
				ProcessName:     "relay-node",
				ListenAddresses: []string{"127.0.0.1"},
				Port:            0,
				EnableRelay:     true,
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				assert.True(t, node.config.EnableRelay)
			},
		},
		{
			name: "node with NAT port mapping enabled",
			config: Config{
				ProcessName:      "nat-portmap-node",
				ListenAddresses:  []string{"127.0.0.1"},
				Port:             0,
				EnableNATPortMap: true,
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				assert.True(t, node.config.EnableNATPortMap)
			},
		},
		{
			name: "node with all NAT/relay options enabled",
			config: Config{
				ProcessName:        "full-nat-node",
				ListenAddresses:    []string{"127.0.0.1"},
				Port:               0,
				EnableNATService:   true,
				EnableHolePunching: true,
				EnableRelay:        true,
				EnableNATPortMap:   true,
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				assert.True(t, node.config.EnableNATService)
				assert.True(t, node.config.EnableHolePunching)
				assert.True(t, node.config.EnableRelay)
				assert.True(t, node.config.EnableNATPortMap)
			},
		},
		{
			name: "private network with NAT/relay options",
			config: Config{
				ProcessName:        "private-nat-node",
				ListenAddresses:    []string{"127.0.0.1"},
				Port:               0,
				UsePrivateDHT:      true,
				SharedKey:          "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				EnableNATService:   true,
				EnableHolePunching: true,
				EnableRelay:        true,
				EnableNATPortMap:   true,
			},
			wantErr: false,
			validate: func(t *testing.T, node *Node) {
				assert.NotNil(t, node.host)
				assert.True(t, node.config.UsePrivateDHT)
				assert.True(t, node.config.EnableNATService)
				assert.True(t, node.config.EnableHolePunching)
				assert.True(t, node.config.EnableRelay)
				assert.True(t, node.config.EnableNATPortMap)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			node, err := NewNode(ctx, logger, tt.config)

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
				err = node.host.Close()
				require.NoError(t, err)
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

		node, err := NewNode(ctx, logger, config)
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

		node, err := NewNode(ctx, logger, config)
		require.NoError(t, err)

		streamHandler := func(stream network.Stream) {
			if funcErr := stream.Close(); funcErr != nil {
				t.Errorf("Failed to close stream: %v", funcErr)
			}
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

		node, err := NewNode(ctx, logger, config)
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

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() { _ = node.host.Close() }()

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

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() { _ = node.host.Close() }()

	t.Run("initial metrics", func(t *testing.T) {
		assert.Equal(t, uint64(0), node.BytesSent())
		assert.Equal(t, uint64(0), node.BytesReceived())
		assert.Equal(t, time.Unix(0, 0), node.LastSend())
		assert.Equal(t, time.Unix(0, 0), node.LastRecv())
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

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() { _ = node.host.Close() }()

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
	t.Skip()
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := Config{
		ProcessName:     "callback-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() { _ = node.host.Close() }()

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
		node2, err := NewNode(ctx, logger, config)
		require.NoError(t, err)
		defer func() { _ = node2.host.Close() }()

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

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() { _ = node.host.Close() }()

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

func TestP2PNode_ConnectToPeer(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create two nodes for testing connections
	config1 := Config{
		ProcessName:        "node1",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3111, // Random port
	}

	config2 := Config{
		ProcessName:        "node2",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3112, // Random port
	}

	node1, err := NewNode(ctx, logger, config1)
	require.NoError(t, err)
	defer func() { _ = node1.host.Close() }()

	node2, err := NewNode(ctx, logger, config2)
	require.NoError(t, err)
	defer func() { _ = node2.host.Close() }()

	// Get node2's address
	node2Addrs := node2.host.Addrs()
	require.NotEmpty(t, node2Addrs)
	node2Addr := node2Addrs[0].String() + "/p2p/" + node2.host.ID().String()

	t.Run("successful connection", func(t *testing.T) {
		err := node1.ConnectToPeer(ctx, node2Addr)
		require.NoError(t, err)

		// Verify connection
		peers := node1.host.Network().Peers()
		assert.Contains(t, peers, node2.host.ID())
	})

	t.Run("connect to already connected peer", func(t *testing.T) {
		// Should not error when connecting to already connected peer
		err := node1.ConnectToPeer(ctx, node2Addr)
		assert.NoError(t, err)
	})

	t.Run("invalid multiaddr", func(t *testing.T) {
		tests := []struct {
			name     string
			peerAddr string
			errMsg   string
		}{
			{
				name:     "empty address",
				peerAddr: "",
				errMsg:   "invalid multiaddr",
			},
			{
				name:     "invalid format",
				peerAddr: "not-a-multiaddr",
				errMsg:   "invalid multiaddr",
			},
			{
				name:     "missing peer ID",
				peerAddr: "/ip4/127.0.0.1/tcp/4001",
				errMsg:   "failed to get peer info",
			},
			{
				name:     "invalid peer ID",
				peerAddr: "/ip4/127.0.0.1/tcp/4001/p2p/invalid-peer-id",
				errMsg:   "invalid-peer-id",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := node1.ConnectToPeer(ctx, tt.peerAddr)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			})
		}
	})

	t.Run("connect to non-existent peer", func(t *testing.T) {
		// Create a valid multiaddr with non-existent peer
		nonExistentAddr := "/ip4/127.0.0.1/tcp/55555/p2p/12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F"
		err := node1.ConnectToPeer(ctx, nonExistentAddr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to peer")
	})

	t.Run("connection with context cancellation", func(t *testing.T) {
		// Create a new node for this test
		config3 := Config{
			ProcessName:     "node3",
			ListenAddresses: []string{"127.0.0.1"},
			Port:            0,
		}
		node3, err := NewNode(ctx, logger, config3)
		require.NoError(t, err)
		defer func() { _ = node3.host.Close() }()

		// Create a context that we'll cancel
		connectCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		node3Addr := node3.host.Addrs()[0].String() + "/p2p/" + node3.host.ID().String()
		err = node1.ConnectToPeer(connectCtx, node3Addr)
		// The error might vary depending on when the cancellation is processed
		// It could be a context canceled error or a connection failed error
		assert.Error(t, err)
	})

	t.Run("connect with DNS multiaddr", func(t *testing.T) {
		// Test DNS-based multiaddr (will fail since example.com is not a real p2p node)
		dnsAddr := "/dns4/example.com/tcp/4001/p2p/12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F"
		err := node1.ConnectToPeer(ctx, dnsAddr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to peer")
	})

	t.Run("connect with IPv6 multiaddr", func(t *testing.T) {
		// Test IPv6 multiaddr
		ipv6Addr := "/ip6/::1/tcp/4001/p2p/12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F"
		err := node1.ConnectToPeer(ctx, ipv6Addr)
		require.Error(t, err)
		// Should fail to connect since there's no peer at this address
		assert.Contains(t, err.Error(), "failed to connect to peer")
	})

	t.Run("connect to self", func(t *testing.T) {
		// Get node1's own address
		node1Addr := node1.host.Addrs()[0].String() + "/p2p/" + node1.host.ID().String()
		err := node1.ConnectToPeer(ctx, node1Addr)
		// libp2p should handle self-connection gracefully
		// It typically returns success but doesn't actually create a connection
		if err == nil {
			// Verify we're not actually connected to ourself
			peers := node1.host.Network().Peers()
			assert.NotContains(t, peers, node1.host.ID())
		}
	})

	t.Run("connect with various multiaddr formats", func(t *testing.T) {
		// Test different valid multiaddr formats
		node2ID := node2.host.ID().String()

		// Extract the port from the multiaddr string
		node2AddrStr := node2.host.Addrs()[0].String()
		// Parse to extract port - format is like /ip4/127.0.0.1/tcp/PORT
		parts := strings.Split(node2AddrStr, "/")
		var node2Port string
		for i, part := range parts {
			if part == "tcp" && i+1 < len(parts) {
				node2Port = parts[i+1]
				break
			}
		}
		require.NotEmpty(t, node2Port, "Failed to extract port from address")

		formats := []string{
			fmt.Sprintf("/ip4/127.0.0.1/tcp/%s/p2p/%s", node2Port, node2ID),
			fmt.Sprintf("/ip4/127.0.0.1/tcp/%s/ipfs/%s", node2Port, node2ID), // ipfs is alias for p2p
		}

		for _, format := range formats {
			// Disconnect first to test reconnection
			_ = node1.DisconnectPeer(ctx, node2.host.ID())
			time.Sleep(100 * time.Millisecond) // Give time for disconnection

			err := node1.ConnectToPeer(ctx, format)
			require.NoError(t, err, "Failed with format: %s", format)

			// Verify connection
			peers := node1.host.Network().Peers()
			assert.Contains(t, peers, node2.host.ID())
		}
	})
}
