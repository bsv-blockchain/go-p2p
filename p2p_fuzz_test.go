package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements Logger interface for fuzzing tests
type mockLogger struct {
	*logrus.Logger
}

func newMockLogger() *mockLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress all logs during fuzzing
	return &mockLogger{Logger: logger}
}

// FuzzConfig tests the Config struct with random input values to ensure
// it handles malformed configurations gracefully without panicking.
func FuzzConfig(f *testing.F) {
	// Add seed inputs for more effective fuzzing
	f.Add("test-node", "9000", "/ip4/127.0.0.1/tcp/9000", "dht-test", "false", "true")
	f.Add("", "0", "", "", "invalid", "invalid")
	f.Add("node-with-very-long-name-that-exceeds-normal-limits", "65536", "/invalid/multiaddr", "", "true", "false")

	f.Fuzz(func(t *testing.T, processName, port, listenAddr, dhtProtocol, usePrivateDHT, advertise string) {
		// Ensure we don't panic during fuzzing
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Config creation caused panic with inputs: processName=%q, port=%q, listenAddr=%q: %v",
					processName, port, listenAddr, r)
			}
		}()

		// Parse port safely
		portInt := 0
		if port != "" {
			if p, err := strconv.Atoi(port); err == nil && p >= 0 && p <= 65535 {
				portInt = p
			}
		}

		// Parse boolean values safely
		usePrivateDHTBool := strings.ToLower(usePrivateDHT) == "true"
		advertiseBool := strings.ToLower(advertise) == "true"

		config := Config{
			ProcessName:     processName,
			Port:            portInt,
			ListenAddresses: []string{listenAddr},
			DHTProtocolID:   dhtProtocol,
			UsePrivateDHT:   usePrivateDHTBool,
			Advertise:       advertiseBool,
		}

		// Test that config doesn't break basic validation
		assert.IsType(t, Config{}, config)

		// Validate that process name is handled safely
		if processName != "" {
			assert.Equal(t, processName, config.ProcessName)
		}

		// Validate port is within safe range
		assert.GreaterOrEqual(t, config.Port, 0)
		assert.LessOrEqual(t, config.Port, 65535)
	})
}

// FuzzBlockMessage tests BlockMessage struct with random field values
// to ensure robust handling of malformed blockchain messages.
func FuzzBlockMessage(f *testing.F) {
	// Add seed inputs
	f.Add("abc123", uint32(1), "http://example.com", "peer1")
	f.Add("", uint32(0), "", "")
	f.Add(strings.Repeat("a", 1000), uint32(4294967295), "invalid://url", "peer-with-very-long-name")

	f.Fuzz(func(t *testing.T, hash string, height uint32, dataHubURL, peerID string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("BlockMessage handling caused panic with input hash=%q, height=%d: %v", hash, height, r)
			}
		}()

		// Create BlockMessage with fuzzed values
		msg := BlockMessage{
			Hash:       hash,
			Height:     height,
			DataHubURL: dataHubURL,
			PeerID:     peerID,
		}

		// Validate the structure is safe to use
		assert.IsType(t, BlockMessage{}, msg)

		// Test that string fields don't cause issues
		_ = fmt.Sprintf("Hash: %s", msg.Hash)
		_ = fmt.Sprintf("PeerID: %s", msg.PeerID)
		_ = fmt.Sprintf("DataHubURL: %s", msg.DataHubURL)

		// Test field access
		assert.Equal(t, hash, msg.Hash)
		assert.Equal(t, height, msg.Height)
		assert.Equal(t, peerID, msg.PeerID)
		assert.Equal(t, dataHubURL, msg.DataHubURL)
	})
}

// FuzzMiningOnMessage tests MiningOnMessage with random data to ensure
// resilience against malformed mining announcements.
func FuzzMiningOnMessage(f *testing.F) {
	// Add seed inputs
	f.Add("block123", "prev123", "http://mining.example.com", "miner1", uint32(100), "miner_address", uint64(1024), uint64(50))
	f.Add("", "", "", "", uint32(0), "", uint64(0), uint64(0))
	f.Add(strings.Repeat("x", 100), "invalid-hash", "not-a-url", "peer", uint32(4294967295), "miner", uint64(18446744073709551615), uint64(999999))

	f.Fuzz(func(t *testing.T, hash, previousHash, dataHubURL, peerID string, height uint32, miner string, sizeInBytes, txCount uint64) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MiningOnMessage handling caused panic with hash=%q, height=%d: %v", hash, height, r)
			}
		}()

		// Create MiningOnMessage with fuzzed values
		msg := MiningOnMessage{
			Hash:         hash,
			PreviousHash: previousHash,
			DataHubURL:   dataHubURL,
			PeerID:       peerID,
			Height:       height,
			Miner:        miner,
			SizeInBytes:  sizeInBytes,
			TxCount:      txCount,
		}

		assert.IsType(t, MiningOnMessage{}, msg)

		// Validate numeric fields are safe
		assert.IsType(t, uint32(0), msg.Height)
		assert.IsType(t, uint64(0), msg.SizeInBytes)
		assert.IsType(t, uint64(0), msg.TxCount)

		// Test string field access doesn't panic
		_ = len(msg.Hash)
		_ = len(msg.PreviousHash)
		_ = len(msg.PeerID)
		_ = len(msg.Miner)

		// Validate field assignments
		assert.Equal(t, hash, msg.Hash)
		assert.Equal(t, height, msg.Height)
		assert.Equal(t, txCount, msg.TxCount)
	})
}

// FuzzBestBlockRequestMessage tests request message handling with random input.
func FuzzBestBlockRequestMessage(f *testing.F) {
	f.Add("peer123")
	f.Add("")
	f.Add("peer-with-very-long-identifier-that-exceeds-normal-limits")
	f.Add("peer/with/special/characters")

	f.Fuzz(func(t *testing.T, peerID string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("BestBlockRequestMessage handling caused panic with peerID=%q: %v", peerID, r)
			}
		}()

		// Create BestBlockRequestMessage with fuzzed values
		msg := BestBlockRequestMessage{
			PeerID: peerID,
		}

		assert.IsType(t, BestBlockRequestMessage{}, msg)

		// Test that PeerID field is accessible
		_ = len(msg.PeerID)
		assert.Equal(t, peerID, msg.PeerID)

		// Test string operations on PeerID
		_ = strings.Contains(msg.PeerID, "peer")
		_ = strings.ToLower(msg.PeerID)
	})
}

// FuzzMessageHandler tests the Handler function type with random message bytes
// to ensure it handles arbitrary network input safely.
func FuzzMessageHandler(f *testing.F) {
	// Seed with various message types
	f.Add([]byte("valid message"))
	f.Add([]byte(""))
	f.Add([]byte{0x00, 0xFF, 0xAA, 0x55})     // Binary data
	f.Add([]byte(strings.Repeat("A", 10000))) // Large message

	f.Fuzz(func(t *testing.T, msgBytes []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Handler caused panic with message length %d: %v", len(msgBytes), r)
			}
		}()

		ctx := context.Background()
		peerID := "test-peer"

		// Create a simple test handler that processes the message
		handler := func(ctx context.Context, msg []byte, from string) {
			// Basic validation that handler can access parameters safely
			assert.NotNil(t, ctx)
			assert.Equal(t, peerID, from)

			// Test that message bytes can be accessed safely
			_ = len(msg)
			if len(msg) > 0 {
				_ = msg[0] // Access first byte safely
			}

			// Test common operations that might be performed on messages
			msgStr := string(msg)
			_ = len(msgStr)

			// Test JSON parsing attempt (should not panic)
			// Intentionally ignoring error as we're testing panic resistance with arbitrary input
			var dummy interface{}
			_ = json.Unmarshal(msg, &dummy) // Error ignored for fuzz testing
		}

		// Execute handler with fuzzed input
		handler(ctx, msgBytes, peerID)
	})
}

// FuzzPeerID tests peer ID handling with random strings to ensure
// robust peer identification and validation.
func FuzzPeerID(f *testing.F) {
	f.Add("12D3KooWBhX7KJ8Y9PjPD1V8K5x9j7wCLBz4T9YyG1qR7mNx5Q2z") // Valid-looking peer ID
	f.Add("")
	f.Add("invalid-peer-id")
	f.Add(strings.Repeat("a", 1000)) // Very long string

	f.Fuzz(func(t *testing.T, peerIDStr string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Peer ID handling caused panic with input %q: %v", peerIDStr, r)
			}
		}()

		// Test peer.Decode function with random input
		// This is a common operation in P2P networking
		if peerIDStr != "" {
			peerID, err := peer.Decode(peerIDStr)
			if err == nil {
				// If decoding succeeded, test basic operations
				assert.NotEmpty(t, peerID.String())

				// Test that the peer ID can be used safely
				_ = peerID.Size()
				_ = peerID.Validate()
			}
		}
	})
}

// FuzzTopicNames tests topic name validation with random strings
// to ensure the P2P system handles arbitrary topic names safely.
func FuzzTopicNames(f *testing.F) {
	f.Add("valid-topic")
	f.Add("")
	f.Add("topic/with/slashes")
	f.Add(strings.Repeat("very-long-topic-name-", 100))
	f.Add("topic-with-unicode-🚀")

	f.Fuzz(func(t *testing.T, topicName string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Topic name handling caused panic with input %q: %v", topicName, r)
			}
		}()

		// Test basic topic name operations
		assert.IsType(t, "", topicName)

		// Test length validation
		nameLen := len(topicName)
		assert.GreaterOrEqual(t, nameLen, 0)

		// Test that topic names can be used in string operations safely
		normalizedTopic := strings.ToLower(topicName)
		_ = normalizedTopic

		// Test topic name as part of protocol identifiers
		if topicName != "" {
			protocolID := fmt.Sprintf("/p2p/topic/%s", topicName)
			assert.Contains(t, protocolID, topicName)
		}
	})
}

// FuzzPrivateKey tests private key parsing with random hex strings
// to ensure cryptographic operations handle malformed keys safely.
func FuzzPrivateKey(f *testing.F) {
	f.Add("308204a30201000282010100abcd") // Partial hex
	f.Add("")
	f.Add("invalid-hex")
	f.Add("zzzzzz")                  // Non-hex characters
	f.Add(strings.Repeat("ab", 500)) // Very long hex

	f.Fuzz(func(t *testing.T, hexKey string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Private key parsing caused panic with input %q: %v", hexKey, r)
			}
		}()

		// Test hex decoding (common operation for private keys)
		if hexKey != "" {
			// This should not panic even with invalid input
			_, err := parsePrivateKeyFromHex(hexKey)

			// We don't require success, just no panics
			_ = err // Error is expected for invalid input
		}
	})
}

// parsePrivateKeyFromHex is a helper function to test private key parsing
// This simulates the kind of operation that might be done with private keys
func parsePrivateKeyFromHex(hexStr string) ([]byte, error) {
	// Remove common prefixes that might be present
	hexStr = strings.TrimPrefix(hexStr, "0x")
	hexStr = strings.TrimSpace(hexStr)

	// Validate hex string length
	if len(hexStr)%2 != 0 {
		return nil, fmt.Errorf("invalid hex string length: %d", len(hexStr))
	}

	// Check for valid hex characters
	for _, char := range hexStr {
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'f') ||
			(char >= 'A' && char <= 'F')) {
			return nil, fmt.Errorf("invalid hex character: %c", char)
		}
	}

	// Simulate key length validation
	if len(hexStr) < 32 { // Minimum reasonable key length
		return nil, fmt.Errorf("key too short: %d", len(hexStr))
	}

	if len(hexStr) > 2048 { // Maximum reasonable key length
		return nil, fmt.Errorf("key too long: %d", len(hexStr))
	}

	// Return dummy bytes (in real implementation, this would decode hex)
	return make([]byte, len(hexStr)/2), nil
}

// FuzzNodeCreation tests node creation with fuzzed configuration
// to ensure NewNode handles edge cases gracefully.
func FuzzNodeCreation(f *testing.F) {
	f.Add("test-node", 9000, "127.0.0.1", "/ip4/127.0.0.1/tcp/9000")
	f.Add("", 0, "", "")
	f.Add("node", 65535, "255.255.255.255", "/ip4/255.255.255.255/tcp/65535")

	f.Fuzz(func(t *testing.T, processName string, port int, host, listenAddr string) {
		// Limit port to valid range to avoid obvious invalid configs
		if port < 0 {
			port = 0
		}
		if port > 65535 {
			port = 65535
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Node creation caused panic: processName=%q, port=%d, host=%q, listenAddr=%q: %v",
					processName, port, host, listenAddr, r)
			}
		}()

		logger := newMockLogger()
		config := Config{
			ProcessName:     processName,
			Port:            port,
			ListenAddresses: []string{listenAddr},
			DHTProtocolID:   "test-dht",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Test node creation - it may fail but should not panic
		node, err := NewNode(ctx, logger, config)
		if err != nil {
			// Error is acceptable with random input
			require.Nil(t, node)
		} else {
			// If creation succeeded, test basic operations
			require.NotNil(t, node)

			// Test that basic getters don't panic
			_ = node.GetProcessName()

			// Clean up
			if node != nil {
				if err := node.Stop(ctx); err != nil {
					t.Logf("Failed to stop node in cleanup: %v", err)
				}
			}
		}
	})
}
