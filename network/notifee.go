package network

import (
	"context"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type PeerHandler struct {
	host      host.Host
	connected func(peer string)
}

func (p *PeerHandler) HandlePeerFound(info peer.AddrInfo) {
	err := p.host.Connect(context.Background(), info)
	if err == nil {
		p.connected(info.ID.String())
	}

}
