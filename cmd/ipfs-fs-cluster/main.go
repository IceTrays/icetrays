package main

import (
	"fmt"
	"github.com/hashicorp/raft"
	"github.com/icetrays/icetrays/consensus"
	"github.com/icetrays/icetrays/modules"
	"go.uber.org/fx"
	"time"
)

type App struct {
	*fx.App
	Raft *raft.Raft
}

func New(opts ...fx.Option) *App {

	app := fx.New(opts...)
	if err := app.Err(); err != nil {
		fmt.Printf("fx.New failed: %v", err)
		panic(err)
	}
	return &App{
		App: app,
	}
}

func T(fsm *consensus.Fsm) {
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for range ticker.C {
			fmt.Println(fsm.State.Root())
		}
	}()
}

func main() {
	var options = []fx.Option{
		fx.Provide(modules.InitConfig),
		fx.Provide(modules.NetConfig),
		fx.Provide(modules.Network),
		fx.Provide(modules.RaftConfig),
		fx.Provide(modules.DataStore),
		fx.Provide(modules.SnapshotStore),
		fx.Provide(modules.Fsm),
		fx.Provide(modules.IpfsClient),
		fx.Provide(modules.Transport),
		fx.Provide(modules.Raft),
		//fx.Provide(modules.RpcClients),
		fx.StopTimeout(time.Minute),
		fx.Provide(modules.Node),
		fx.Invoke(modules.Server2),
		fx.Invoke(T),
	}
	app := New(options...)
	app.Run()

}
