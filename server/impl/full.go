package impl

import (
	"github.com/memoio/go-settlement/server/impl/common"
	"github.com/memoio/go-settlement/server/impl/node"

	"github.com/memoio/go-settlement/server/api"
)

var _ api.FullNode = &FullNodeAPI{}

type FullNodeAPI struct {
	api.Common
	node.ChainAPI
}

func New() *FullNodeAPI {
	n := node.NewNode()
	com := new(common.CommonAPI)

	return &FullNodeAPI{com, n}
}
