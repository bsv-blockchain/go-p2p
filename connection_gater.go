package p2p

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

// ConnectionGater provides fine-grained control over which connections to accept or reject
type ConnectionGater struct {
	mu              sync.RWMutex
	blockedPeers    map[peer.ID]time.Time
	blockedSubnets  []string
	maxConnsPerPeer int
	peerConns       map[peer.ID]int
	logger          Logger
}

// NewConnectionGater creates a new connection gater with specified configuration
func NewConnectionGater(logger Logger, maxConnsPerPeer int) *ConnectionGater {
	return &ConnectionGater{
		blockedPeers:    make(map[peer.ID]time.Time),
		blockedSubnets:  make([]string, 0),
		maxConnsPerPeer: maxConnsPerPeer,
		peerConns:       make(map[peer.ID]int),
		logger:          logger,
	}
}

// BlockPeer blocks a specific peer for a duration
func (cg *ConnectionGater) BlockPeer(p peer.ID, duration time.Duration) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	cg.blockedPeers[p] = time.Now().Add(duration)
}

// UnblockPeer removes a peer from the blocklist
func (cg *ConnectionGater) UnblockPeer(p peer.ID) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	delete(cg.blockedPeers, p)
}

// BlockSubnet blocks connections from a specific subnet
func (cg *ConnectionGater) BlockSubnet(subnet string) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	cg.blockedSubnets = append(cg.blockedSubnets, subnet)
}

// isPeerBlocked checks if a peer is currently blocked
func (cg *ConnectionGater) isPeerBlocked(p peer.ID) bool {
	cg.mu.RLock()
	defer cg.mu.RUnlock()
	
	if expiry, exists := cg.blockedPeers[p]; exists {
		if time.Now().Before(expiry) {
			return true
		}
		// Clean up expired block
		delete(cg.blockedPeers, p)
	}
	return false
}

// InterceptPeerDial is called before dialing a peer
func (cg *ConnectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	if cg.isPeerBlocked(p) {
		cg.logger.Debugf("[ConnectionGater] Blocked dial to peer: %s", p)
		return false
	}
	return true
}

// InterceptAddrDial is called before dialing an address
func (cg *ConnectionGater) InterceptAddrDial(p peer.ID, addr multiaddr.Multiaddr) (allow bool) {
	if cg.isPeerBlocked(p) {
		cg.logger.Debugf("[ConnectionGater] Blocked dial to address %s for peer: %s", addr, p)
		return false
	}
	
	// Check if address is in blocked subnet
	cg.mu.RLock()
	defer cg.mu.RUnlock()
	
	if len(cg.blockedSubnets) > 0 {
		if ip, err := manet.ToIP(addr); err == nil {
			ipStr := ip.String()
			for _, subnet := range cg.blockedSubnets {
				if len(ipStr) >= len(subnet) && ipStr[:len(subnet)] == subnet {
					cg.logger.Debugf("[ConnectionGater] Blocked dial to subnet %s: %s", subnet, addr)
					return false
				}
			}
		}
	}
	
	return true
}

// InterceptAccept is called before accepting a connection
func (cg *ConnectionGater) InterceptAccept(connAddr network.ConnMultiaddrs) (allow bool) {
	// Check if remote address is in blocked subnet
	remoteAddr := connAddr.RemoteMultiaddr()
	
	cg.mu.RLock()
	defer cg.mu.RUnlock()
	
	if len(cg.blockedSubnets) > 0 {
		if ip, err := manet.ToIP(remoteAddr); err == nil {
			ipStr := ip.String()
			for _, subnet := range cg.blockedSubnets {
				if len(ipStr) >= len(subnet) && ipStr[:len(subnet)] == subnet {
					cg.logger.Debugf("[ConnectionGater] Blocked accept from subnet %s: %s", subnet, remoteAddr)
					return false
				}
			}
		}
	}
	
	return true
}

// InterceptSecured is called after the handshake
func (cg *ConnectionGater) InterceptSecured(dir network.Direction, p peer.ID, connAddr network.ConnMultiaddrs) (allow bool) {
	if cg.isPeerBlocked(p) {
		cg.logger.Debugf("[ConnectionGater] Blocked secured connection from peer: %s", p)
		return false
	}
	
	// Check connection limit per peer
	if cg.maxConnsPerPeer > 0 {
		cg.mu.Lock()
		defer cg.mu.Unlock()
		
		if cg.peerConns[p] >= cg.maxConnsPerPeer {
			cg.logger.Debugf("[ConnectionGater] Peer %s exceeded max connections (%d)", p, cg.maxConnsPerPeer)
			return false
		}
		
		cg.peerConns[p]++
	}
	
	return true
}

// InterceptUpgraded is called after protocol negotiation
func (cg *ConnectionGater) InterceptUpgraded(conn network.Conn) (allow bool, reason control.DisconnectReason) {
	// Note: Connection cleanup happens periodically or when new connections are attempted
	// We can't reliably track connection closes from the gater
	return true, 0
}

// Ensure ConnectionGater implements the interface
var _ connmgr.ConnectionGater = (*ConnectionGater)(nil)