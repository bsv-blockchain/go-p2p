package p2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	t *testing.T
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.t.Logf("[DEBUG] "+format, args...)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.t.Logf("[INFO] "+format, args...)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.t.Logf("[WARN] "+format, args...)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.t.Logf("[ERROR] "+format, args...)
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.t.Fatalf("[FATAL] "+format, args...)
}

func TestNATTraversalConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "Basic NAT configuration",
			config: Config{
				ProcessName:        "test-nat-basic",
				ListenAddresses:    []string{"127.0.0.1"},
				Port:               0, // Random port
				EnableNATService:   true,
				EnableNATPortMap:   true,
				EnableHolePunching: true,
				EnableRelay:        false,
				EnableAutoNATv2:    false,
			},
		},
		{
			name: "AutoNAT v2 enabled",
			config: Config{
				ProcessName:        "test-nat-autonat",
				ListenAddresses:    []string{"127.0.0.1"},
				Port:               0,
				EnableNATService:   true,
				EnableAutoNATv2:    true,
				EnableHolePunching: true,
			},
		},
		{
			name: "Force public reachability",
			config: Config{
				ProcessName:       "test-nat-public",
				ListenAddresses:   []string{"127.0.0.1"},
				Port:              0,
				EnableNATService:  true,
				EnableAutoNATv2:   true,
				ForceReachability: "public",
			},
		},
		{
			name: "Force private reachability",
			config: Config{
				ProcessName:       "test-nat-private",
				ListenAddresses:   []string{"127.0.0.1"},
				Port:              0,
				EnableNATService:  true,
				ForceReachability: "private",
			},
		},
		{
			name: "Relay enabled with service",
			config: Config{
				ProcessName:        "test-nat-relay",
				ListenAddresses:    []string{"127.0.0.1"},
				Port:               0,
				EnableRelay:        true,
				EnableRelayService: true,
				EnableAutoNATv2:    true,
			},
		},
		{
			name: "Full NAT traversal stack",
			config: Config{
				ProcessName:        "test-nat-full",
				ListenAddresses:    []string{"127.0.0.1"},
				Port:               0,
				EnableNATService:   true,
				EnableNATPortMap:   true,
				EnableHolePunching: true,
				EnableRelay:        true,
				EnableRelayService: true,
				EnableAutoNATv2:    true,
				ForceReachability:  "public",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			logger := &MockLogger{t: t}

			// Create node with NAT configuration
			node, err := NewNode(ctx, logger, tt.config)
			require.NoError(t, err, "Failed to create node with NAT config")
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

func TestNATConfigurationValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		shouldErr bool
	}{
		{
			name: "Valid ForceReachability - public",
			config: Config{
				ProcessName:       "test-valid-public",
				ListenAddresses:   []string{"127.0.0.1"},
				Port:              0,
				ForceReachability: "public",
			},
			shouldErr: false,
		},
		{
			name: "Valid ForceReachability - private",
			config: Config{
				ProcessName:       "test-valid-private",
				ListenAddresses:   []string{"127.0.0.1"},
				Port:              0,
				ForceReachability: "private",
			},
			shouldErr: false,
		},
		{
			name: "Valid ForceReachability - auto (empty)",
			config: Config{
				ProcessName:       "test-valid-auto",
				ListenAddresses:   []string{"127.0.0.1"},
				Port:              0,
				ForceReachability: "",
			},
			shouldErr: false,
		},
		{
			name: "Invalid ForceReachability value",
			config: Config{
				ProcessName:       "test-invalid-reach",
				ListenAddresses:   []string{"127.0.0.1"},
				Port:              0,
				ForceReachability: "invalid", // This will be ignored, defaulting to auto
			},
			shouldErr: false, // Should not error, just use auto-detect
		},
		{
			name: "EnableRelayService without EnableRelay",
			config: Config{
				ProcessName:        "test-relay-service-only",
				ListenAddresses:    []string{"127.0.0.1"},
				Port:               0,
				EnableRelay:        false,
				EnableRelayService: true, // This will be ignored since EnableRelay is false
			},
			shouldErr: false, // Should not error, relay service just won't be enabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			logger := &MockLogger{t: t}

			node, err := NewNode(ctx, logger, tt.config)

			if tt.shouldErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error")
				if node != nil {
					// Clean up
					_ = node.Stop(ctx)
				}
			}
		})
	}
}

func TestNATOptionsIntegration(t *testing.T) {
	// Test that NAT options are properly integrated into libp2p options
	configs := []Config{
		{
			EnableNATService: true,
			EnableNATPortMap: false,
		},
		{
			EnableNATService: false,
			EnableNATPortMap: true,
		},
		{
			EnableNATService: true,
			EnableNATPortMap: true,
		},
	}

	for i, config := range configs {
		t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
			opts := []libp2p.Option{}
			resultOpts := addNATAndRelayOptions(opts, config)

			// Check that options were added
			// We can't directly inspect libp2p.Option values,
			// but we can verify the function doesn't panic
			assert.NotNil(t, resultOpts)

			// Verify that NATPortMap is added if either EnableNATService or EnableNATPortMap is true
			if config.EnableNATService || config.EnableNATPortMap {
				// At least one NAT option should be added
				assert.NotEqual(t, len(opts), len(resultOpts), "NAT options should be added")
			}
		})
	}
}

func TestAutoNATv2Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger := &MockLogger{t: t}

	// Create two nodes - one with AutoNAT v2, one without
	config1 := Config{
		ProcessName:     "node-with-autonat",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		EnableAutoNATv2: true,
		EnableRelay:     true,
	}

	config2 := Config{
		ProcessName:     "node-without-autonat",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		EnableAutoNATv2: false,
		EnableRelay:     false,
	}

	node1, err := NewNode(ctx, logger, config1)
	require.NoError(t, err)
	defer node1.Stop(ctx)

	node2, err := NewNode(ctx, logger, config2)
	require.NoError(t, err)
	defer node2.Stop(ctx)

	// Start both nodes
	err = node1.Start(ctx, nil)
	require.NoError(t, err)

	err = node2.Start(ctx, nil)
	require.NoError(t, err)

	// Nodes should be able to start with different NAT configurations
	assert.NotEqual(t, node1.host.ID(), node2.host.ID())

	// Both should have addresses
	assert.NotEmpty(t, node1.host.Addrs())
	assert.NotEmpty(t, node2.host.Addrs())
}
