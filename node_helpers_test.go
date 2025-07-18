package p2p

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePrivateKey(t *testing.T) {
	ctx := context.Background()

	// Test successful key generation
	t.Run("successful generation", func(t *testing.T) {
		key, err := generatePrivateKey(ctx)
		require.NoError(t, err)
		require.NotNil(t, key)

		// Verify it's a valid Ed25519 key
		pubKey := (*key).GetPublic()
		assert.NotNil(t, pubKey)
		assert.Equal(t, crypto.Ed25519, (*key).Type())
	})

	// Test multiple generations produce different keys
	t.Run("unique keys", func(t *testing.T) {
		key1, err1 := generatePrivateKey(ctx)
		require.NoError(t, err1)

		key2, err2 := generatePrivateKey(ctx)
		require.NoError(t, err2)

		// Compare raw bytes to ensure they're different
		bytes1, _ := (*key1).Raw()
		bytes2, _ := (*key2).Raw()
		assert.NotEqual(t, bytes1, bytes2)
	})
}

func TestDecodeHexEd25519PrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		hexKey  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid hex key",
			hexKey:  generateValidHexKey(t),
			wantErr: false,
		},
		{
			name:    "invalid hex string",
			hexKey:  "not-hex",
			wantErr: true,
			errMsg:  "encoding/hex",
		},
		{
			name:    "empty hex string",
			hexKey:  "",
			wantErr: true,
		},
		{
			name:    "invalid key format",
			hexKey:  "abcdef1234567890",
			wantErr: true,
		},
		{
			name:    "wrong key type",
			hexKey:  hex.EncodeToString([]byte("wrong-key-format-that-is-long-enough")),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := decodeHexEd25519PrivateKey(tt.hexKey)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, key)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
				assert.Equal(t, crypto.Ed25519, (*key).Type())
			}
		})
	}
}

func TestBuildAdvertiseMultiAddrs(t *testing.T) {
	tests := []struct {
		name         string
		addrs        []string
		defaultPort  int
		expectedLen  int
		validateFunc func(t *testing.T, result []multiaddr.Multiaddr)
	}{
		{
			name:        "simple IP addresses",
			addrs:       []string{"192.168.1.1", "10.0.0.1"},
			defaultPort: 4001,
			expectedLen: 2,
			validateFunc: func(t *testing.T, result []multiaddr.Multiaddr) {
				assert.Contains(t, result[0].String(), "192.168.1.1")
				assert.Contains(t, result[0].String(), "4001")
				assert.Contains(t, result[1].String(), "10.0.0.1")
			},
		},
		{
			name:        "IP with custom port",
			addrs:       []string{"192.168.1.1:5001", "10.0.0.1:6001"},
			defaultPort: 4001,
			expectedLen: 2,
			validateFunc: func(t *testing.T, result []multiaddr.Multiaddr) {
				assert.Contains(t, result[0].String(), "5001")
				assert.Contains(t, result[1].String(), "6001")
			},
		},
		{
			name:        "DNS addresses",
			addrs:       []string{"example.com", "test.local:8080"},
			defaultPort: 4001,
			expectedLen: 2,
			validateFunc: func(t *testing.T, result []multiaddr.Multiaddr) {
				assert.Contains(t, result[0].String(), "/dns4/example.com/tcp/4001")
				assert.Contains(t, result[1].String(), "/dns4/test.local/tcp/8080")
			},
		},
		{
			name:        "mixed addresses",
			addrs:       []string{"192.168.1.1", "example.com:9000", "::1"},
			defaultPort: 4001,
			expectedLen: 3,
		},
		{
			name:        "invalid addresses skipped",
			addrs:       []string{"192.168.1.1", "invalid:port:format", "256.256.256.256"},
			defaultPort: 4001,
			expectedLen: 1,
		},
		{
			name:        "empty input",
			addrs:       []string{},
			defaultPort: 4001,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAdvertiseMultiAddrs(tt.addrs, tt.defaultPort)
			assert.Len(t, result, tt.expectedLen)
			
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestGetIPFromMultiaddr(t *testing.T) {
	tests := []struct {
		name     string
		addrStr  string
		expected string
		wantErr  bool
	}{
		{
			name:     "IPv4 address",
			addrStr:  "/ip4/192.168.1.1/tcp/4001",
			expected: "192.168.1.1",
			wantErr:  false,
		},
		{
			name:     "IPv6 address",
			addrStr:  "/ip6/::1/tcp/4001",
			expected: "::1",
			wantErr:  false,
		},
		{
			name:     "DNS4 address",
			addrStr:  "/dns4/example.com/tcp/4001",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "DNS6 address",
			addrStr:  "/dns6/example.com/tcp/4001",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "no IP component",
			addrStr:  "/tcp/4001",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "complex multiaddr with peer ID",
			addrStr:  "/ip4/192.168.1.1/tcp/4001/p2p/12D3KooWTest",
			expected: "192.168.1.1",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := multiaddr.NewMultiaddr(tt.addrStr)
			require.NoError(t, err)

			ip, err := getIPFromMultiaddr(maddr)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, ip)
			}
		})
	}
}

func TestExtractIPFromMultiaddr(t *testing.T) {
	tests := []struct {
		name     string
		addrStr  string
		expected string
	}{
		{
			name:     "IPv4 extraction",
			addrStr:  "/ip4/192.168.1.1/tcp/4001",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv6 extraction",
			addrStr:  "/ip6/2001:db8::1/tcp/4001",
			expected: "2001:db8::1",
		},
		{
			name:     "no IP in address",
			addrStr:  "/tcp/4001",
			expected: "",
		},
		{
			name:     "complex path with IP",
			addrStr:  "/ip4/10.0.0.1/tcp/4001/p2p/12D3KooWTest/http",
			expected: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := multiaddr.NewMultiaddr(tt.addrStr)
			require.NoError(t, err)

			ip := extractIPFromMultiaddr(maddr)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name      string
		addrStr   string
		isPrivate bool
	}{
		// Private IPs - RFC 1918
		{
			name:      "10.x.x.x private",
			addrStr:   "/ip4/10.0.0.1/tcp/4001",
			isPrivate: true,
		},
		{
			name:      "172.16.x.x private",
			addrStr:   "/ip4/172.16.0.1/tcp/4001",
			isPrivate: true,
		},
		{
			name:      "172.31.x.x private",
			addrStr:   "/ip4/172.31.255.255/tcp/4001",
			isPrivate: true,
		},
		{
			name:      "192.168.x.x private",
			addrStr:   "/ip4/192.168.1.1/tcp/4001",
			isPrivate: true,
		},
		{
			name:      "localhost",
			addrStr:   "/ip4/127.0.0.1/tcp/4001",
			isPrivate: true,
		},
		// Public IPs
		{
			name:      "public IP 8.8.8.8",
			addrStr:   "/ip4/8.8.8.8/tcp/4001",
			isPrivate: false,
		},
		{
			name:      "public IP 1.1.1.1",
			addrStr:   "/ip4/1.1.1.1/tcp/4001",
			isPrivate: false,
		},
		{
			name:      "172.15.x.x public",
			addrStr:   "/ip4/172.15.0.1/tcp/4001",
			isPrivate: false,
		},
		{
			name:      "172.32.x.x public",
			addrStr:   "/ip4/172.32.0.1/tcp/4001",
			isPrivate: false,
		},
		// Edge cases
		{
			name:      "no IP in multiaddr",
			addrStr:   "/tcp/4001",
			isPrivate: false,
		},
		{
			name:      "DNS address",
			addrStr:   "/dns4/example.com/tcp/4001",
			isPrivate: false,
		},
		{
			name:      "IPv6 address",
			addrStr:   "/ip6/::1/tcp/4001",
			isPrivate: false, // Function only handles IPv4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := multiaddr.NewMultiaddr(tt.addrStr)
			require.NoError(t, err)

			result := isPrivateIP(maddr)
			assert.Equal(t, tt.isPrivate, result)
		})
	}
}

// Helper functions for tests

func generateValidHexKey(t *testing.T) string {
	// Generate a real Ed25519 key and encode it
	priv, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	bytes, err := crypto.MarshalPrivateKey(priv)
	require.NoError(t, err)

	return hex.EncodeToString(bytes)
}

func TestIsPrivateIP_EdgeCases(t *testing.T) {
	// Test with actual net.IP values to ensure our CIDR checks work correctly
	testIPs := []struct {
		ip        string
		isPrivate bool
		desc      string
	}{
		// Boundary cases for 10.0.0.0/8
		{"10.0.0.0", true, "10.0.0.0/8 start"},
		{"10.255.255.255", true, "10.0.0.0/8 end"},
		{"9.255.255.255", false, "just before 10.0.0.0/8"},
		{"11.0.0.0", false, "just after 10.0.0.0/8"},

		// Boundary cases for 172.16.0.0/12
		{"172.16.0.0", true, "172.16.0.0/12 start"},
		{"172.31.255.255", true, "172.16.0.0/12 end"},
		{"172.15.255.255", false, "just before 172.16.0.0/12"},
		{"172.32.0.0", false, "just after 172.16.0.0/12"},

		// Boundary cases for 192.168.0.0/16
		{"192.168.0.0", true, "192.168.0.0/16 start"},
		{"192.168.255.255", true, "192.168.0.0/16 end"},
		{"192.167.255.255", false, "just before 192.168.0.0/16"},
		{"192.169.0.0", false, "just after 192.168.0.0/16"},
	}

	for _, tt := range testIPs {
		t.Run(tt.desc, func(t *testing.T) {
			addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/4001", tt.ip))
			require.NoError(t, err)

			result := isPrivateIP(addr)
			assert.Equal(t, tt.isPrivate, result, "IP %s should be private=%v", tt.ip, tt.isPrivate)
		})
	}
}

// Benchmark tests for helper functions

func BenchmarkGeneratePrivateKey(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = generatePrivateKey(ctx)
	}
}

func BenchmarkIsPrivateIP(b *testing.B) {
	addrs := []multiaddr.Multiaddr{}
	testIPs := []string{
		"10.0.0.1", "192.168.1.1", "8.8.8.8", "172.16.0.1",
		"1.1.1.1", "127.0.0.1", "172.32.0.1", "192.169.0.1",
	}

	for _, ip := range testIPs {
		addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/4001", ip))
		addrs = append(addrs, addr)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isPrivateIP(addrs[i%len(addrs)])
	}
}