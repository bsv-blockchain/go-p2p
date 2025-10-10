package p2p

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeerCacheSaveAndLoad(t *testing.T) {
	// Create temp file for cache
	tmpDir, err := os.MkdirTemp("", "peercache-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cacheFile := filepath.Join(tmpDir, "peers.json")

	// Create new cache and add peers
	cache := NewPeerCache()

	// Generate test peer IDs
	peer1, err := peer.Decode("12D3KooWDYmEobVGYR8UgkxNrBvj7W92ZCvvyYxAeKpJfFGCqGms")
	require.NoError(t, err)
	peer2, err := peer.Decode("12D3KooWLRPJAA5o6LHEmnG7rWMyQnai5AcVPjVZ1m9jqhGVTqGm")
	require.NoError(t, err)

	// Add peers to cache
	cache.AddOrUpdatePeer(peer1, []string{"/ip4/192.168.1.1/tcp/4001"}, true)
	// For peer2, first add it, then mark a failed connection
	cache.AddOrUpdatePeer(peer2, []string{"/ip4/192.168.1.2/tcp/4002", "/ip4/10.0.0.1/tcp/4002"}, false)

	// Save cache
	err = cache.Save(cacheFile)
	require.NoError(t, err)

	// Load cache
	loadedCache, err := LoadPeerCache(cacheFile)
	require.NoError(t, err)

	// Verify loaded data
	assert.Equal(t, 2, loadedCache.Count())
	assert.Equal(t, PeerCacheVersion, loadedCache.Version)

	// Check peer details
	bestPeers := loadedCache.GetBestPeers(10, DefaultCacheTTL)
	assert.Len(t, bestPeers, 2)

	// Find peers in results (order may vary based on timestamps)
	var foundPeer1, foundPeer2 *CachedPeer
	for i := range bestPeers {
		if bestPeers[i].ID == peer1.String() {
			foundPeer1 = &bestPeers[i]
		}
		if bestPeers[i].ID == peer2.String() {
			foundPeer2 = &bestPeers[i]
		}
	}

	require.NotNil(t, foundPeer1, "peer1 should be in cache")
	require.NotNil(t, foundPeer2, "peer2 should be in cache")

	// Check peer1 details (connected successfully)
	t.Logf("Peer1: ConnectionCount=%d, FailureCount=%d", foundPeer1.ConnectionCount, foundPeer1.FailureCount)
	assert.Equal(t, 1, foundPeer1.ConnectionCount)
	assert.Equal(t, 0, foundPeer1.FailureCount)

	// Check peer2 details (failed connection)
	t.Logf("Peer2: ConnectionCount=%d, FailureCount=%d", foundPeer2.ConnectionCount, foundPeer2.FailureCount)
	assert.Equal(t, 0, foundPeer2.ConnectionCount)
	assert.Equal(t, 1, foundPeer2.FailureCount)
}

func TestPeerCacheGetBestPeers(t *testing.T) {
	cache := NewPeerCache()

	// Generate test peer IDs
	var peers []peer.ID
	for i := 0; i < 5; i++ {
		// Create deterministic peer IDs for testing
		peerStr := "12D3KooWDYmEobVGYR8UgkxNrBvj7W92ZCvvyYxAeKpJfFGCqGm" + string(rune('a'+i))
		p, err := peer.Decode(peerStr[:len(peerStr)-1])
		if err != nil {
			// If decode fails, just use a simple ID
			p = peer.ID("peer" + string(rune('0'+i)))
		}
		peers = append(peers, p)
	}

	// Add peers with different success rates
	cache.AddOrUpdatePeer(peers[0], []string{"/ip4/1.1.1.1/tcp/4001"}, true)
	cache.AddOrUpdatePeer(peers[0], nil, true) // 2 successes

	cache.AddOrUpdatePeer(peers[1], []string{"/ip4/2.2.2.2/tcp/4002"}, true)
	cache.AddOrUpdatePeer(peers[1], nil, false) // 1 success, 1 failure

	cache.AddOrUpdatePeer(peers[2], []string{"/ip4/3.3.3.3/tcp/4003"}, false)
	cache.AddOrUpdatePeer(peers[2], nil, false) // 0 successes, 2 failures

	cache.AddOrUpdatePeer(peers[3], []string{"/ip4/4.4.4.4/tcp/4004"}, false)
	cache.AddOrUpdatePeer(peers[3], nil, false)
	cache.AddOrUpdatePeer(peers[3], nil, false)
	cache.AddOrUpdatePeer(peers[3], nil, false)
	cache.AddOrUpdatePeer(peers[3], nil, false) // 0 successes, 5 failures (should be filtered)

	cache.AddOrUpdatePeer(peers[4], []string{"/ip4/5.5.5.5/tcp/4005"}, true) // 1 success

	// Get best peers
	bestPeers := cache.GetBestPeers(3, DefaultCacheTTL)

	// Should get 3 peers (peers[3] filtered out due to 5 failures)
	assert.Len(t, bestPeers, 3)

	// Check order (higher success ratio first)
	assert.Equal(t, peers[0].String(), bestPeers[0].ID) // 100% success
	// peers[1] and peers[4] both have some success, order may vary
}

func TestPeerCachePrune(t *testing.T) {
	cache := NewPeerCache()

	// Add 10 peers
	for i := 0; i < 10; i++ {
		peerID := peer.ID("peer" + string(rune('0'+i)))
		cache.AddOrUpdatePeer(peerID, []string{"/ip4/127.0.0.1/tcp/400" + string(rune('0'+i))}, i%2 == 0)
	}

	assert.Equal(t, 10, cache.Count())

	// Prune to max 5 peers
	cache.Prune(5, DefaultCacheTTL)
	assert.Equal(t, 5, cache.Count())

	// Best peers should remain
	bestPeers := cache.GetBestPeers(10, DefaultCacheTTL)
	assert.Len(t, bestPeers, 5)
}

func TestPeerCacheTTL(t *testing.T) {
	cache := NewPeerCache()

	// Add a peer with manipulated time
	peerID := peer.ID("old-peer")
	cache.AddOrUpdatePeer(peerID, []string{"/ip4/127.0.0.1/tcp/4001"}, true)

	// Manually set last seen to old time
	cache.mu.Lock()
	if len(cache.Peers) > 0 {
		cache.Peers[0].LastSeen = time.Now().Add(-40 * 24 * time.Hour) // 40 days ago
	}
	cache.mu.Unlock()

	// Get best peers with 30 day TTL
	bestPeers := cache.GetBestPeers(10, 30*24*time.Hour)

	// Old peer should be filtered out
	assert.Empty(t, bestPeers)
}

func TestPeerCacheRemovePeer(t *testing.T) {
	cache := NewPeerCache()

	// Add peers
	peer1 := peer.ID("peer1")
	peer2 := peer.ID("peer2")
	peer3 := peer.ID("peer3")

	cache.AddOrUpdatePeer(peer1, []string{"/ip4/1.1.1.1/tcp/4001"}, true)
	cache.AddOrUpdatePeer(peer2, []string{"/ip4/2.2.2.2/tcp/4002"}, true)
	cache.AddOrUpdatePeer(peer3, []string{"/ip4/3.3.3.3/tcp/4003"}, true)

	assert.Equal(t, 3, cache.Count())

	// Remove peer2
	cache.RemovePeer(peer2)
	assert.Equal(t, 2, cache.Count())

	// Check remaining peers
	bestPeers := cache.GetBestPeers(10, DefaultCacheTTL)
	assert.Len(t, bestPeers, 2)

	foundPeer2 := false
	for _, p := range bestPeers {
		if p.ID == peer2.String() {
			foundPeer2 = true
		}
	}
	assert.False(t, foundPeer2, "peer2 should have been removed")
}
