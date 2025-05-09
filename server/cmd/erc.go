package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"

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

		key, err := utils.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}

		uAddr := utils.ToAddress(key.PubKey)

		uid := api.GetNonce(uAddr, uAddr)

		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uid)
		msg := blake2b.Sum256(buf)

		sig, err := utils.Sign(key.SecretKey, msg[:])
		if err != nil {
			return err
		}

		addr, err := api.CreateErcToken(uid, sig, uAddr)
		if err != nil {
			return err
		}

		fmt.Println("create token addr: ", addr)

		return nil
	},
}
