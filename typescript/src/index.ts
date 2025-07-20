import { createLibp2p, Libp2p } from 'libp2p';
import { tcp } from '@libp2p/tcp';
import { yamux } from '@chainsafe/libp2p-yamux';
import { noise } from '@chainsafe/libp2p-noise';
import { gossipsub, type GossipSub } from '@chainsafe/libp2p-gossipsub';
import { bootstrap } from '@libp2p/bootstrap';
import { fromString as uint8ArrayFromString } from 'uint8arrays/from-string';
import { identify } from '@libp2p/identify';

export interface P2PNodeOptions {
  listenAddresses?: string[];
  bootstrapPeers?: string[];
  usePrivateDHT?: boolean;
  sharedKey?: string;
  dhtProtocolID?: string;
  port?: number;
}

export class P2PNode {
  private node: Libp2p<{ pubsub: GossipSub }> | null = null;
  private topic: string = 'bitcoin/mainnet-block';
  private messageHandler: (message: Uint8Array) => void;

  private constructor(messageHandler: (message: Uint8Array) => void, options: P2PNodeOptions = {}) {
    this.messageHandler = messageHandler;
  }

  static async create(messageHandler: (message: Uint8Array) => void, options: P2PNodeOptions = {}): Promise<P2PNode> {
    const instance = new P2PNode(messageHandler, options);
    await instance.init(options);
    return instance;
  }

  private async init(options: P2PNodeOptions) {
    const libp2pOptions: any = {
      addresses: {
        listen: options.listenAddresses || ['/ip4/0.0.0.0/tcp/0']
      },
      transports: [tcp()],
      streamMuxers: [yamux()],
      connectionEncrypters: [noise()],
      services: {
        pubsub: gossipsub({ allowPublishToZeroTopicPeers: true }),
        identify: identify()
      },
      peerDiscovery: options.bootstrapPeers ? [bootstrap({ list: options.bootstrapPeers })] : []
    };

    if (options.usePrivateDHT && options.sharedKey) {
      const psk = new Uint8Array(32);
      // Convert hex sharedKey to Uint8Array
      for (let i = 0; i < options.sharedKey.length; i += 2) {
        psk[i / 2] = parseInt(options.sharedKey.substring(i, i + 2), 16);
      }
      libp2pOptions.privateNetwork = psk;
      
      if (options.dhtProtocolID) {
        libp2pOptions.dht = {
          protocolPrefix: options.dhtProtocolID
        };
      }
    }

    this.node = await createLibp2p(libp2pOptions);

    await this.node.start();

    // Subscribe to the topic
    this.node.services.pubsub.subscribe(this.topic);

    // Handle incoming messages
    this.node.services.pubsub.addEventListener('gossipsub:message', (evt: any) => {
      const msg = evt.detail.msg;
      if (msg.topic === this.topic) {
        this.messageHandler(msg.data);
      }
    });
  }

  async stop() {
    if (this.node) {
      await this.node.stop();
    }
  }

  async publish(message: string) {
    if (!this.node) throw new Error('Node not initialized');
    const msg = uint8ArrayFromString(message);
    await this.node.services.pubsub.publish(this.topic, msg);
  }
}
