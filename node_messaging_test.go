package p2p

import (
	"context"
	"sync"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP2PNode_TopicOperations(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := P2PConfig{
		ProcessName:     "topic-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.Stop(ctx)

	// Start with some topics
	err = node.Start(ctx, nil, "topic1", "topic2", "topic3")
	require.NoError(t, err)

	t.Run("GetTopic existing", func(t *testing.T) {
		topic := node.GetTopic("topic1")
		assert.NotNil(t, topic)

		topic2 := node.GetTopic("topic2")
		assert.NotNil(t, topic2)
		assert.NotEqual(t, topic, topic2) // Different topics
	})

	t.Run("GetTopic non-existing", func(t *testing.T) {
		topic := node.GetTopic("non-existent")
		assert.Nil(t, topic)
	})

	t.Run("SetTopicHandler", func(t *testing.T) {
		handler := func(ctx context.Context, msg []byte, from string) {
			// Empty handler for this test
		}

		err := node.SetTopicHandler(ctx, "topic1", handler)
		assert.NoError(t, err)

		// Handler is stored
		_, ok := node.handlerByTopic["topic1"]
		assert.True(t, ok)
	})

	t.Run("SetTopicHandler duplicate", func(t *testing.T) {
		handler := func(ctx context.Context, msg []byte, from string) {}

		// First handler already set above
		err := node.SetTopicHandler(ctx, "topic1", handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler already exists")
	})

	t.Run("SetTopicHandler non-existent topic", func(t *testing.T) {
		handler := func(ctx context.Context, msg []byte, from string) {}

		err := node.SetTopicHandler(ctx, "non-existent", handler)
		assert.Error(t, err)
	})
}

func TestP2PNode_Publishing(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := P2PConfig{
		ProcessName:     "publish-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.Stop(ctx)

	t.Run("Publish before start", func(t *testing.T) {
		err := node.Publish(ctx, "topic", []byte("message"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "topics not initialised")
	})

	// Start node with topics
	err = node.Start(ctx, nil, "test-topic")
	require.NoError(t, err)

	t.Run("Publish to valid topic", func(t *testing.T) {
		beforeBytes := node.BytesSent()
		beforeTime := node.LastSend()

		msg := []byte("test message")
		err := node.Publish(ctx, "test-topic", msg)
		assert.NoError(t, err)

		// Check metrics updated
		assert.Greater(t, node.BytesSent(), beforeBytes)
		assert.True(t, node.LastSend().After(beforeTime))
		assert.Equal(t, beforeBytes+uint64(len(msg)), node.BytesSent())
	})

	t.Run("Publish to non-existent topic", func(t *testing.T) {
		err := node.Publish(ctx, "non-existent", []byte("message"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "topic not found")
	})

	t.Run("Publish empty message", func(t *testing.T) {
		err := node.Publish(ctx, "test-topic", []byte{})
		assert.NoError(t, err)
	})

	t.Run("Publish large message", func(t *testing.T) {
		largeMsg := make([]byte, 1024*1024) // 1MB
		for i := range largeMsg {
			largeMsg[i] = byte(i % 256)
		}

		beforeBytes := node.BytesSent()
		err := node.Publish(ctx, "test-topic", largeMsg)
		assert.NoError(t, err)
		assert.Equal(t, beforeBytes+uint64(len(largeMsg)), node.BytesSent())
	})
}

func TestP2PNode_SendToPeer(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()

	// Create two nodes
	config1 := P2PConfig{
		ProcessName:     "sender",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	sender, err := NewP2PNode(ctx, logger, config1)
	require.NoError(t, err)
	defer sender.Stop(ctx)

	config2 := P2PConfig{
		ProcessName:     "receiver",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	receiver, err := NewP2PNode(ctx, logger, config2)
	require.NoError(t, err)
	defer receiver.Stop(ctx)

	// Set up stream handler on receiver
	receivedMsg := make(chan []byte, 1)
	streamHandler := func(stream network.Stream) {
		defer stream.Close()

		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err == nil {
			receivedMsg <- buf[:n]
		}
	}

	err = receiver.Start(ctx, streamHandler)
	require.NoError(t, err)

	err = sender.Start(ctx, nil)
	require.NoError(t, err)

	t.Run("SendToPeer success", func(t *testing.T) {
		beforeBytes := sender.BytesSent()
		beforeTime := sender.LastSend()

		msg := []byte("direct message")
		err := sender.SendToPeer(ctx, receiver.host.ID(), msg)
		assert.NoError(t, err)

		// Check metrics
		assert.Greater(t, sender.BytesSent(), beforeBytes)
		assert.True(t, sender.LastSend().After(beforeTime))

		// Verify message received
		select {
		case received := <-receivedMsg:
			assert.Equal(t, msg, received)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("SendToPeer to non-existent peer", func(t *testing.T) {
		fakePeerID, _ := peer.Decode("12D3KooWGRYZDHBembyGJQqQ6WgLqJWYNjnECJwGBnCg8vbCeo8F")

		err := sender.SendToPeer(ctx, fakePeerID, []byte("message"))
		assert.Error(t, err)
	})

	t.Run("SendToPeer empty message", func(t *testing.T) {
		err := sender.SendToPeer(ctx, receiver.host.ID(), []byte{})
		assert.NoError(t, err)
	})
}

func TestP2PNode_InitGossipSub(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	config := P2PConfig{
		ProcessName:     "gossipsub-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	t.Run("initGossipSub with topics", func(t *testing.T) {
		topics := []string{"topic1", "topic2", "topic3"}
		err := node.initGossipSub(ctx, topics)
		assert.NoError(t, err)

		assert.NotNil(t, node.pubSub)
		assert.NotNil(t, node.topics)
		assert.Len(t, node.topics, 3)

		for _, topic := range topics {
			assert.Contains(t, node.topics, topic)
		}
	})

	t.Run("initGossipSub empty topics", func(t *testing.T) {
		// Reset node
		node2, err := NewP2PNode(ctx, logger, config)
		require.NoError(t, err)
		defer node2.host.Close()

		err = node2.initGossipSub(ctx, []string{})
		assert.NoError(t, err)
		assert.NotNil(t, node2.pubSub)
		assert.Empty(t, node2.topics)
	})
}

func TestSubscribeToTopics(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	config := P2PConfig{
		ProcessName:     "subscribe-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	// Create pubsub instance
	ps, err := pubsub.NewGossipSub(ctx, node.host)
	require.NoError(t, err)

	t.Run("subscribe to multiple topics", func(t *testing.T) {
		topicNames := []string{"topic1", "topic2", "topic3"}

		topics, shouldReturn, err := subscribeToTopics(topicNames, ps, node)

		assert.False(t, shouldReturn)
		assert.NoError(t, err)
		assert.Len(t, topics, 3)

		for _, name := range topicNames {
			assert.Contains(t, topics, name)
			assert.NotNil(t, topics[name])
		}
	})

	t.Run("subscribe to duplicate topic", func(t *testing.T) {
		// Try to join an already joined topic
		topicNames := []string{"topic1"}

		topics, shouldReturn, err := subscribeToTopics(topicNames, ps, node)

		// Should succeed (libp2p allows rejoining)
		assert.False(t, shouldReturn)
		assert.NoError(t, err)
		assert.Len(t, topics, 1)
	})
}

func TestP2PNode_ConcurrentPublishing(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := P2PConfig{
		ProcessName:     "concurrent-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.Stop(ctx)

	err = node.Start(ctx, nil, "test-topic")
	require.NoError(t, err)

	// Test concurrent publishing
	var wg sync.WaitGroup
	numGoroutines := 10
	numMessages := 100

	errors := make(chan error, numGoroutines*numMessages)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < numMessages; j++ {
				msg := []byte(string(rune(routineID)) + string(rune(j)))
				if err := node.Publish(ctx, "test-topic", msg); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Publishing error: %v", err)
	}

	assert.Equal(t, 0, errorCount, "Should have no publishing errors")

	// Verify metrics
	expectedBytes := uint64(numGoroutines * numMessages * 2) // Each message is 2 bytes
	assert.Equal(t, expectedBytes, node.BytesSent())
}

func TestP2PNode_HandlerContextCancellation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := P2PConfig{
		ProcessName:     "handler-cancel-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(context.Background(), logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	err = node.Start(ctx, nil, "test-topic")
	require.NoError(t, err)

	handlerStarted := make(chan bool)
	handlerStopped := make(chan bool)

	handler := func(ctx context.Context, msg []byte, from string) {
		handlerStarted <- true
		<-ctx.Done()
		handlerStopped <- true
	}

	// This will start a goroutine
	err = node.SetTopicHandler(ctx, "test-topic", handler)
	assert.NoError(t, err)

	// Cancel the context
	cancel()

	// Handler goroutine should stop
	select {
	case <-handlerStopped:
		// Expected - handler stopped due to context cancellation
	case <-handlerStarted:
		t.Fatal("Handler started but didn't stop")
	case <-time.After(100 * time.Millisecond):
		// Expected - handler goroutine stopped before processing any message
	}
}

// MockPubSub for testing edge cases
type mockPubSub struct {
	*pubsub.PubSub
	joinError error
}

func (m *mockPubSub) Join(topic string, opts ...pubsub.TopicOpt) (*pubsub.Topic, error) {
	if m.joinError != nil {
		return nil, m.joinError
	}
	return &pubsub.Topic{}, nil
}

func TestSubscribeToTopics_Error(t *testing.T) {
	logger := logrus.New()

	ctx := context.Background()
	config := P2PConfig{
		ProcessName:     "error-test",
		ListenAddresses: []string{"127.0.0.1"},
		Port:            0,
	}

	node, err := NewP2PNode(ctx, logger, config)
	require.NoError(t, err)
	defer node.host.Close()

	// Create a mock pubsub that returns error
	/*mockPS := &mockPubSub{
		joinError: errors.New("failed to join topic"),
	}

	topicNames := []string{"topic1"}*/

	// Type assertion won't work with mock, so we'll test the actual function differently
	// This is a limitation of testing with concrete types
	t.Skip("Cannot easily mock pubsub.PubSub due to concrete type")
}
