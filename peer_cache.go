package p2p

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// PeerCacheVersion defines the version of the peer cache format
	PeerCacheVersion = 1
	// DefaultCacheTTL defines the default time-to-live for cached peers
	DefaultCacheTTL = 30 * 24 * time.Hour // 30 days
	// DefaultMaxCachedPeers defines the default maximum number of peers to cache
	DefaultMaxCachedPeers = 100
)

// CachedPeer represents a peer saved to disk for persistence across restarts
type CachedPeer struct {
	ID              string    `json:"id"`
	Addresses       []string  `json:"addresses"`
	LastSeen        time.Time `json:"last_seen"`
	LastConnected   time.Time `json:"last_connected"`
	ConnectionCount int       `json:"connection_count"`
	FailureCount    int       `json:"failure_count"`
}

// PeerCache manages persistent peer storage to improve network connectivity
// across restarts by remembering successful peer connections
type PeerCache struct {
	Peers   []CachedPeer `json:"peers"`
	Version int          `json:"version"`
	mu      sync.RWMutex
}

// NewPeerCache creates a new peer cache instance
func NewPeerCache() *PeerCache {
	return &PeerCache{
		Peers:   make([]CachedPeer, 0),
		Version: PeerCacheVersion,
	}
}

// LoadPeerCache loads peer cache from disk
func LoadPeerCache(filepath string) (*PeerCache, error) {
	// Expand home directory if needed
	if len(filepath) > 1 && filepath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		filepath = home + filepath[1:]
	}

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		// Return empty cache if file doesn't exist
		return NewPeerCache(), nil
	}

	// Read file
	data, err := os.ReadFile(filepath) // #nosec G304 - filepath is validated and from configuration
	if err != nil {
		return nil, fmt.Errorf("failed to read peer cache: %w", err)
	}

	// Parse JSON
	var cache PeerCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse peer cache: %w", err)
	}

	// Version check
	if cache.Version != PeerCacheVersion {
		// For now, just return empty cache if version mismatch
		return NewPeerCache(), nil
	}

	return &cache, nil
}

// Save writes the peer cache to disk
func (pc *PeerCache) Save(filepath string) error {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	// Expand home directory if needed
	if len(filepath) > 1 && filepath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		filepath = home + filepath[1:]
	}

	// Create directory if it doesn't exist
	lastSlash := strings.LastIndex(filepath, "/")
	if lastSlash > 0 {
		dir := filepath[:lastSlash]
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(pc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal peer cache: %w", err)
	}

	// Write to temp file first (atomic write)
	tmpFile := filepath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write peer cache: %w", err)
	}

	// Rename to final location
	if err := os.Rename(tmpFile, filepath); err != nil {
		return fmt.Errorf("failed to rename peer cache: %w", err)
	}

	return nil
}

// AddOrUpdatePeer adds or updates a peer in the cache
func (pc *PeerCache) AddOrUpdatePeer(peerID peer.ID, addresses []string, connected bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	id := peerID.String()
	now := time.Now()

	// Look for existing peer
	for i, p := range pc.Peers {
		if p.ID == id {
			// Update existing peer
			pc.Peers[i].LastSeen = now
			// Only update addresses if we have new ones
			if len(addresses) > 0 {
				pc.Peers[i].Addresses = addresses
			}
			if connected {
				pc.Peers[i].LastConnected = now
				pc.Peers[i].ConnectionCount++
				pc.Peers[i].FailureCount = 0 // Reset failures on success
			} else {
				pc.Peers[i].FailureCount++
			}
			return
		}
	}

	// Add new peer
	newPeer := CachedPeer{
		ID:              id,
		Addresses:       addresses,
		LastSeen:        now,
		ConnectionCount: 0,
		FailureCount:    0,
	}

	if connected {
		newPeer.LastConnected = now
		newPeer.ConnectionCount = 1
	} else {
		// If not connected on first attempt, count as failure
		newPeer.FailureCount = 1
	}

	pc.Peers = append(pc.Peers, newPeer)
}

// GetBestPeers returns the best peers sorted by reliability and recency
func (pc *PeerCache) GetBestPeers(limit int, ttl time.Duration) []CachedPeer {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	// Filter out old peers
	cutoff := time.Now().Add(-ttl)
	validPeers := make([]CachedPeer, 0)

	for _, p := range pc.Peers {
		if p.LastSeen.After(cutoff) && p.FailureCount < 5 {
			validPeers = append(validPeers, p)
		}
	}

	// Sort by reliability and recency
	sort.Slice(validPeers, func(i, j int) bool {
		// First, prioritize peers we've successfully connected to
		if validPeers[i].ConnectionCount > 0 && validPeers[j].ConnectionCount == 0 {
			return true
		}
		if validPeers[i].ConnectionCount == 0 && validPeers[j].ConnectionCount > 0 {
			return false
		}

		// Then sort by success ratio
		ratioI := float64(validPeers[i].ConnectionCount) / float64(validPeers[i].ConnectionCount+validPeers[i].FailureCount+1)
		ratioJ := float64(validPeers[j].ConnectionCount) / float64(validPeers[j].ConnectionCount+validPeers[j].FailureCount+1)
		if ratioI != ratioJ {
			return ratioI > ratioJ
		}

		// Finally, sort by last connected time
		return validPeers[i].LastConnected.After(validPeers[j].LastConnected)
	})

	// Return up to limit peers
	if limit > len(validPeers) {
		limit = len(validPeers)
	}
	return validPeers[:limit]
}

// Prune removes old or unreliable peers
func (pc *PeerCache) Prune(maxPeers int, ttl time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	cutoff := time.Now().Add(-ttl)
	newPeers := make([]CachedPeer, 0)

	// Remove old and highly unreliable peers
	for _, p := range pc.Peers {
		if p.LastSeen.After(cutoff) && p.FailureCount < 10 {
			newPeers = append(newPeers, p)
		}
	}

	// If still over limit, keep only the best peers
	if len(newPeers) > maxPeers {
		// Sort by quality
		sort.Slice(newPeers, func(i, j int) bool {
			ratioI := float64(newPeers[i].ConnectionCount) / float64(newPeers[i].ConnectionCount+newPeers[i].FailureCount+1)
			ratioJ := float64(newPeers[j].ConnectionCount) / float64(newPeers[j].ConnectionCount+newPeers[j].FailureCount+1)
			if ratioI != ratioJ {
				return ratioI > ratioJ
			}
			return newPeers[i].LastConnected.After(newPeers[j].LastConnected)
		})
		newPeers = newPeers[:maxPeers]
	}

	pc.Peers = newPeers
}

// Count returns the number of cached peers
func (pc *PeerCache) Count() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return len(pc.Peers)
}

// RemovePeer removes a specific peer from the cache
func (pc *PeerCache) RemovePeer(peerID peer.ID) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	id := peerID.String()
	newPeers := make([]CachedPeer, 0, len(pc.Peers))

	for _, p := range pc.Peers {
		if p.ID != id {
			newPeers = append(newPeers, p)
		}
	}

	pc.Peers = newPeers
}
