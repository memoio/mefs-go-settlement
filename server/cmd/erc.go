package main

import (
	"crypto/rand"
	"net/http"

	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/api/client"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
)

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
