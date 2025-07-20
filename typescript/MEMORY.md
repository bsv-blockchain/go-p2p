# Teranode P2P Proof of Concept - libp2p Message Listening Architecture

## Overview
The Teranode P2P PoC application (located at `./typescript/teranode-p2p-poc`) is a Go-based peer-to-peer networking application that uses libp2p for distributed message communication. The application establishes a P2P node that listens to various blockchain-related topics and stores received messages in a SQLite database.

## Core Architecture

### Main Components
1. **P2PNode** - Core libp2p node implementation
2. **Message Model** - Database schema for storing received messages
3. **Configuration System** - YAML-based configuration with Viper
4. **HTTP/WebSocket Server** - Web interface for message viewing
5. **Database Layer** - SQLite with GORM for message persistence

## libp2p Connection and Listener Setup

### 1. Node Initialization (`NewP2PNode`)
The P2P node initialization process:

**Private Key Management:**
- Generates or loads Ed25519 private key for node identity
- Keys provide cryptographic identity for peer authentication
- Supports both auto-generation and pre-configured keys

**Network Configuration:**
- Supports both public and private DHT networks
- Private networks use pre-shared keys (PSK) for isolation
- Configurable listen addresses and advertise addresses
- Default port: 9901

**libp2p Host Setup:**
```go
// Standard public network setup
opts := []libp2p.Option{
    libp2p.ListenAddrStrings(listenMultiAddresses...),
    libp2p.Identity(*pk),
}

// Private network with PSK
psk := pnet.PSK(pskBytes)
opts = append(opts, libp2p.PrivateNetwork(psk))
```

### 2. GossipSub Protocol Initialization
The application uses libp2p's GossipSub for topic-based messaging:

**GossipSub Setup:**
```go
ps, err := pubsub.NewGossipSub(ctx, s.host,
    pubsub.WithMessageSignaturePolicy(pubsub.StrictSign))
```

**Topic Subscription:**
- Joins multiple predefined topics during initialization
- Topics include: bitcoin/mainnet-bestblock, bitcoin/mainnet-block, etc.
- Each topic creates a `pubsub.Topic` object for message handling

### 3. Message Listening Architecture

**Topic Handler Registration:**
The `SetTopicHandler` function establishes message listeners:

```go
func (s *P2PNode) SetTopicHandler(ctx context.Context, topicName string, handler Handler) error {
    topic := s.topics[topicName]
    sub, err := topic.Subscribe()
    if err != nil {
        return err
    }
    
    s.handlerByTopic[topicName] = handler
    
    // Start goroutine for message listening
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                m, err := sub.Next(ctx)
                if err != nil {
                    continue
                }
                // Process message through handler
                handler(ctx, m.Data, m.ReceivedFrom.String())
            }
        }
    }()
    
    return nil
}
```

**Message Processing Flow:**
1. **Subscription**: Each topic gets a dedicated subscription via `topic.Subscribe()`
2. **Goroutine Listener**: Separate goroutine runs continuous message polling
3. **Message Retrieval**: Uses `sub.Next(ctx)` to get next message from topic
4. **Handler Invocation**: Calls registered handler function with message data
5. **Database Storage**: Handler stores message in SQLite database
6. **WebSocket Broadcast**: Notifies connected web clients of new messages

### 4. Peer Discovery and Connection

**DHT-Based Discovery:**
- Uses Kademlia DHT for peer discovery
- Supports both public IPFS DHT and private DHT
- Continuous peer discovery in background goroutines

**Bootstrap Nodes:**
- Connects to predefined bootstrap addresses
- Example: `/dns4/teranode-bootstrap.bsvb.tech/tcp/9901/p2p/12D3KooW...`
- Enables initial network entry and peer discovery

**Static Peer Connections:**
- Maintains connections to configured static peers
- Automatic reconnection on connection loss

## Message Storage and Processing

### Message Model Structure
```go
type Message struct {
    ID         uint      `gorm:"primaryKey"`
    Topic      string    `gorm:"index;not null"`
    Data       string    `gorm:"type:text;not null"`
    Peer       string    `gorm:"type:text"`
    ReceivedAt time.Time `gorm:"autoCreateTime"`
}
```

### Handler Implementation
The main application registers handlers for each topic:
```go
err = node.SetTopicHandler(ctx, topicCopy, func(ctx context.Context, data []byte, peer string) {
    msg := model.Message{
        Topic:      topicCopy,
        Data:       string(data),
        Peer:       peer,
        ReceivedAt: time.Now(),
    }
    if err := db.Create(&msg).Error; err != nil {
        log.Errorf("Failed to store message for topic %s: %v", topicCopy, err)
    } else {
        log.Infof("Stored message for topic %s from %s", topicCopy, peer)
        websocket.BroadcastMessage(msg)
    }
})
```

## Configuration Details

### Default Topics Monitored
- `bitcoin/mainnet-bestblock` - Best block announcements
- `bitcoin/mainnet-block` - New block propagation
- `bitcoin/mainnet-subtree` - Merkle subtree updates
- `bitcoin/mainnet-mining_on` - Mining status messages
- `bitcoin/mainnet-handshake` - Peer handshake messages
- `bitcoin/mainnet-rejected_tx` - Rejected transaction notifications

### Network Configuration
- **Port**: 9901 (default)
- **Private DHT**: Enabled by default with shared key
- **Bootstrap**: Single bootstrap node for network entry
- **Database**: SQLite file at `./teranode-p2p.db`

## Key Features

### Concurrent Message Handling
- Each topic has dedicated goroutine for message processing
- Non-blocking message reception and processing
- Thread-safe operations with sync.Map and atomic operations

### Network Resilience
- Automatic peer reconnection
- Error handling for network disruptions
- Context-based cancellation for graceful shutdown

### Monitoring and Metrics
- Tracks bytes sent/received
- Peer connection timestamps
- Message reception logging
- WebSocket real-time updates

This architecture provides a robust foundation for blockchain P2P communication using libp2p's proven networking stack with GossipSub for efficient topic-based message distribution.