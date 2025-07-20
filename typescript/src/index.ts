import { createLibp2p, Libp2p } from 'libp2p';
import { tcp } from '@libp2p/tcp';
import { yamux } from '@chainsafe/libp2p-yamux';
import { noise } from '@chainsafe/libp2p-noise';
import { gossipsub, type GossipSub } from '@chainsafe/libp2p-gossipsub';
import { bootstrap } from '@libp2p/bootstrap';
import { pubsubPeerDiscovery } from '@libp2p/pubsub-peer-discovery'
import { identify } from '@libp2p/identify';
import { kadDHT } from '@libp2p/kad-dht';
import { ping } from '@libp2p/ping';
import { preSharedKey } from '@libp2p/pnet';

export interface P2PNodeOptions {
  listenAddresses?: string[];
  bootstrapPeers?: string[];
  usePrivateDHT?: boolean;
  sharedKey?: string;
  dhtProtocolID?: string;
  port?: number;
}

export class P2PNode {
  node: Libp2p<{ pubsub: GossipSub }> | null = null;
  private topics: string[] = [
    'bitcoin/mainnet-bestblock',
    'bitcoin/mainnet-block',
    'bitcoin/mainnet-subtree',
    'bitcoin/mainnet-mining_on',
    'bitcoin/mainnet-handshake',
    'bitcoin/mainnet-rejected_tx',
    ];
  private messageHandler: (message: Uint8Array) => void;

  private constructor(messageHandler: (message: Uint8Array) => void, options: P2PNodeOptions = {}) {
    this.messageHandler = messageHandler;
  }

  static async create(messageHandler: (message: Uint8Array) => void, options: P2PNodeOptions = {}): Promise<P2PNode> {
    const instance = new P2PNode(messageHandler, options);
    await instance.init(options);
    return instance;
  }

  // Create a logger that works for both our use and libp2p's expectations
  private createLogger() {
    const baseLogger = {
      log: (message: any, ...args: any[]) => {
        console.log(`[P2P] ${new Date().toISOString()}`, message, ...args);
      },
      error: (message: any, ...args: any[]) => {
        console.error(`[P2P ERROR] ${new Date().toISOString()}`, message, ...args);
      },
      warn: (message: any, ...args: any[]) => {
        console.warn(`[P2P WARN] ${new Date().toISOString()}`, message, ...args);
      },
      info: (message: any, ...args: any[]) => {
        console.info(`[P2P INFO] ${new Date().toISOString()}`, message, ...args);
      },
      debug: (message: any, ...args: any[]) => {
        console.debug(`[P2P DEBUG] ${new Date().toISOString()}`, message, ...args);
      },
      trace: (message: any, ...args: any[]) => {
        console.trace(`[P2P TRACE] ${new Date().toISOString()}`, message, ...args);
      },
      forComponent: (component: string) => ({
        log: (message: any, ...args: any[]) => {
          console.log(`[P2P:${component}] ${new Date().toISOString()}`, message, ...args);
        },
        error: (message: any, ...args: any[]) => {
          console.error(`[P2P:${component} ERROR] ${new Date().toISOString()}`, message, ...args);
        },
        warn: (message: any, ...args: any[]) => {
          console.warn(`[P2P:${component} WARN] ${new Date().toISOString()}`, message, ...args);
        },
        info: (message: any, ...args: any[]) => {
          console.info(`[P2P:${component} INFO] ${new Date().toISOString()}`, message, ...args);
        },
        debug: (message: any, ...args: any[]) => {
          console.debug(`[P2P:${component} DEBUG] ${new Date().toISOString()}`, message, ...args);
        },
        trace: (message: any, ...args: any[]) => {
          console.trace(`[P2P:${component} TRACE] ${new Date().toISOString()}`, message, ...args);
        }
      })
    };
    return baseLogger;
  }

  private logger = this.createLogger();

  private async init(options: P2PNodeOptions) {
    const libp2pOptions: any = {
      addresses: {
        listen: options.listenAddresses || ['/ip4/0.0.0.0/tcp/0']
      },
      transports: [tcp()],
      streamMuxers: [yamux()],
      connectionEncrypters: [noise()],
      services: {
        pubsub: gossipsub({ 
          allowPublishToZeroTopicPeers: true,
          emitSelf: false,
          fallbackToFloodsub: true,
          floodPublish: true,
          doPX: true
        }),
        identify: identify(),
        ping: ping()
      },
      peerDiscovery: [
        bootstrap({ list: options.bootstrapPeers || [] }), 
        pubsubPeerDiscovery({
          topics: this.topics,
          interval: 5000,
        })],
    };

    // Add DHT service - configure differently for private vs public networks
    if (options.usePrivateDHT && options.dhtProtocolID) {
      // Private DHT configuration to match Go implementation
      libp2pOptions.services.dht = kadDHT({
        protocol: options.dhtProtocolID + '/kad/1.0.0',
        clientMode: false,
        validators: {},
        selectors: {}
      });
    } else {
      // Public DHT configuration
      libp2pOptions.services.dht = kadDHT({
        protocol: '/ipfs/kad/1.0.0',
        clientMode: false
      });
    }

    if (options.usePrivateDHT && options.sharedKey) {
      this.logger.info('Using private DHT with shared key:', options.sharedKey);
      
      // Format PSK in the same way as Go implementation
      const pskString = `/key/swarm/psk/1.0.0/\n/base16/\n${options.sharedKey}`;
      
      this.logger.info('PSK formatted string:', pskString);
      libp2pOptions.connectionProtector = preSharedKey({
        psk: new TextEncoder().encode(pskString)
      });
    }

    this.logger.info('Creating libp2p node with options:', {
      listenAddresses: options.listenAddresses,
      bootstrapPeers: options.bootstrapPeers,
      usePrivateDHT: options.usePrivateDHT
    });

    this.node = await createLibp2p(libp2pOptions);

    this.logger.info('Starting libp2p node...');
    await this.node.start();
    this.logger.info('Libp2p node started successfully');

    // Wait for pubsub service to be ready
    this.logger.info('Waiting for pubsub service to be ready...');
    let pubsubReady = false;
    let attempts = 0;
    const maxAttempts = 10;
    
    while (!pubsubReady && attempts < maxAttempts) {
      try {
        if (this.node.services.pubsub && this.node.services.pubsub.isStarted()) {
          pubsubReady = true;
          this.logger.info('Pubsub service is ready');
        } else {
          this.logger.info(`Pubsub not ready yet, attempt ${attempts + 1}/${maxAttempts}`);
          await new Promise(resolve => setTimeout(resolve, 1000));
          attempts++;
        }
      } catch (error) {
        this.logger.warn(`Error checking pubsub status (attempt ${attempts + 1}):`, error);
        await new Promise(resolve => setTimeout(resolve, 1000));
        attempts++;
      }
    }
    
    if (!pubsubReady) {
      throw new Error('Pubsub service failed to start after maximum attempts');
    }

    // Add event listeners for debugging
    this.node.addEventListener('peer:discovery', async (evt) => {
      this.logger.info('Peer discovered:', evt.detail.id.toString());
      this.logger.info('Peer multiaddrs:', evt.detail.multiaddrs.map(ma => ma.toString()));
      
      // Try to manually connect to discovered peer
      if (this.node) {
        try {
          this.logger.info('Attempting to connect to discovered peer...');
          await this.node.dial(evt.detail.id);
          this.logger.info('Successfully dialed discovered peer');
        } catch (error) {
          this.logger.error('Failed to dial discovered peer:', error);
        }
      }
    });

    this.node.addEventListener('peer:connect', (evt) => {
      this.logger.info('✅ Peer connected:', evt.detail.toString());
      if (this.node) {
        this.logger.info('Total connected peers:', this.node.getPeers().length);
      }
    });

    this.node.addEventListener('peer:disconnect', (evt) => {
      this.logger.info('❌ Peer disconnected:', evt.detail.toString());
      if (this.node) {
        this.logger.info('Remaining connected peers:', this.node.getPeers().length);
      }
    });

    // Subscribe to topics and advertise them
    for (const topic of this.topics) {
      this.logger.info(`Subscribing to topic: ${topic}`);
      this.node.services.pubsub.subscribe(topic);
      const subscribers = this.node.services.pubsub.getSubscribers(topic).map(p => p.toString());
      this.logger.info(`Subscribers for ${topic}:`, subscribers);
      
      // Advertise topic subscription via DHT (similar to Go implementation)
      try {
        if ('dht' in this.node.services) {
          const topicKey = new TextEncoder().encode(`/pubsub/topic/${topic}`);
          await (this.node.services as any).dht.provide(topicKey);
          this.logger.info(`Advertising topic: ${topic}`);
        } else {
          this.logger.warn('DHT service not available for topic advertising');
        }
      } catch (error) {
        this.logger.warn(`Failed to advertise topic ${topic}:`, error);
      }
    }

    // Create component-specific logger for gossipsub
    const gossipLogger = this.logger.forComponent('gossipsub');

    // Handle incoming messages
    this.node.services.pubsub.addEventListener('gossipsub:message', (evt: any) => {
      const msg = evt.detail.msg;
      gossipLogger.info('Received message from topic:', msg.topic);
      this.messageHandler(msg.data);
    });
  }

  async stop() {
    if (this.node) {
      await this.node.stop();
    }
  }

  async publish(topic: string, message: Uint8Array) {
    if (!this.node) {
      throw new Error('Node not initialized');
    }
    await this.node.services.pubsub.publish(topic, message);
    this.logger.info(`Published message to topic: ${topic}`);
  }

  getConnectedPeers() {
    if (!this.node) {
      return [];
    }
    return this.node.getPeers();
  }

  getNodeId() {
    if (!this.node) {
      return null;
    }
    return this.node.peerId.toString();
  }

  async getTopicPeers(topic: string) {
    if (!this.node) {
      return [];
    }
    const subscribers = this.node.services.pubsub.getSubscribers(topic);
    // Handle different return formats
    if (Array.isArray(subscribers)) {
      return subscribers;
    }
    // If it returns an object with subscribers property
    if (subscribers && typeof subscribers === 'object' && 'subscribers' in subscribers) {
      return (subscribers as any).subscribers || [];
    }
    return [];
  }
}