package p2p

import (
	"context"
	"testing"
	"time"
)

// Common test constants
const testLocalhost = "127.0.0.1"

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	t *testing.T
}

// Debugf logs debug messages with formatted output
func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.t.Logf("[DEBUG] "+format, args...)
}

// Infof logs info messages with formatted output
func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.t.Logf("[INFO] "+format, args...)
}

// Warnf logs warning messages with formatted output
func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.t.Logf("[WARN] "+format, args...)
}

// Errorf logs error messages with formatted output
func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.t.Logf("[ERROR] "+format, args...)
}

// Fatalf logs fatal messages with formatted output and terminates the test
func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.t.Fatalf("[FATAL] "+format, args...)
}

// Test helpers to reduce code duplication

// createTestContext creates a context with timeout for testing
func createTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// createTestLogger creates a mock logger for testing
func createTestLogger(t *testing.T) *MockLogger {
	return &MockLogger{t: t}
}

// createBasicConfig creates a basic test configuration
func createBasicConfig(processName string) Config {
	return Config{
		ProcessName:     processName,
		ListenAddresses: []string{testLocalhost},
		Port:            0,
	}
}

// createAndStartTestNode creates and starts a test node with basic setup
func createAndStartTestNode(ctx context.Context, t *testing.T, logger Logger, config Config) *Node {
	t.Helper()

	node, err := NewNode(ctx, logger, config)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	if err := node.Start(ctx, nil); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	return node
}

// verifyNodeRunning verifies a node is running properly
func verifyNodeRunning(t *testing.T, node *Node, nodeName string) {
	t.Helper()

	if node == nil {
		t.Fatal("Node is nil")
	}

	if node.host == nil {
		t.Fatal("Node host is nil")
	}

	if node.host.ID() == "" {
		t.Fatal("Node host ID is empty")
	}

	addrs := node.host.Addrs()
	if len(addrs) == 0 {
		t.Fatal("Node should have listening addresses")
	}

	t.Logf("Node %s (ID: %s) listening on: %v", nodeName, node.host.ID(), addrs)
}

// setupNodeCleanup sets up proper cleanup for a test node
func setupNodeCleanup(ctx context.Context, t testing.TB, node *Node, nodeName string) {
	t.Helper()

	t.Cleanup(func() {
		if node != nil {
			if err := node.Stop(ctx); err != nil {
				t.Logf("Failed to stop %s in cleanup: %v", nodeName, err)
			}
		}
	})
}
