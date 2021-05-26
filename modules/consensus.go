package modules

import (
	"context"
	"github.com/hashicorp/raft"
	"github.com/icetrays/icetrays/consensus"
	"github.com/icetrays/icetrays/datastore"
	"github.com/icetrays/icetrays/network"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	p2praft "github.com/libp2p/go-libp2p-raft"
	ma "github.com/multiformats/go-multiaddr"
	"go.uber.org/fx"
	"time"
)

func Network(lc fx.Lifecycle, cfg *network.NetConfig) (*network.Network, error) {
	n, err := network.NewNetwork(*cfg)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			return n.Close()
		},
	})
	return n, nil
}

func RaftConfig(js Config) *raft.Config {
	cfg := raft.DefaultConfig()
	cfg.SnapshotThreshold = 100
	cfg.LogLevel = js.Raft.LogLevel
	cfg.LocalID = raft.ServerID(js.P2P.Identity.PeerID)
	return cfg
}

func DataStore(lc fx.Lifecycle, js Config) (*datastore.BadgerDB, error) {
	d, err := datastore.NewBadgerStore(js.DBPath)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			return d.Close()
		},
	})
	return d, err
}

func SnapshotStore() (raft.SnapshotStore, error) {
	return raft.NewFileSnapshotStore("snapshot", 5, nil)
}

func Fsm(store *datastore.BadgerDB, api *httpapi.HttpApi) (*consensus.Fsm, error) {
	return consensus.NewFsm(store, api)
}

func IpfsClient(js Config) (*httpapi.HttpApi, error) {
	addr, err := ma.NewMultiaddr(js.Ipfs)
	if err != nil {
		return nil, err
	}
	return httpapi.NewApi(addr)
}

func Transport(n *network.Network) (raft.Transport, error) {
	return p2praft.NewLibp2pTransport(n.Host(), time.Minute*2)
}

func Raft(lc fx.Lifecycle, conf *raft.Config, fsm *consensus.Fsm, snaps raft.SnapshotStore, trans raft.Transport, badger *datastore.BadgerDB, js Config) (*raft.Raft, error) {
	servers := make([]raft.Server, len(js.Raft.Peers))
	for i := 0; i < len(js.Raft.Peers); i++ {
		servers[i] = raft.Server{
			Suffrage: 0,
			ID:       raft.ServerID(js.Raft.Peers[i]),
			Address:  raft.ServerAddress(js.Raft.Peers[i]),
		}
	}
	r, err := raft.NewRaft(conf, fsm, datastore.NewLogDB(badger), datastore.NewStableDB(badger), snaps, trans)
	if err != nil {
		return nil, err
	}
	_ = r.BootstrapCluster(raft.Configuration{Servers: servers})
	lc.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			return r.Shutdown().Error()
		},
	})
	return r, nil
}

func Node(lc fx.Lifecycle, r *raft.Raft, fsm *consensus.Fsm, js Config, net *network.Network, ipfs *httpapi.HttpApi) (*consensus.Node, error) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			cancel()
			return nil
		},
	})
	return consensus.NewNode(ctx, r, fsm, js.P2P.Identity.PeerID, net, ipfs)
}

//type Clients struct {
//	c map[string]http.GreeterClient
//}
//
//func (client Clients) Client(id string) http.GreeterClient {
//	if c, ok := client.c[id]; !ok {
//		return nil
//	} else {
//		return c
//	}
//}
//
//func RpcClients(n *network.Network, js Config) (*Clients, error) {
//	c := make(map[string]http.GreeterClient)
//	for i := 0; i < len(js.Raft.Peers); i++ {
//		if js.Raft.Peers[i] == js.P2P.Identity.PeerID {
//			continue
//		}
//		conn, err := n.Connect(n.Context(), js.Raft.Peers[i])
//		if err != nil {
//			return nil, err
//		}
//		c[js.Raft.Peers[i]] = http.NewGreeterClient(conn)
//	}
//	return &Clients{c: c}, nil
//}
