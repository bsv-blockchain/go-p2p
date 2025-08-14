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
