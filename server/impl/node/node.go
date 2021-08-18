package node

import (
	"math/big"

	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/contract"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
)

type Node struct {
	ercMap   map[utils.Address]contract.ErcToken
	roleMap  map[utils.Address]contract.RoleMgr
	nonceMap map[utils.Address]uint64
}

func NewNode() *Node {
	n := &Node{
		ercMap:   make(map[utils.Address]contract.ErcToken),
		roleMap:  make(map[utils.Address]contract.RoleMgr),
		nonceMap: make(map[utils.Address]uint64),
	}

	// todo: persist and load

	return n
}

func (n *Node) CreateErcToken(uid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error) {
	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return utils.NilAddress, ErrRes
	}

	et := contract.NewErcToken(caller)

	n.ercMap[et.GetContractAddress()] = et

	log.Info("create erctoken for: ", caller.String())

	return et.GetContractAddress(), nil
}

func (n *Node) CreateRoleMgr(uid uuid.UUID, sig []byte, caller, founder, token utils.Address) (utils.Address, error) {
	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return utils.NilAddress, ErrRes
	}

	kposit := new(big.Int).Mul(new(big.Int).SetUint64(contract.KeeperDeposit), new(big.Int).SetUint64(contract.Token))
	pposit := new(big.Int).Mul(new(big.Int).SetUint64(contract.ProviderDeposit), new(big.Int).SetUint64(contract.Token))

	rm := contract.NewRoleMgr(caller, founder, token, kposit, pposit)

	n.roleMap[rm.GetContractAddress()] = rm

	log.Info("create roleMgr for: ", caller.String())

	return rm.GetContractAddress(), nil
}
