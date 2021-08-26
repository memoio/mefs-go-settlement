package main

import (
	"fmt"
	"os"

	"github.com/memoio/go-settlement/server/impl"
	"github.com/memoio/go-settlement/utils"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
)

var log = utils.Logger("main")

func main() {
	local := []*cli.Command{
		runCmd,
		createCmd,
	}

	app := &cli.App{
		Name:                 "settle",
		Usage:                "Memoriae settement chain",
		Version:              "1.0.0",
		EnableBashCompletion: true,
		Flags:                []cli.Flag{},

		Commands: local,
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err) // nolint:errcheck
		os.Exit(1)
	}
}

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start settlement server",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		log.Info("Starting server")
		fullapi := impl.New()

		h, err := FullNodeHandler(fullapi, false)
		if err != nil {
			log.Errorf("failed to instantiate rpc handler: %s", err)
			return err
		}

		endpoint, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/18000")
		if err != nil {
			return err
		}

		shutdownChan := make(chan struct{})

		// Serve the RPC.
		rpcStopper, err := ServeRPC(h, "server", endpoint)
		if err != nil {
			log.Errorf("failed to start json-rpc endpoint: %s", err)
			return err
		}

		finishCh := impl.MonitorShutdown(shutdownChan,
			impl.ShutdownHandler{Component: "rpc server", StopFunc: rpcStopper},
		)
		<-finishCh

		return nil
	},
}
