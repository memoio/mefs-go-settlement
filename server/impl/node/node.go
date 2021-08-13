package node

import (
	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/contract"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
)

type Node struct {
	ercMap  map[utils.Address]contract.ErcToken
	roleMap map[utils.Address]contract.RoleMgr
	fsMap   map[utils.Address]contract.FsMgr
}

func NewNode() *Node {
	n := &Node{
		ercMap:  make(map[utils.Address]contract.ErcToken),
		roleMap: make(map[utils.Address]contract.RoleMgr),
		fsMap:   make(map[utils.Address]contract.FsMgr),
	}

	return n
}

func (n *Node) CreateErcToken(uid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error) {
	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return utils.NilAddress, ErrRes
	}

	et, err := contract.NewErcToken(caller)
	if err != nil {
		return utils.NilAddress, err
	}

	n.ercMap[et.GetContractAddress()] = et

	log.Info("create erctoken for: ", caller.String())

	return et.GetContractAddress(), nil
}
