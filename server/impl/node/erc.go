package node

import (
	"math/big"

	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/contract"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
)

func (n *Node) getTokenMgr(addr utils.Address) (contract.ErcToken, error) {
	erc, ok := n.ercMap[addr]
	if ok {
		return erc, nil
	}
	return nil, ErrRes
}

func (n *Node) CreateErcToken(uid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error) {
	n.Lock()
	defer n.Unlock()

	n.count++
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

func (n *Node) TotalSupply(tAddr, caller utils.Address) *big.Int {
	n.RLock()
	defer n.RUnlock()

	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return new(big.Int)
	}

	return er.TotalSupply(caller)
}

func (n *Node) BalanceOf(tAddr, caller, tokenOwner utils.Address) *big.Int {
	n.RLock()
	defer n.RUnlock()

	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return new(big.Int)
	}

	return er.BalanceOf(caller, tokenOwner)

}

func (n *Node) Allowance(tAddr, caller, tokenOwner, spender utils.Address) *big.Int {
	n.RLock()
	defer n.RUnlock()

	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return new(big.Int)
	}

	return er.Allowance(caller, tokenOwner, spender)

}

// 处理， caller is from msg.Sender in real contract
func (n *Node) Approve(uid uuid.UUID, sig []byte, tAddr, caller, spender utils.Address, value *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	er.Approve(caller, spender, value)
	return nil
}

func (n *Node) Transfer(uid uuid.UUID, sig []byte, tAddr, caller, to utils.Address, value *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.Transfer(caller, to, value)
}

func (n *Node) TransferFrom(uid uuid.UUID, sig []byte, tAddr, caller, from, to utils.Address, value *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.TransferFrom(caller, from, to, value)
}

func (n *Node) MintToken(uid uuid.UUID, sig []byte, tAddr, caller, target utils.Address, mintedAmount *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.MintToken(caller, target, mintedAmount)
}

func (n *Node) Burn(uid uuid.UUID, sig []byte, tAddr, caller utils.Address, burnAmount *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.Burn(caller, burnAmount)
}

func (n *Node) AirDrop(uid uuid.UUID, sig []byte, tAddr, caller utils.Address, addrs []utils.Address, money *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.AirDrop(caller, addrs, money)
}
