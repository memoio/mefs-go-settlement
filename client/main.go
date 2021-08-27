package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/memoio/go-settlement/server/api"
	"github.com/memoio/go-settlement/server/api/client"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

var log = utils.Logger("main")

func GetApi() (api.FullNode, jsonrpc.ClientCloser, error) {
	apiaddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/18000")
	if err != nil {
		return nil, nil, err
	}

	_, raddr, err := manet.DialArgs(apiaddr)
	if err != nil {
		return nil, nil, err
	}

	endpoint := "ws://" + raddr + "/rpc/v0"
	//log.Info("rpc endpoint:", endpoint)

	var headers http.Header
	ctx := context.Background()

	return client.NewFullNodeRPC(ctx, endpoint, headers)
}

func main() {
	log.Info("create")
	api, closer, _ := GetApi()

	defer closer()

	key, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		return
	}

	uAddr := utils.ToAddress(key.PubKey)

	uid := api.GetNonce(uAddr, uAddr)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)

	sig, err := utils.Sign(key.SecretKey, msg[:])
	if err != nil {
		return
	}

	addr, err := api.CreateErcToken(uid, sig, uAddr)
	if err != nil {
		return
	}

	fmt.Println("create token addr: ", addr)

	bal := api.BalanceOf(addr, uAddr, uAddr)

	fmt.Println(uAddr, "has balance: ", bal)

	return
}
