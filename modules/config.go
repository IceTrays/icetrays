package modules

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/icetrays/icetrays/network"
	"github.com/jinzhu/configor"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multibase"
	"github.com/pkg/errors"
	"io/ioutil"
	"time"
)

type Config = struct {
	P2P struct {
		Identity struct {
			PeerID  string `json:"peer_id"`
			PrivKey string `json:"priv_key"`
		} `json:"identity"`
		Bootstrap      []string `json:"bootstrap" default:"[]"`
		Listen         []string `json:"listen" default:"[\"/ip4/0.0.0.0/tcp/10040\"]"`
		Secret         string   `json:"secret"`
		EnableRelayHop bool     `json:"enable_relay_hop"`
		LowWater       int      `json:"low_water" default:"100"`
		HighWater      int      `json:"high_water" default:"400"`
		GracePeriod    int64    `json:"grace_period" default:"120000000000"`
		EnableMdns     bool     `json:"enable_mdns" default:"true"`
	} `json:"p2p"`

	Ipfs   string `default:"/ip4/127.0.0.1/tcp/5001" json:"ipfs"`
	DBPath string `default:"cluster-ds" json:"db_path"`
	Raft   struct {
		Peers    []string `json:"peers"`
		LogLevel string   `default:"DEBUG" json:"log_level"`
	} `json:"raft"`
	Port int `json:"port"`
}

func InitConfig() Config {
	var config = Config{}
	err := configor.Load(&config, "config.json")
	if err != nil {
		panic(err)
	}
	if config.P2P.Identity.PrivKey == "" {
		fmt.Println("KeyPair not found, GenerateEd25519Key...")
		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			panic(err)
		}
		bs, _ := priv.Bytes()
		s, _ := multibase.Encode(multibase.Base58BTC, bs)
		config.P2P.Identity.PrivKey = s
		id, _ := peer.IDFromPublicKey(priv.GetPublic())
		config.P2P.Identity.PeerID = id.Pretty()
		fmt.Println("peer id: ", id)
	}
	bs, _ := json.MarshalIndent(config, "", "\t")
	if err := ioutil.WriteFile("config.json", bs, 0644); err != nil {
		panic(fmt.Errorf("write config file: %s", err.Error()))
	}
	return config
}

func NetConfig(cfg Config) (*network.NetConfig, error) {
	_, s, _ := multibase.Decode(cfg.P2P.Identity.PrivKey)
	privKey, err := crypto.UnmarshalPrivateKey(s)
	if err != nil {
		return nil, err
	}
	id, err := peer.IDFromPublicKey(privKey.GetPublic())
	if err != nil || id.Pretty() != cfg.P2P.Identity.PeerID {
		return nil, fmt.Errorf("invalid key pair")
	}

	listens := []ma.Multiaddr{}
	for _, v := range cfg.P2P.Listen {
		mu, err := ma.NewMultiaddr(v)
		if err != nil {
			return nil, errors.Wrap(err, "decode multi addr: ")
		}
		listens = append(listens, mu)
	}
	sc, err := hex.DecodeString(cfg.P2P.Secret)
	if err != nil {
		return nil, errors.Wrap(err, "decode libp2p secret: ")
	}
	return &network.NetConfig{
		EnableRelayHop: cfg.P2P.EnableRelayHop,
		LowWater:       cfg.P2P.LowWater,
		HighWater:      cfg.P2P.HighWater,
		GracePeriod:    time.Duration(cfg.P2P.GracePeriod),
		Secret:         sc,
		PrivKey:        privKey,
		Addrs:          listens,
		EnableMdns:     cfg.P2P.EnableMdns,
	}, nil
}
