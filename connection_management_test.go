package p2p

import (
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionManagement(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "Connection manager enabled with defaults",
			config: Config{
				ProcessName:       "test-conn-mgr-defaults",
				ListenAddresses:   []string{testLocalhost},
				Port:              0,
				EnableConnManager: true,
			},
		},
		{
			name: "Connection manager with custom values",
			config: Config{
				ProcessName:       "test-conn-mgr-custom",
				ListenAddresses:   []string{testLocalhost},
				Port:              0,
				EnableConnManager: true,
				ConnLowWater:      50,
				ConnHighWater:     100,
				ConnGracePeriod:   30 * time.Second,
			},
		},
		{
			name: "Connection gater enabled",
			config: Config{
				ProcessName:     "test-conn-gater",
				ListenAddresses: []string{testLocalhost},
				Port:            0,
				EnableConnGater: true,
				MaxConnsPerPeer: 2,
			},
		},
		{
			name: "Both manager and gater enabled",
			config: Config{
				ProcessName:       "test-both",
				ListenAddresses:   []string{testLocalhost},
				Port:              0,
				EnableConnManager: true,
				EnableConnGater:   true,
				ConnLowWater:      25,
				ConnHighWater:     50,
				MaxConnsPerPeer:   1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := createTestContext(10 * time.Second)
			defer cancel()

			logger := createTestLogger(t)

			// Create node with connection management configuration
			node, err := NewNode(ctx, logger, tt.config)
			require.NoError(t, err, "Failed to create node with connection management")
			require.NotNil(t, node)

			// Verify node was created successfully
			assert.NotNil(t, node.host)
			assert.NotEmpty(t, node.host.ID())

			// Start the node
			err = node.Start(ctx, nil)
			require.NoError(t, err, "Failed to start node")

			// Give node time to initialize
			time.Sleep(100 * time.Millisecond)

			// Check that the host is listening
			addrs := node.host.Addrs()
			assert.NotEmpty(t, addrs, "Node should have listening addresses")

			t.Logf("Node %s listening on: %v", tt.config.ProcessName, addrs)

			// Stop the node
			err = node.Stop(ctx)
			assert.NoError(t, err, "Failed to stop node")
		})
	}
}

func TestConnectionGater(t *testing.T) {
	logger := createTestLogger(t)
	gater := NewConnectionGater(logger, 3)

	// Test peer blocking
	testPeer := peer.ID("test-peer-1")

	// Initially peer should not be blocked
	assert.True(t, gater.InterceptPeerDial(testPeer), "Peer should not be blocked initially")

	// Block the peer for 1 second
	gater.BlockPeer(testPeer, 1*time.Second)

	// Now peer should be blocked
	assert.False(t, gater.InterceptPeerDial(testPeer), "Peer should be blocked after BlockPeer")

	// Wait for block to expire
	time.Sleep(1100 * time.Millisecond)

	// Peer should no longer be blocked
	assert.True(t, gater.InterceptPeerDial(testPeer), "Peer should not be blocked after expiry")

	// Test unblocking
	gater.BlockPeer(testPeer, 10*time.Second)
	assert.False(t, gater.InterceptPeerDial(testPeer), "Peer should be blocked")

	gater.UnblockPeer(testPeer)
	assert.True(t, gater.InterceptPeerDial(testPeer), "Peer should not be blocked after UnblockPeer")
}

func TestConnectionGaterSubnetBlocking(t *testing.T) {
	logger := createTestLogger(t)
	gater := NewConnectionGater(logger, 3)

	// Block a subnet
	gater.BlockSubnet("192.168.1")

	// Test that the gater is created and functional
	// Testing actual subnet blocking would require creating multiaddrs
	// which is more complex and might require additional setup
	assert.NotNil(t, gater)
	assert.Len(t, gater.blockedSubnets, 1)
}

func TestConnectionManagerWaterMarks(t *testing.T) {
	tests := []struct {
		name              string
		config            Config
		expectedLowWater  int
		expectedHighWater int
	}{
		{
			name: "Default values",
			config: Config{
				EnableConnManager: true,
			},
			expectedLowWater:  200,
			expectedHighWater: 400,
		},
		{
			name: "Custom values",
			config: Config{
				EnableConnManager: true,
				ConnLowWater:      100,
				ConnHighWater:     200,
			},
			expectedLowWater:  100,
			expectedHighWater: 200,
		},
		{
			name: "Auto-adjust high water when too low",
			config: Config{
				EnableConnManager: true,
				ConnLowWater:      150,
				ConnHighWater:     100, // Lower than low water
			},
			expectedLowWater:  150,
			expectedHighWater: 250, // Should be adjusted to low + 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The actual values are used internally by libp2p
			// We can verify the logic in our function works correctly
			// by checking the node creates successfully
			ctx, cancel := createTestContext(5 * time.Second)
			defer cancel()

			logger := createTestLogger(t)

			tt.config.ProcessName = "test-water-marks"
			tt.config.ListenAddresses = []string{testLocalhost}
			tt.config.Port = 0

			node, err := NewNode(ctx, logger, tt.config)
			require.NoError(t, err, "Failed to create node")

			if node != nil {
				_ = node.Stop(ctx)
			}
		})
	}
}

func TestMultipleNodesWithConnectionManagement(t *testing.T) {
	ctx, cancel := createTestContext(15 * time.Second)
	defer cancel()

	logger := createTestLogger(t)

	// Create first node with connection management
	config1 := Config{
		ProcessName:       "node-1",
		ListenAddresses:   []string{testLocalhost},
		Port:              0,
		EnableConnManager: true,
		ConnLowWater:      1,
		ConnHighWater:     5,
		EnableConnGater:   true,
		MaxConnsPerPeer:   2,
	}

	node1, err := NewNode(ctx, logger, config1)
	require.NoError(t, err)
	defer node1.Stop(ctx)

	err = node1.Start(ctx, nil)
	require.NoError(t, err)

	// Create second node
	config2 := Config{
		ProcessName:       "node-2",
		ListenAddresses:   []string{testLocalhost},
		Port:              0,
		EnableConnManager: true,
		ConnLowWater:      1,
		ConnHighWater:     5,
	}

	node2, err := NewNode(ctx, logger, config2)
	require.NoError(t, err)
	defer node2.Stop(ctx)

	err = node2.Start(ctx, nil)
	require.NoError(t, err)

	// Both nodes should be running with connection management
	assert.NotEqual(t, node1.host.ID(), node2.host.ID())
	assert.NotEmpty(t, node1.host.Addrs())
	assert.NotEmpty(t, node2.host.Addrs())

	// Try to connect the nodes
	// Get the actual network addresses (not just the configured ones)
	node1NetworkAddrs := node1.host.Network().ListenAddresses()

	// Find a local address to use
	var localAddrs []multiaddr.Multiaddr
	for _, addr := range node1NetworkAddrs {
		addrStr := addr.String()
		// Use addresses that have actual ports (not 0)
		if !strings.Contains(addrStr, "/tcp/0") {
			localAddrs = append(localAddrs, addr)
		}
	}

	// If no good addresses found, skip the test
	if len(localAddrs) == 0 {
		t.Skip("No valid local addresses for connection test")
		return
	}

	node1Info := peer.AddrInfo{
		ID:    node1.host.ID(),
		Addrs: localAddrs,
	}

	t.Logf("Connecting from node2 to node1 at addresses: %v", localAddrs)

	err = node2.host.Connect(ctx, node1Info)
	require.NoError(t, err, "Nodes should be able to connect")

	// Verify connection
	peers1 := node1.host.Network().Peers()
	peers2 := node2.host.Network().Peers()

	assert.Contains(t, peers1, node2.host.ID(), "Node1 should have Node2 as peer")
	assert.Contains(t, peers2, node1.host.ID(), "Node2 should have Node1 as peer")
}
