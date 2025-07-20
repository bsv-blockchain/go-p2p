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
    usePrivateDHT: true,
    sharedKey: '285b49e6d910726a70f205086c39cbac6d8dcc47839053a21b1f614773bbc137',
    dhtProtocolID: '/teranode',
    port: 9901
  };

  const node = await P2PNode.create(messageHandler, options);

  console.log('P2P node started');

  // Publish a test message after a short delay
  setTimeout(async () => {
    try {
      // await node.publish('Hello from TypeScript P2P demo!');
      console.log('Would have published test message');
    } catch (error) {
      console.error('Failed to publish:', error);
    }
  }, 5000);

  // Periodically log connected peers
  setInterval(async () => {
    // Note: Add methods to P2PNode if needed for getting peers/stats
    console.log('Node is running...');
  }, 30000);

  // Graceful shutdown
  process.on('SIGINT', async () => {
    await node.stop();
    console.log('Node stopped');
    process.exit(0);
  });
}

main().catch(console.error);
