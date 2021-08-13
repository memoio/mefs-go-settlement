package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/api/client"
	"github.com/memoio/go-settlement/server/impl"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
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

		endpoint, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/18000")
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

var createCmd = &cli.Command{
	Name:      "create",
	Usage:     "Send funds between accounts",
	ArgsUsage: "[targetAddress] [amount]",
	Flags:     []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		log.Info("create")
		apiaddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/18000")
		if err != nil {
			return err
		}

		_, raddr, err := manet.DialArgs(apiaddr)
		if err != nil {
			return err
		}

		endpoint := "ws://" + raddr + "/rpc/v0"
		//log.Info("rpc endpoint:", endpoint)

		var headers http.Header
		ctx := cctx.Context

		api, closer, err := client.NewFullNodeRPC(ctx, endpoint, headers)
		if err != nil {
			log.Error("create erctoken: ", err)
			return err
		}

		defer closer()

		uid := uuid.New()
		key, err := utils.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}

		msg := blake2b.Sum256(uid[:])

		sig, err := utils.Sign(key.SecretKey, msg[:])
		if err != nil {
			return err
		}

		addr, err := api.CreateErcToken(uid, sig, utils.ToAddress(key.PubKey))
		if err != nil {
			return err
		}

		log.Info("create token addr: ", addr)

		return nil
	},
}
