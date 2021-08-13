package node

import (
	"errors"

	"github.com/google/uuid"
	"github.com/memoio/go-settlement/utils"
)

var log = utils.Logger("node")

var (
	ErrRes = errors.New("error result in node")
)

type ChainAPI interface {
	CreateErcToken(uuid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error)
}
