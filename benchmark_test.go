// Package p2p provides benchmark tests for P2P networking functionality.
package p2p

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func BenchmarkP2PNode_Publish(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	config := Config{
		ProcessName:        "bench-publish",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3111,
	}

	node, err := NewNode(ctx, logger, config)
	require.NoError(b, err)
	setupNodeCleanup(b, node, ctx, "bench-publish")

	err = node.Start(ctx, nil, "bench-topic")
	require.NoError(b, err)

	// Test different message sizes
	messageSizes := []int{
		100,     // 100 bytes
		1024,    // 1 KB
		10240,   // 10 KB
		102400,  // 100 KB
		1048576, // 1 MB
	}

	for _, size := range messageSizes {
		msg := make([]byte, size)
		for i := range msg {
			msg[i] = byte(i % 256)
		}

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				err := node.Publish(ctx, "bench-topic", msg)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkP2PNode_SendToPeer(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()

	// Create two nodes
	config1 := Config{
		ProcessName:        "bench-sender",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3111,
	}

	sender, err := NewNode(ctx, logger, config1)
	require.NoError(b, err)
	setupNodeCleanup(b, sender, ctx, "bench-sender")

	config2 := Config{
		ProcessName:        "bench-receiver",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3112,
	}

	receiver, err := NewNode(ctx, logger, config2)
	require.NoError(b, err)
	setupNodeCleanup(b, receiver, ctx, "bench-receiver")

	// Set up stream handler
	streamHandler := func(stream network.Stream) {
		buf := make([]byte, 1048576) // 1MB buffer
		for {
			_, err = stream.Read(buf)
			if err != nil {
				if closeErr := stream.Close(); closeErr != nil {
					b.Logf("Failed to close stream: %v", closeErr)
				}
				return
			}
		}
	}

	err = sender.Start(ctx, nil)
	require.NoError(b, err)

	err = receiver.Start(ctx, streamHandler)
	require.NoError(b, err)

	// Connect nodes
	err = sender.host.Connect(ctx, peer.AddrInfo{
		ID:    receiver.host.ID(),
		Addrs: receiver.host.Addrs(),
	})
	require.NoError(b, err)

	// Benchmark different message sizes
	messageSizes := []int{100, 1024, 10240, 102400}

	for _, size := range messageSizes {
		msg := make([]byte, size)
		for i := range msg {
			msg[i] = byte(i % 256)
		}

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				err := sender.SendToPeer(ctx, receiver.host.ID(), msg)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkP2PNode_ConcurrentConnections(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()

	// Create a central node
	centralConfig := Config{
		ProcessName:        "central",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3111,
	}

	central, err := NewNode(ctx, logger, centralConfig)
	require.NoError(b, err)
	setupNodeCleanup(b, central, ctx, "central")

	err = central.Start(ctx, nil)
	require.NoError(b, err)

	centralAddr := fmt.Sprintf("%s/p2p/%s", central.host.Addrs()[0], central.host.ID())

	concurrencyLevels := []int{10, 50, 100}

	for _, numNodes := range concurrencyLevels {
		b.Run(fmt.Sprintf("nodes_%d", numNodes), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				nodes := make([]*Node, numNodes)
				var wg sync.WaitGroup

				// Create and connect nodes concurrently
				wg.Add(numNodes)
				for j := 0; j < numNodes; j++ {
					go func(idx int) {
						defer wg.Done()

						config := Config{
							ProcessName:     fmt.Sprintf("node%d", idx),
							ListenAddresses: []string{"127.0.0.1"},
							Port:            3112 + j,
							StaticPeers:     []string{centralAddr},
						}

						var node *Node
						node, err = NewNode(ctx, logger, config)
						if err != nil {
							b.Error(err)
							return
						}
						nodes[idx] = node

						err = node.Start(ctx, nil)
						if err != nil {
							b.Error(err)
						}
					}(j)
				}

				wg.Wait()

				// Clean up nodes
				for _, node := range nodes {
					if node != nil {
						err = node.Stop(ctx)
						require.NoError(b, err)
					}
				}
			}
		})
	}
}

func BenchmarkP2PNode_MessageRouting(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a network of nodes
	numNodes := 5
	nodes := make([]*Node, numNodes)

	for i := 0; i < numNodes; i++ {
		config := Config{
			ProcessName:        fmt.Sprintf("router%d", i),
			ListenAddresses:    []string{"127.0.0.1"},
			AdvertiseAddresses: []string{"127.0.0.1"},
			Port:               3111 + i,
		}

		if i > 0 {
			// Connect to previous node
			prevNode := nodes[i-1]
			config.StaticPeers = []string{
				fmt.Sprintf("%s/p2p/%s", prevNode.host.Addrs()[0], prevNode.host.ID()),
			}
		}

		node, err := NewNode(ctx, logger, config)
		require.NoError(b, err)

		err = node.Start(ctx, nil, "bench-routing")
		require.NoError(b, err)

		nodes[i] = node
		defer func(n *Node, ctx context.Context) {
			if err := n.Stop(ctx); err != nil {
				b.Logf("Failed to stop node %s in cleanup: %v", n.GetProcessName(), err)
			}
		}(node, ctx)
	}

	// Set up message handlers
	messageCount := &sync.Map{}
	for i, node := range nodes {
		nodeID := i
		handler := func(_ context.Context, _ []byte, _ string) {
			// Count messages received
			val, _ := messageCount.LoadOrStore(nodeID, int64(0))
			messageCount.Store(nodeID, val.(int64)+1)
		}

		err := node.SetTopicHandler(ctx, "bench-routing", handler)
		require.NoError(b, err)
	}

	// Wait for network to stabilize
	time.Sleep(2 * time.Second)

	msg := []byte("benchmark routing message")

	b.ResetTimer()
	b.SetBytes(int64(len(msg)))

	for i := 0; i < b.N; i++ {
		// Publish from random node
		nodeIdx := i % numNodes
		err := nodes[nodeIdx].Publish(ctx, "bench-routing", msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkP2PNode_AtomicOperations(b *testing.B) {
	node := &Node{}

	b.Run("BytesSent", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				atomic.AddUint64(&node.bytesSent, 100)
			}
		})
	})

	b.Run("BytesReceived", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				atomic.AddUint64(&node.bytesReceived, 100)
			}
		})
	})

	b.Run("LastSend", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				atomic.StoreInt64(&node.lastSend, time.Now().Unix())
			}
		})
	})

	b.Run("LastRecv", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				atomic.StoreInt64(&node.lastRecv, time.Now().Unix())
			}
		})
	})
}

func BenchmarkP2PNode_PeerManagement(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	config := Config{
		ProcessName:        "bench-peer-mgmt",
		ListenAddresses:    []string{"127.0.0.1"},
		AdvertiseAddresses: []string{"127.0.0.1"},
		Port:               3111,
	}

	node, err := NewNode(ctx, logger, config)
	require.NoError(b, err)
	defer func() {
		if err := node.host.Close(); err != nil {
			b.Logf("Failed to close host in cleanup: %v", err)
		}
	}()

	// Pre-populate with peer data
	numPeers := 1000
	peerIDs := make([]peer.ID, numPeers)

	for i := 0; i < numPeers; i++ {
		// Generate unique peer IDs
		peerID, _ := peer.Decode(fmt.Sprintf("12D3KooW%039d", i))
		peerIDs[i] = peerID
		node.UpdatePeerHeight(peerID, int32(i)) //nolint:gosec // used in tests
		node.peerConnTimes.Store(peerID, time.Now())
	}

	b.Run("UpdatePeerHeight", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				peerID := peerIDs[i%numPeers]
				node.UpdatePeerHeight(peerID, int32(i)) //nolint:gosec // used in tests
				i++
			}
		})
	})

	b.Run("ConnectedPeers", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = node.ConnectedPeers()
		}
	})

	b.Run("GetPeerIPs", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			peerID := peerIDs[i%numPeers]
			_ = node.GetPeerIPs(peerID)
		}
	})
}

func BenchmarkP2PNode_HelperFunctions(b *testing.B) {
	b.Run("isPrivateIP", func(b *testing.B) {
		testAddrs := []multiaddr.Multiaddr{
			multiaddr.StringCast("/ip4/10.0.0.1/tcp/4001"),
			multiaddr.StringCast("/ip4/192.168.1.1/tcp/4001"),
			multiaddr.StringCast("/ip4/8.8.8.8/tcp/4001"),
			multiaddr.StringCast("/ip4/172.16.0.1/tcp/4001"),
			multiaddr.StringCast("/ip4/1.1.1.1/tcp/4001"),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := testAddrs[i%len(testAddrs)]
			_ = isPrivateIP(addr)
		}
	})

	b.Run("extractIPFromMultiaddr", func(b *testing.B) {
		testAddrs := []multiaddr.Multiaddr{
			multiaddr.StringCast("/ip4/192.168.1.1/tcp/4001"),
			multiaddr.StringCast("/ip6/::1/tcp/4001"),
			multiaddr.StringCast("/dns4/example.com/tcp/4001"),
			multiaddr.StringCast("/ip4/10.0.0.1/tcp/4001/p2p/12D3KooWTest"),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := testAddrs[i%len(testAddrs)]
			_ = extractIPFromMultiaddr(addr)
		}
	})

	b.Run("buildAdvertiseMultiAddrs", func(b *testing.B) {
		logger := logrus.New()
		testCases := [][]string{
			{"192.168.1.1", "10.0.0.1"},
			{"example.com", "test.local:8080"},
			{"1.2.3.4:5000", "::1"},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addrs := testCases[i%len(testCases)]
			_ = buildAdvertiseMultiAddrs(logger, addrs, 4001)
		}
	})
}

// Memory allocation benchmarks
func BenchmarkP2PNode_MemoryAllocation(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	b.Run("NewNode", func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			config := Config{
				ProcessName:        fmt.Sprintf("mem-test-%d", i),
				ListenAddresses:    []string{"127.0.0.1"},
				AdvertiseAddresses: []string{"127.0.0.1"},
				Port:               3111 + i,
			}

			node, err := NewNode(ctx, logger, config)
			if err != nil {
				b.Fatal(err)
			}
			err = node.host.Close()
			require.NoError(b, err)
		}
	})

	b.Run("PeerInfo_allocation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			peers := make([]PeerInfo, 0, 100)
			for j := 0; j < 100; j++ {
				now := time.Now()
				peers = append(peers, PeerInfo{
					ID:            peer.ID(fmt.Sprintf("peer%d", j)),
					Addrs:         []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/127.0.0.1/tcp/4001")},
					CurrentHeight: int32(j), //nolint:gosec // used in tests
					ConnTime:      &now,
				})
			}
		}
	})
}
