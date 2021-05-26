package network

import (
	"context"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type PeerHandler struct {
	host host.Host
}

func (p *PeerHandler) HandlePeerFound(info peer.AddrInfo) {
	_ = p.host.Connect(context.Background(), info)
}
