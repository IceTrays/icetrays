package network

import (
	"context"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	noise "github.com/libp2p/go-libp2p-noise"
	record "github.com/libp2p/go-libp2p-record"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"sync"
	"time"
)

type NetConfig struct {
	EnableRelayHop bool
	LowWater       int
	HighWater      int
	GracePeriod    time.Duration
	Secret         []byte
	PrivKey        crypto.PrivKey
	Addrs          []ma.Multiaddr
	EnableMdns     bool
}

func newDHT(ctx context.Context, h host.Host, store datastore.Datastore, extraopts ...dual.Option) (*dual.DHT, error) {
	opts := []dual.Option{
		dual.DHTOption(dht.NamespacedValidator("pk", record.PublicKeyValidator{})),
		//dual.DHTOption(dht.NamespacedValidator("ipns", ipns.Validator{KeyBook: h.Peerstore()})),
		dual.DHTOption(dht.Concurrency(10)),
	}

	opts = append(opts, extraopts...)

	//if batchingDs, ok := store.(datastore.Batching); ok {
	//	dhtDatastore := namespace.Wrap(batchingDs, datastore.NewKey("dht"))
	//	opts = append(opts, dual.DHTOption(dht.Datastore(dhtDatastore)))
	//	//logger.Debug("enabling DHT record persistence to datastore")
	//}

	return dual.New(ctx, h, opts...)
}

func baseOpts(psk pnet.PSK) []libp2p.Option {
	return []libp2p.Option{
		libp2p.PrivateNetwork(psk),
		libp2p.EnableNATService(),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// TODO: quic does not support private networks
		// libp2p.Transport(libp2pquic.NewTransport),
		libp2p.DefaultTransports,
	}
}

func NewHost(ctx context.Context, cfg NetConfig) (host.Host, error) {
	relayOpts := []relay.RelayOpt{}
	if cfg.EnableRelayHop {
		relayOpts = append(relayOpts, relay.OptHop)
	}
	connman := connmgr.NewConnManager(cfg.LowWater, cfg.HighWater, cfg.GracePeriod)
	opts := []libp2p.Option{
		libp2p.ListenAddrs(cfg.Addrs...),
		libp2p.NATPortMap(),
		libp2p.ConnectionManager(connman),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err := newDHT(ctx, h, nil)
			return idht, err
		}),
		libp2p.EnableRelay(relayOpts...),
		libp2p.EnableAutoRelay(),
	}

	finalOpts := []libp2p.Option{
		libp2p.Identity(cfg.PrivKey),
	}
	finalOpts = append(finalOpts, baseOpts(cfg.Secret)...)
	finalOpts = append(finalOpts, opts...)

	h, err := libp2p.New(
		ctx,
		finalOpts...,
	)
	return h, err
}

type Network struct {
	host  host.Host
	mdns  discovery.Service
	lock  sync.Mutex
	conns map[string]*grpc.ClientConn
}

func NewNetwork(cfg NetConfig) (*Network, error) {
	net := &Network{
		conns: map[string]*grpc.ClientConn{},
	}
	h, err := NewHost(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	net.host = h
	if cfg.EnableMdns {
		mdns, err := discovery.NewMdnsService(context.Background(), h, time.Second*20, "ipfs-fs-cluster")
		if err != nil {
			panic(err)
		}
		handle := PeerHandler{
			host: h,
		}
		mdns.RegisterNotifee(&handle)
		net.mdns = mdns
	}
	return net, nil
}

func (net *Network) Connect(dialCtx context.Context, id string) (*grpc.ClientConn, error) {
	net.lock.Lock()
	defer net.lock.Unlock()
	if conn, ok := net.conns[id]; ok && conn.GetState() != connectivity.Shutdown {
		return conn, nil
	} else {
		conn, err := grpc.DialContext(dialCtx, id, DialOption(net.host), grpc.WithInsecure())
		if err != nil {
			return conn, err
		}
		net.conns[id] = conn
		return conn, nil
	}
}

func (net *Network) Close() error {
	net.lock.Lock()
	defer net.lock.Unlock()
	for _, conn := range net.conns {
		_ = conn.Close()
	}
	if net.mdns != nil {
		_ = net.mdns.Close()
	}
	return net.host.Close()
}

func (net *Network) Host() host.Host {
	return net.host
}
