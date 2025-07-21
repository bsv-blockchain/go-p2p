package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP2PNode_StaticPeerConnection(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create two nodes for testing connections
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create first node
	config1 := Config{
		ProcessName:        "node1",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3111, // Random port
	}

	node1, err := NewNode(ctx, logger, config1)
	require.NoError(t, err)
	defer func() {
		if err := node1.Stop(ctx); err != nil { //nolint:govet // Intentional shadowing in defer
			t.Logf("Failed to stop node1 in cleanup: %v", err)
		}
	}()

	// Get node1's address for static peer config
	node1Addr := fmt.Sprintf("%s/p2p/%s", node1.host.Addrs()[0], node1.host.ID())

	// Create second node with static peer
	config2 := Config{
		ProcessName:        "node2",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3112,
		StaticPeers:        []string{node1Addr},
	}

	node2, err := NewNode(ctx, logger, config2)
	require.NoError(t, err)
	defer func() {
		if err := node2.Stop(ctx); err != nil {
			t.Logf("Failed to stop node2 in cleanup: %v", err)
		}
	}()

	t.Run("connectToStaticPeers", func(t *testing.T) {
		// Test connection to static peers
		allConnected := node2.connectToStaticPeers(ctx, config2.StaticPeers)
		assert.True(t, allConnected)

		// Verify connection
		assert.Equal(t, network.Connected, node2.host.Network().Connectedness(node1.host.ID()))
	})

	t.Run("already connected peer", func(t *testing.T) {
		// Call again when already connected
		allConnected := node2.connectToStaticPeers(ctx, config2.StaticPeers)
		assert.True(t, allConnected)
	})

	t.Run("invalid static peer address", func(t *testing.T) {
		invalidPeers := []string{
			"invalid-address",
			"/ip4/1.2.3.4/tcp/4001/p2p/invalid-peer-id",
		}

		allConnected := node2.connectToStaticPeers(ctx, invalidPeers)
		assert.False(t, allConnected)
	})
}

func TestP2PNode_ShouldSkipPeer(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	config := Config{
		ProcessName:     "skip-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		OptimiseRetries: false,
	}

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() {
		if err := node.host.Close(); err != nil {
			t.Logf("Failed to close host in cleanup: %v", err)
		}
	}()

	selfID := node.host.ID()
	otherID, _ := peer.Decode("12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F")

	tests := []struct {
		name            string
		addr            peer.AddrInfo
		setupFunc       func()
		optimizeRetries bool
		expectedSkip    bool
	}{
		{
			name: "skip self connection",
			addr: peer.AddrInfo{
				ID: selfID,
			},
			expectedSkip: true,
		},
		{
			name: "skip peer with no addresses",
			addr: peer.AddrInfo{
				ID:    otherID,
				Addrs: []multiaddr.Multiaddr{},
			},
			expectedSkip: true,
		},
		{
			name: "don't skip valid peer",
			addr: peer.AddrInfo{
				ID:    otherID,
				Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/1.2.3.4/tcp/4001")},
			},
			expectedSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			node.config.OptimiseRetries = tt.optimizeRetries
			peerAddrErrorMap := &sync.Map{}

			result := node.shouldSkipPeer(tt.addr, peerAddrErrorMap)
			assert.Equal(t, tt.expectedSkip, result)
		})
	}
}

func TestP2PNode_ShouldSkipBasedOnErrors(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	config := Config{
		ProcessName:     "error-skip-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		OptimiseRetries: true,
	}

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() {
		if err := node.host.Close(); err != nil {
			t.Logf("Failed to close host in cleanup: %v", err)
		}
	}()

	peerID, _ := peer.Decode("12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F")

	tests := []struct {
		name         string
		addr         peer.AddrInfo
		errorString  string
		expectedSkip bool
	}{
		{
			name: "skip on peer id mismatch",
			addr: peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/1.2.3.4/tcp/4001")},
			},
			errorString:  "peer id mismatch",
			expectedSkip: true,
		},
		{
			name: "skip on no good addresses with localhost only",
			addr: peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/127.0.0.1/tcp/4001")},
			},
			errorString:  "no good addresses",
			expectedSkip: true,
		},
		{
			name: "don't skip on no good addresses with public IP",
			addr: peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/1.2.3.4/tcp/4001")},
			},
			errorString:  "no good addresses",
			expectedSkip: false,
		},
		{
			name: "don't skip on other errors",
			addr: peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/1.2.3.4/tcp/4001")},
			},
			errorString:  "connection refused",
			expectedSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerAddrErrorMap := &sync.Map{}

			// Store the error
			peerAddrErrorMap.Store(tt.addr.ID.String(), tt.errorString)

			result := node.shouldSkipBasedOnErrors(tt.addr, peerAddrErrorMap)
			assert.Equal(t, tt.expectedSkip, result)
		})
	}
}

func TestP2PNode_ShouldSkipNoGoodAddresses(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	config := Config{
		ProcessName:     "no-good-addr-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() {
		if err := node.host.Close(); err != nil {
			t.Logf("Failed to close host in cleanup: %v", err)
		}
	}()

	peerID, _ := peer.Decode("12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F")

	tests := []struct {
		name         string
		addr         peer.AddrInfo
		expectedSkip bool
	}{
		{
			name: "skip peer with no addresses",
			addr: peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{},
			},
			expectedSkip: true,
		},
		{
			name: "skip peer with only localhost",
			addr: peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/127.0.0.1/tcp/4001")},
			},
			expectedSkip: true,
		},
		{
			name: "don't skip peer with public address",
			addr: peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/1.2.3.4/tcp/4001")},
			},
			expectedSkip: false,
		},
		{
			name: "don't skip peer with multiple addresses including localhost",
			addr: peer.AddrInfo{
				ID: peerID,
				Addrs: []multiaddr.Multiaddr{
					multiaddr.StringCast("/ip4/127.0.0.1/tcp/4001"),
					multiaddr.StringCast("/ip4/1.2.3.4/tcp/4001"),
				},
			},
			expectedSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.shouldSkipNoGoodAddresses(tt.addr)
			assert.Equal(t, tt.expectedSkip, result)
		})
	}
}

func TestP2PNode_AttemptConnection(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()

	// Create two nodes
	config1 := Config{
		ProcessName:        "node1",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3111,
	}

	node1, err := NewNode(ctx, logger, config1)
	require.NoError(t, err)
	defer node1.host.Close()

	config2 := Config{
		ProcessName:        "node2",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.1"},
		Port:               3112,
	}

	node2, err := NewNode(ctx, logger, config2)
	require.NoError(t, err)
	defer node2.host.Close()

	t.Run("successful connection", func(t *testing.T) {
		peerAddrMap := &sync.Map{}
		peerAddrErrorMap := &sync.Map{}

		peerAddr := peer.AddrInfo{
			ID:    node1.host.ID(),
			Addrs: node1.host.Addrs(),
		}

		node2.attemptConnection(ctx, peerAddr, peerAddrMap, peerAddrErrorMap)

		// Check connection was attempted
		_, attempted := peerAddrMap.Load(peerAddr.ID.String())
		assert.True(t, attempted)

		// Should be connected
		assert.Equal(t, network.Connected, node2.host.Network().Connectedness(node1.host.ID()))
	})

	t.Run("already attempted connection", func(t *testing.T) {
		peerAddrMap := &sync.Map{}
		peerAddrErrorMap := &sync.Map{}

		peerAddr := peer.AddrInfo{
			ID:    node1.host.ID(),
			Addrs: node1.host.Addrs(),
		}

		// Mark as already attempted
		peerAddrMap.Store(peerAddr.ID.String(), true)

		// Should not attempt again
		node2.attemptConnection(ctx, peerAddr, peerAddrMap, peerAddrErrorMap)

		// Error map should not have an entry (no attempt made)
		_, hasError := peerAddrErrorMap.Load(peerAddr.ID.String())
		assert.False(t, hasError)
	})

	t.Run("failed connection", func(t *testing.T) {
		peerAddrMap := &sync.Map{}
		peerAddrErrorMap := &sync.Map{}

		// Invalid peer address
		invalidPeerID, _ := peer.Decode("12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F")
		peerAddr := peer.AddrInfo{
			ID:    invalidPeerID,
			Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/1.2.3.4/tcp/4001")},
		}

		node2.attemptConnection(ctx, peerAddr, peerAddrMap, peerAddrErrorMap)

		// Should have attempted
		_, attempted := peerAddrMap.Load(peerAddr.ID.String())
		assert.True(t, attempted)

		// Should have error
		_, hasError := peerAddrErrorMap.Load(peerAddr.ID.String())
		assert.True(t, hasError)
	})
}

func TestP2PNode_PrivateNetwork(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()

	t.Run("setUpPrivateNetwork success", func(t *testing.T) {
		config := Config{
			ProcessName:     "private-test",
			ListenAddresses: []string{"127.0.0.1"},
			Port:            4001,
			SharedKey:       "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}

		pk, err := generatePrivateKey(ctx)
		require.NoError(t, err)

		host, err := setUpPrivateNetwork(logger, config, pk)
		require.NoError(t, err)
		require.NotNil(t, host)
		defer func() {
			if err := host.Close(); err != nil {
				t.Logf("Failed to close host in cleanup: %v", err)
			}
		}()

		// Verify host is running
		assert.NotEmpty(t, host.ID())
		assert.NotEmpty(t, host.Addrs())
	})

	t.Run("setUpPrivateNetwork with advertise addresses", func(t *testing.T) {
		config := Config{
			ProcessName:        "private-advertise-test",
			ListenAddresses:    []string{"0.0.0.0"},
			AdvertiseAddresses: []string{"1.2.3.4"},
			Port:               4002,
			SharedKey:          "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}

		pk, err := generatePrivateKey(ctx)
		require.NoError(t, err)

		host, err := setUpPrivateNetwork(logger, config, pk)
		require.NoError(t, err)
		require.NotNil(t, host)
		defer func() {
			if err := host.Close(); err != nil {
				t.Logf("Failed to close host in cleanup: %v", err)
			}
		}()
	})

	t.Run("setUpPrivateNetwork invalid shared key", func(t *testing.T) {
		config := Config{
			ProcessName:     "bad-private-test",
			ListenAddresses: []string{"127.0.0.1"},
			Port:            4003,
			SharedKey:       "invalid-key",
		}

		pk, err := generatePrivateKey(ctx)
		require.NoError(t, err)

		host, err := setUpPrivateNetwork(logger, config, pk)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error decoding shared key")
		assert.Nil(t, host)
	})
}

func TestP2PNode_InitDHT(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	config := Config{
		ProcessName:     "dht-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            3111,
	}

	node, err := NewNode(ctx, logger, config)
	require.NoError(t, err)
	defer func() {
		if err := node.host.Close(); err != nil {
			t.Logf("Failed to close host in cleanup: %v", err)
		}
	}()

	t.Run("initDHT success", func(t *testing.T) {
		// Use a short timeout context to avoid waiting for bootstrap
		dhtCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		dht, err := node.initDHT(dhtCtx, node.host)
		// May timeout on bootstrap, but DHT should still be created
		assert.NotNil(t, dht)
		if err != nil {
			assert.Contains(t, err.Error(), "context")
		}
	})
}

func TestP2PNode_InitPrivateDHT(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()

	t.Run("no bootstrap addresses", func(t *testing.T) {
		config := Config{
			ProcessName:        "private-dht-test",
			ListenAddresses:    []string{"127.0.0.1"},
			Port:               3111,
			BootstrapAddresses: []string{},
			DHTProtocolID:      "/test/dht/1.0.0",
		}

		node, err := NewNode(ctx, logger, config)
		require.NoError(t, err)
		defer func() {
			if err := node.host.Close(); err != nil { //nolint:govet // Intentional shadowing in defer
				t.Logf("Failed to close host in cleanup: %v", err)
			}
		}()

		dht, err := node.initPrivateDHT(ctx, node.host)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bootstrapAddresses not set")
		assert.Nil(t, dht)
	})

	t.Run("invalid bootstrap addresses", func(t *testing.T) {
		config := Config{
			ProcessName:        "private-dht-test2",
			ListenAddresses:    []string{"127.0.0.1"},
			Port:               0,
			BootstrapAddresses: []string{"invalid-address"},
			DHTProtocolID:      "/test/dht/1.0.0",
		}

		node, err := NewNode(ctx, logger, config)
		require.NoError(t, err)
		defer func() {
			if err := node.host.Close(); err != nil { //nolint:govet // Intentional shadowing in defer
				t.Logf("Failed to close host in cleanup: %v", err)
			}
		}()

		dht, err := node.initPrivateDHT(ctx, node.host)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to any bootstrap addresses")
		assert.Nil(t, dht)
	})

	t.Run("missing DHT protocol ID", func(t *testing.T) {
		config := Config{
			ProcessName:        "private-dht-test3",
			ListenAddresses:    []string{"127.0.0.1"},
			Port:               0,
			BootstrapAddresses: []string{"/ip4/1.2.3.4/tcp/4001/p2p/12D3KooWTest"},
			DHTProtocolID:      "",
		}

		node, err := NewNode(ctx, logger, config)
		require.NoError(t, err)
		defer func() {
			if err := node.host.Close(); err != nil { //nolint:govet // Intentional shadowing in defer
				t.Logf("Failed to close host in cleanup: %v", err)
			}
		}()

		dht, err := node.initPrivateDHT(ctx, node.host)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting p2p_dht_protocol_id")
		assert.Nil(t, dht)
	})
}

// MockHost for testing - implements minimal host.Host interface
type mockHost struct {
	host.Host

	id       peer.ID
	network  network.Network
	conns    map[peer.ID]network.Connectedness
	connLock sync.RWMutex
}

func newMockHost(id peer.ID) *mockHost {
	return &mockHost{
		id:    id,
		conns: make(map[peer.ID]network.Connectedness),
	}
}

func (m *mockHost) ID() peer.ID {
	return m.id
}

func (m *mockHost) Network() network.Network {
	return m.network
}

func (m *mockHost) Connect(_ context.Context, pi peer.AddrInfo) error {
	m.connLock.Lock()
	defer m.connLock.Unlock()

	m.conns[pi.ID] = network.Connected
	return nil
}

func (m *mockHost) Addrs() []multiaddr.Multiaddr {
	return []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/127.0.0.1/tcp/4001")}
}
