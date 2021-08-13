package client

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/memoio/go-settlement/server/api"
)

// NewCommonRPC RPCClient returns an RPC client connected to a node
// @addr			reference ./httpparse/ParseApiInfo()
// @requestHeader 	reference ./httpparse/ParseApiInfo()
func NewCommonRPC(ctx context.Context, addr string, requestHeader http.Header) (api.Common, jsonrpc.ClientCloser, error) {
	var res api.CommonStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Memoriae",
		api.GetInternalStructs(&res), requestHeader)

	return &res, closer, err
}

// NewFullNodeRPC creates a new httpparse jsonrpc remotecli.
func NewFullNodeRPC(ctx context.Context, addr string, requestHeader http.Header) (api.FullNode, jsonrpc.ClientCloser, error) {
	var res api.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Memoriae",
		api.GetInternalStructs(&res), requestHeader)

	return &res, closer, err
}
