import { P2PNode } from './src/index';
import { toString as uint8ArrayToString } from 'uint8arrays/to-string';

async function main() {
  // Message handler that logs received messages
  const messageHandler = (message: Uint8Array) => {
    const str = uint8ArrayToString(message);
    console.log('Received message on bitcoin/mainnet-block:', str);
  };

  // Configuration from teranode-p2p-poc/config.yaml
  const options = {
    listenAddresses: ['/ip4/127.0.0.1/tcp/9902'],
    bootstrapPeers: ['/dns4/teranode-bootstrap.bsvb.tech/tcp/9901/p2p/12D3KooWESmhNAN8s6NPdGNvJH3zJ4wMKDxapXKNUe2DzkAwKYqK'],
    usePrivateDHT: true,
    sharedKey: '285b49e6d910726a70f205086c39cbac6d8dcc47839053a21b1f614773bbc137',
    dhtProtocolID: '/teranode',
    port: 9901,
    logLevel: 'debug'
  };

  const p2p = await P2PNode.create(messageHandler, options);

  console.log('P2P node started');

  // Periodically log connected peers and status
  setInterval(async () => {
    const connectedPeers = p2p.getConnectedPeers();
    const nodeId = p2p.getNodeId();
    
    console.log('\n=== Node Status ===');
    console.log('Node ID:', nodeId);
    console.log('Connected peers:', connectedPeers.length);
    
    if (connectedPeers.length > 0) {
      console.log('Peer IDs:', connectedPeers.map(p => p.toString()));
      
      // Check topic subscribers for each topic
      const topics = [
        'bitcoin/mainnet-bestblock',
        'bitcoin/mainnet-block',
        'bitcoin/mainnet-subtree',
        'bitcoin/mainnet-mining_on',
        'bitcoin/mainnet-handshake',
        'bitcoin/mainnet-rejected_tx'
      ];
      
      for (const topic of topics) {
        const subscribers = await p2p.getTopicPeers(topic);
        if (subscribers.length > 0) {
          console.log(`Topic ${topic} has ${subscribers.length} subscribers:`, subscribers.map(p => p.toString()));
        }
      }
    } else {
      console.log('No peers connected yet...');
    }
    console.log('==================\n');
  }, 15000);

  // Graceful shutdown
  process.on('SIGINT', async () => {
    await p2p.stop();
    console.log('Node stopped');
    process.exit(0);
  });
}

main().catch(console.error);
