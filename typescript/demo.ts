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
    listenAddresses: ['/ip4/127.0.0.1/tcp/9901'],
    bootstrapPeers: ['/dns4/teranode-bootstrap.bsvb.tech/tcp/9901/p2p/12D3KooWESmhNAN8s6NPdGNvJH3zJ4wMKDxapXKNUe2DzkAwKYqK'],
    staticPeers: [
      // Active Teranode peers discovered from Go implementation
      '/dns4/teranode-mainnet-peer.taal.com/tcp/9905/p2p/12D3KooWJGPdPPw72GU6gFF4LqUjeFF7qmPCS2bZK8ywMvybYfXD',
      '/dns4/teranode-mainnet-us-01.bsvb.tech/tcp/9905/p2p/12D3KooWPJAHHaNy5BsViK1B5iTQmz5cLaUheAKEuNkHqMbwZ8jd',
      '/dns4/teranode-eks-mainnet-us-1-peer.bsvb.tech/tcp/9911/p2p/12D3KooWFjGChbwVteGsqH6NfHtKbtdW5XgnvmQRpem2kUAQjsGq',
      '/dns4/bsva-ovh-teranode-eu-1.bsvb.tech/tcp/9905/p2p/12D3KooWAdBeSVue71DTmfMEKyBG2s1hg91zJnze85rt2uKCZWbW',
      '/dns4/teranode-eks-mainnet-eu-1-peer.bsvb.tech/tcp/9911/p2p/12D3KooWRioUF2AYvC6ofiXhjE5V3MLiVrRKMAEyHiz5iYQgnB5f'
    ],
    usePrivateDHT: true,
    sharedKey: '285b49e6d910726a70f205086c39cbac6d8dcc47839053a21b1f614773bbc137',
    dhtProtocolID: '/teranode',
    port: 9901,
    logLevel: 'debug'
  };

  const p2p = await P2PNode.create(messageHandler, options);

  console.log('P2P node started');

  // Try to discover and connect to known active peers
  const knownActivePeers = [
    '12D3KooWJGPdPPw72GU6gFF4LqUjeFF7qmPCS2bZK8ywMvybYfXD',
    '12D3KooWPJAHHaNy5BsViK1B5iTQmz5cLaUheAKEuNkHqMbwZ8jd',
    '12D3KooWFjGChbwVteGsqH6NfHtKbtdW5XgnvmQRpem2kUAQjsGq'
  ];

  // Wait a bit for DHT to initialize, then try to discover active peers
  setTimeout(async () => {
    console.log('\n=== Attempting to discover active peers ===');
    for (const peerId of knownActivePeers) {
      await p2p.connectToPeerById(peerId);
      // Wait a bit between connection attempts
      await new Promise(resolve => setTimeout(resolve, 2000));
    }
    console.log('=== Peer discovery complete ===\n');
  }, 10000); // Wait 10 seconds for DHT to be ready

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
