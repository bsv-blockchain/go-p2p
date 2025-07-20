import { createLibp2p, Libp2p } from 'libp2p';
import { tcp } from '@libp2p/tcp';
import { mplex } from '@libp2p/mplex';
import { yamux } from '@chainsafe/libp2p-yamux';
import { noise } from '@chainsafe/libp2p-noise';
import { gossipsub } from '@chainsafe/libp2p-gossipsub';
import type { GossipsubEvents } from '@chainsafe/libp2p-gossipsub';
import { PubSub } from '@libp2p/interface';
import { bootstrap } from '@libp2p/bootstrap';
import { fromString as uint8ArrayFromString } from 'uint8arrays/from-string';
import { toString as uint8ArrayToString } from 'uint8arrays/to-string';

export interface P2PNodeOptions {
  listenAddresses?: string[];
  bootstrapPeers?: string[];
}

export class P2PNode {
  private node: Libp2p<{ pubsub: PubSub<GossipsubEvents> }> | null = null;
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
    this.node = await createLibp2p({
      addresses: {
        listen: options.listenAddresses || ['/ip4/0.0.0.0/tcp/0']
      },
      transports: [tcp()],
      streamMuxers: [yamux(), mplex()],
      connectionEncrypters: [noise()],
      services: {
        pubsub: gossipsub({ allowPublishToZeroTopicPeers: true })
      },
      peerDiscovery: options.bootstrapPeers ? [bootstrap({ list: options.bootstrapPeers })] : []
    });

    await this.node.start();

    // Subscribe to the topic
    this.node.services.pubsub.subscribe(this.topic);

    // Handle incoming messages
    this.node.services.pubsub.addEventListener('gossipsub:message', (evt) => {
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
