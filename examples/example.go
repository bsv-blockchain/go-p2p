// Package main demonstrates basic usage of the go-p2p library
package main

import (
	"context"
	"log"
	"time"

	"github.com/bsv-blockchain/go-p2p"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Configure P2P node
	config := p2p.Config{
		ProcessName:     "example-node",
		ListenAddresses: []string{"0.0.0.0"},
		Port:            8333,
		Advertise:       true,
		UsePrivateDHT:   false, // Use public DHT for this example
	}

	// Create P2P node
	node, err := p2p.NewP2PNode(ctx, logger, config)
	if err != nil {
		log.Fatalf("Failed to create P2P node: %v", err)
	}

	// Set up a simple message handler for the "example" topic
	messageHandler := func(_ context.Context, msg []byte, from string) {
		logger.Infof("Received message from %s: %s", from, string(msg))
	}

	// Start the node with the example topic
	topicName := "example-topic"
	if err := node.Start(ctx, nil, topicName); err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}

	// Set the message handler for our topic
	if err := node.SetTopicHandler(ctx, topicName, messageHandler); err != nil {
		log.Fatalf("Failed to set topic handler: %v", err)
	}

	logger.Infof("P2P node started with ID: %s", node.HostID().String())
	logger.Infof("Listening for messages on topic: %s", topicName)

	// Publish a test message after a short delay
	go func() {
		time.Sleep(5 * time.Second)
		testMessage := []byte("Hello from go-p2p example!")
		if err := node.Publish(ctx, topicName, testMessage); err != nil {
			logger.Errorf("Failed to publish message: %v", err)
		} else {
			logger.Infof("Published test message: %s", testMessage)
		}
	}()

	// Keep the node running and display peer info periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			peers := node.CurrentlyConnectedPeers()
			logger.Infof("Currently connected to %d peers", len(peers))

			stats := map[string]interface{}{
				"bytes_sent":     node.BytesSent(),
				"bytes_received": node.BytesReceived(),
				"last_send":      node.LastSend(),
				"last_recv":      node.LastRecv(),
			}
			logger.Infof("Node statistics: %+v", stats)
		}
	}
}
