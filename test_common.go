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
