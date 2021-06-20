package network

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"net"
)

const Protocol = "/grpc/0.0.1"

func DialOption(hosts host.Host) grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		id, err := peer.Decode(s)
		if err != nil {
			fmt.Printf("peer.ID:%s\n", s)
			return nil, errors.Wrapf(err, "parse peer.ID error %s", s)
		}
		return gostream.Dial(ctx, hosts, id, Protocol)
	})
}
