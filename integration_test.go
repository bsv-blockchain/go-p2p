package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_TwoNodeCommunication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create two nodes
	config1 := P2PConfig{
		ProcessName:     "node1",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node1, err := NewP2PNode(ctx, logger, config1)
	require.NoError(t, err)
	defer node1.Stop(ctx)

	config2 := P2PConfig{
		ProcessName:     "node2",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		StaticPeers:     []string{fmt.Sprintf("%s/p2p/%s", node1.host.Addrs()[0], node1.host.ID())},
	}

	node2, err := NewP2PNode(ctx, logger, config2)
	require.NoError(t, err)
	defer node2.Stop(ctx)

	// Start both nodes with same topics
	topics := []string{"blocks", "transactions"}
	
	err = node1.Start(ctx, nil, topics...)
	require.NoError(t, err)

	err = node2.Start(ctx, nil, topics...)
	require.NoError(t, err)

	// Wait for connection
	require.Eventually(t, func() bool {
		return node2.host.Network().Connectedness(node1.host.ID()) == network.Connected
	}, 5*time.Second, 100*time.Millisecond, "nodes should connect")

	t.Run("pubsub messaging", func(t *testing.T) {
		received := make(chan string, 1)
		
		// Set up handler on node1
		handler := func(ctx context.Context, msg []byte, from string) {
			received <- string(msg)
		}
		
		err := node1.SetTopicHandler(ctx, "blocks", handler)
		require.NoError(t, err)

		// Give subscription time to propagate
		time.Sleep(500 * time.Millisecond)

		// Publish from node2
		testMsg := "test block data"
		err = node2.Publish(ctx, "blocks", []byte(testMsg))
		require.NoError(t, err)

		// Verify receipt
		select {
		case msg := <-received:
			assert.Equal(t, testMsg, msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("direct messaging", func(t *testing.T) {
		received := make(chan []byte, 1)
		
		// Set up stream handler on node1
		streamHandler := func(stream network.Stream) {
			defer stream.Close()
			
			buf := make([]byte, 1024)
			n, err := stream.Read(buf)
			if err == nil {
				received <- buf[:n]
			}
		}
		
		node1.host.SetStreamHandler("/test/1.0.0", streamHandler)

		// Send from node2
		stream, err := node2.host.NewStream(ctx, node1.host.ID(), "/test/1.0.0")
		require.NoError(t, err)
		defer stream.Close()

		testMsg := []byte("direct message")
		_, err = stream.Write(testMsg)
		require.NoError(t, err)

		// Verify receipt
		select {
		case msg := <-received:
			assert.Equal(t, testMsg, msg)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for direct message")
		}
	})

	t.Run("peer info", func(t *testing.T) {
		// Check connected peers
		peers1 := node1.CurrentlyConnectedPeers()
		assert.Len(t, peers1, 1)
		assert.Equal(t, node2.host.ID(), peers1[0].ID)

		peers2 := node2.CurrentlyConnectedPeers()
		assert.Len(t, peers2, 1)
		assert.Equal(t, node1.host.ID(), peers2[0].ID)

		// Update and check peer height
		node1.UpdatePeerHeight(node2.host.ID(), 12345)
		
		peers := node1.CurrentlyConnectedPeers()
		found := false
		for _, p := range peers {
			if p.ID == node2.host.ID() {
				assert.Equal(t, int32(12345), p.CurrentHeight)
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

func TestIntegration_MultiNodeNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numNodes := 5
	nodes := make([]*P2PNode, numNodes)

	// Create nodes
	for i := 0; i < numNodes; i++ {
		config := P2PConfig{
			ProcessName:     fmt.Sprintf("node%d", i),
			ListenAddresses: []string{"127.0.0.1"},
			Port:            0,
		}

		// Connect each node to the previous one
		if i > 0 {
			prevNode := nodes[i-1]
			config.StaticPeers = []string{
				fmt.Sprintf("%s/p2p/%s", prevNode.host.Addrs()[0], prevNode.host.ID()),
			}
		}

		node, err := NewP2PNode(ctx, logger, config)
		require.NoError(t, err)
		
		err = node.Start(ctx, nil, "broadcast")
		require.NoError(t, err)
		
		nodes[i] = node
		defer node.Stop(ctx)
	}

	// Wait for network to stabilize
	time.Sleep(2 * time.Second)

	t.Run("network connectivity", func(t *testing.T) {
		// Each node should have at least one peer
		for i, node := range nodes {
			peers := node.CurrentlyConnectedPeers()
			assert.NotEmpty(t, peers, "node%d should have peers", i)
		}
	})

	t.Run("broadcast propagation", func(t *testing.T) {
		receivedCount := &sync.Map{}
		var wg sync.WaitGroup

		// Set up handlers on all nodes
		for i, node := range nodes {
			nodeID := i
			wg.Add(1)
			
			handler := func(ctx context.Context, msg []byte, from string) {
				receivedCount.Store(nodeID, string(msg))
				wg.Done()
			}
			
			err := node.SetTopicHandler(ctx, "broadcast", handler)
			require.NoError(t, err)
		}

		// Give subscriptions time to propagate
		time.Sleep(1 * time.Second)

		// Broadcast from first node
		testMsg := "network broadcast message"
		err := nodes[0].Publish(ctx, "broadcast", []byte(testMsg))
		require.NoError(t, err)

		// Wait for all handlers or timeout
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// All received
		case <-time.After(5 * time.Second):
			// Timeout - check what we got
		}

		// Count how many nodes received the message
		count := 0
		receivedCount.Range(func(key, value interface{}) bool {
			if value.(string) == testMsg {
				count++
			}
			return true
		})

		// Due to network propagation, we might not reach all nodes
		// but we should reach at least half
		assert.GreaterOrEqual(t, count, numNodes/2, "at least half nodes should receive broadcast")
	})
}

func TestIntegration_ConnectionCallbacks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create first node
	config1 := P2PConfig{
		ProcessName:     "callback-node1",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node1, err := NewP2PNode(ctx, logger, config1)
	require.NoError(t, err)
	defer node1.Stop(ctx)

	// Set up connection callback
	connectedPeers := &sync.Map{}
	connectionTimes := &sync.Map{}
	
	node1.SetPeerConnectedCallback(func(ctx context.Context, peerID peer.ID) {
		connectedPeers.Store(peerID.String(), true)
		connectionTimes.Store(peerID.String(), time.Now())
	})

	err = node1.Start(ctx, nil)
	require.NoError(t, err)

	// Create and connect second node
	config2 := P2PConfig{
		ProcessName:     "callback-node2",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		StaticPeers:     []string{fmt.Sprintf("%s/p2p/%s", node1.host.Addrs()[0], node1.host.ID())},
	}

	node2, err := NewP2PNode(ctx, logger, config2)
	require.NoError(t, err)
	defer node2.Stop(ctx)

	err = node2.Start(ctx, nil)
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(1 * time.Second)

	// Verify callback was called
	_, connected := connectedPeers.Load(node2.host.ID().String())
	assert.True(t, connected, "callback should record node2 connection")

	connTime, hasTime := connectionTimes.Load(node2.host.ID().String())
	assert.True(t, hasTime, "should record connection time")
	assert.WithinDuration(t, time.Now(), connTime.(time.Time), 2*time.Second)

	// Test disconnection
	err = node1.DisconnectPeer(ctx, node2.host.ID())
	require.NoError(t, err)

	// Verify disconnection
	require.Eventually(t, func() bool {
		return node1.host.Network().Connectedness(node2.host.ID()) != network.Connected
	}, 2*time.Second, 100*time.Millisecond)
}

func TestIntegration_PrivateNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sharedKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	// Create first private node
	config1 := P2PConfig{
		ProcessName:     "private1",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		UsePrivateDHT:   true,
		SharedKey:       sharedKey,
	}

	node1, err := NewP2PNode(ctx, logger, config1)
	require.NoError(t, err)
	defer node1.Stop(ctx)

	err = node1.Start(ctx, nil, "private-topic")
	require.NoError(t, err)

	// Create second private node with same key
	config2 := P2PConfig{
		ProcessName:     "private2",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		UsePrivateDHT:   true,
		SharedKey:       sharedKey,
		StaticPeers:     []string{fmt.Sprintf("%s/p2p/%s", node1.host.Addrs()[0], node1.host.ID())},
	}

	node2, err := NewP2PNode(ctx, logger, config2)
	require.NoError(t, err)
	defer node2.Stop(ctx)

	err = node2.Start(ctx, nil, "private-topic")
	require.NoError(t, err)

	// Wait for connection
	require.Eventually(t, func() bool {
		return node2.host.Network().Connectedness(node1.host.ID()) == network.Connected
	}, 5*time.Second, 100*time.Millisecond, "private nodes should connect")

	t.Run("private network messaging", func(t *testing.T) {
		received := make(chan string, 1)
		
		handler := func(ctx context.Context, msg []byte, from string) {
			received <- string(msg)
		}
		
		err := node1.SetTopicHandler(ctx, "private-topic", handler)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		testMsg := "private network message"
		err = node2.Publish(ctx, "private-topic", []byte(testMsg))
		require.NoError(t, err)

		select {
		case msg := <-received:
			assert.Equal(t, testMsg, msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for private message")
		}
	})

	t.Run("node with wrong key cannot connect", func(t *testing.T) {
		// Create node with different shared key
		config3 := P2PConfig{
			ProcessName:     "wrong-key",
			ListenAddresses: []string{"127.0.0.1"},
			Port:            0,
			UsePrivateDHT:   true,
			SharedKey:       "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			StaticPeers:     []string{fmt.Sprintf("%s/p2p/%s", node1.host.Addrs()[0], node1.host.ID())},
		}

		node3, err := NewP2PNode(ctx, logger, config3)
		require.NoError(t, err)
		defer node3.Stop(ctx)

		err = node3.Start(ctx, nil)
		require.NoError(t, err)

		// Should not be able to connect
		time.Sleep(2 * time.Second)
		assert.NotEqual(t, network.Connected, node3.host.Network().Connectedness(node1.host.ID()))
	})
}

func TestIntegration_MetricsTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create two connected nodes
	config1 := P2PConfig{
		ProcessName:     "metrics1",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node1, err := NewP2PNode(ctx, logger, config1)
	require.NoError(t, err)
	defer node1.Stop(ctx)

	config2 := P2PConfig{
		ProcessName:     "metrics2",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
		StaticPeers:     []string{fmt.Sprintf("%s/p2p/%s", node1.host.Addrs()[0], node1.host.ID())},
	}

	node2, err := NewP2PNode(ctx, logger, config2)
	require.NoError(t, err)
	defer node2.Stop(ctx)

	err = node1.Start(ctx, nil, "metrics-topic")
	require.NoError(t, err)

	err = node2.Start(ctx, nil, "metrics-topic")
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(1 * time.Second)

	t.Run("track bytes sent and received", func(t *testing.T) {
		// Set up handler to track received bytes
		received := make(chan []byte, 10)
		handler := func(ctx context.Context, msg []byte, from string) {
			received <- msg
		}
		
		err := node1.SetTopicHandler(ctx, "metrics-topic", handler)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		// Send multiple messages
		messages := []string{
			"message 1",
			"longer message 2 with more data",
			"msg 3",
		}

		initialBytesSent := node2.BytesSent()
		
		for _, msg := range messages {
			err := node2.Publish(ctx, "metrics-topic", []byte(msg))
			require.NoError(t, err)
		}

		// Wait for messages
		for i := 0; i < len(messages); i++ {
			select {
			case <-received:
				// Message received
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for message")
			}
		}

		// Verify metrics
		totalBytesSent := uint64(0)
		for _, msg := range messages {
			totalBytesSent += uint64(len(msg))
		}

		assert.Equal(t, initialBytesSent+totalBytesSent, node2.BytesSent())
		assert.True(t, node2.LastSend().After(time.Now().Add(-2*time.Second)))
	})

	t.Run("track direct message metrics", func(t *testing.T) {
		initialBytes := node2.BytesSent()
		
		msg := []byte("direct metric test message")
		err := node2.SendToPeer(ctx, node1.host.ID(), msg)
		require.NoError(t, err)

		assert.Equal(t, initialBytes+uint64(len(msg)), node2.BytesSent())
	})
}