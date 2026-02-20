package p2p

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestP2PConfig_Validation(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		validate func(t *testing.T, config Config)
	}{
		{
			name: "valid basic config",
			config: Config{
				ProcessName:     "test-node",
				ListenAddresses: []string{"127.0.0.1"},
				Port:            4001,
			},
			validate: func(t *testing.T, config Config) {
				assert.Equal(t, "test-node", config.ProcessName)
				assert.Equal(t, 4001, config.Port)
				assert.Len(t, config.ListenAddresses, 1)
			},
		},
		{
			name: "config with private DHT",
			config: Config{
				ProcessName:        "private-node",
				ListenAddresses:    []string{"0.0.0.0"},
				Port:               5001,
				UsePrivateDHT:      true,
				SharedKey:          "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				DHTProtocolID:      "/custom/dht/1.0.0",
				BootstrapAddresses: []string{"/ip4/10.0.0.1/tcp/4001/p2p/12D3KooWTest"},
			},
			validate: func(t *testing.T, config Config) {
				assert.True(t, config.UsePrivateDHT)
				assert.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", config.SharedKey)
				assert.Equal(t, "/custom/dht/1.0.0", config.DHTProtocolID)
				assert.Len(t, config.BootstrapAddresses, 1)
			},
		},
		{
			name: "config with advertise addresses",
			config: Config{
				ProcessName:        "public-node",
				ListenAddresses:    []string{"0.0.0.0"},
				AdvertiseAddresses: []string{"1.2.3.4", "example.com:4002"},
				Port:               4001,
				Advertise:          true,
			},
			validate: func(t *testing.T, config Config) {
				assert.True(t, config.Advertise)
				assert.Len(t, config.AdvertiseAddresses, 2)
				assert.Contains(t, config.AdvertiseAddresses, "1.2.3.4")
				assert.Contains(t, config.AdvertiseAddresses, "example.com:4002")
			},
		},
		{
			name: "config with static peers",
			config: Config{
				ProcessName:     "connected-node",
				ListenAddresses: []string{"127.0.0.1"},
				Port:            4001,
				StaticPeers: []string{
					"/ip4/192.168.1.10/tcp/4001/p2p/12D3KooWTest1",
					"/ip4/192.168.1.11/tcp/4001/p2p/12D3KooWTest2",
				},
				OptimiseRetries: true,
			},
			validate: func(t *testing.T, config Config) {
				assert.True(t, config.OptimiseRetries)
				assert.Len(t, config.StaticPeers, 2)
			},
		},
		{
			name: "empty config fields",
			config: Config{
				ProcessName:     "",
				ListenAddresses: []string{},
				Port:            0,
			},
			validate: func(t *testing.T, config Config) {
				assert.Empty(t, config.ProcessName)
				assert.Empty(t, config.ListenAddresses)
				assert.Zero(t, config.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.config)
		})
	}
}

func TestP2PNode_AtomicOperations(t *testing.T) {
	node := &Node{
		bytesSent:     0,
		bytesReceived: 0,
		lastSend:      0,
		lastRecv:      0,
	}

	// Test concurrent atomic operations
	var wg sync.WaitGroup
	iterations := 1000
	goroutines := 10

	// Test bytesSent increments
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				atomic.AddUint64(&node.bytesSent, 1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, uint64(goroutines*iterations), atomic.LoadUint64(&node.bytesSent))

	// Test bytesReceived increments
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				atomic.AddUint64(&node.bytesReceived, 10)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, uint64(goroutines*iterations*10), atomic.LoadUint64(&node.bytesReceived))

	// Test timestamp updates
	now := time.Now().Unix()
	atomic.StoreInt64(&node.lastSend, now)
	atomic.StoreInt64(&node.lastRecv, now+1)

	assert.Equal(t, now, atomic.LoadInt64(&node.lastSend))
	assert.Equal(t, now+1, atomic.LoadInt64(&node.lastRecv))
}

func TestP2PNode_ThreadSafeMaps(t *testing.T) {
	node := &Node{
		peerHeights:   sync.Map{},
		peerConnTimes: sync.Map{},
	}

	// Test concurrent map operations
	var wg sync.WaitGroup
	peers := 100
	testPeerIDs := []string{
		"12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F",
		"12D3KooWLRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8G",
		"12D3KooWMRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8H",
	}

	// Test peerHeights concurrent access
	wg.Add(peers * 2)
	for i := 0; i < peers; i++ {
		peerID := testPeerIDs[i%len(testPeerIDs)]

		// Writer goroutine
		go func(id string, height int32) {
			defer wg.Done()
			node.peerHeights.Store(id, height)
		}(peerID, int32(i))

		// Reader goroutine
		go func(id string) {
			defer wg.Done()
			node.peerHeights.Load(id)
		}(peerID)
	}
	wg.Wait()

	// Verify some values were stored
	count := 0
	node.peerHeights.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	assert.Positive(t, count)

	// Test peerConnTimes concurrent access
	wg.Add(peers)
	for i := 0; i < peers; i++ {
		go func(i int) {
			defer wg.Done()
			peerID := testPeerIDs[i%len(testPeerIDs)]
			node.peerConnTimes.Store(peerID, time.Now())
			node.peerConnTimes.Load(peerID)
		}(i)
	}
	wg.Wait()
}

func TestHandler_FunctionType(t *testing.T) {
	// Test that Handler type can be properly instantiated
	var handler Handler = func(_ context.Context, _ []byte, _ string) {
		// Sample handler implementation
	}

	assert.NotNil(t, handler)
}

func TestConstants(t *testing.T) {
	// Verify constants are defined correctly
	assert.Equal(t, "[Node] error creating DHT: %w", errorCreatingDhtMessage)
	assert.Equal(t, "/ip4/%s/tcp/%d", multiAddrIPTemplate)
}
