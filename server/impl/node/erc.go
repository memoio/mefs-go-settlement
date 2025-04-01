package node

import (
	"encoding/binary"
	"math/big"

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

func (n *Node) CreateErcToken(uid uint64, sig []byte, caller utils.Address) (utils.Address, error) {
	n.Lock()
	defer n.Unlock()

	n.count++

	cn := n.getAndIncNonce(caller)
	if cn != uid {
		return utils.NilAddress, ErrNonce
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return utils.NilAddress, ErrRes
	}

	et := contract.NewErcToken(caller)

	n.ercMap[et.GetContractAddress()] = et

	log.Info("create erctoken for: ", caller.String())

	return et.GetContractAddress(), nil
}

// 处理， caller is from msg.Sender in real contract
func (n *Node) Approve(uid uint64, sig []byte, tAddr, caller, spender utils.Address, value *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	cn := n.getAndIncNonce(caller)
	if cn != uid {
		return ErrNonce
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)

	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	er.Approve(caller, spender, value)
	return nil
}

func (n *Node) Transfer(uid uint64, sig []byte, tAddr, caller, to utils.Address, value *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	cn := n.getAndIncNonce(caller)
	if cn != uid {
		return ErrNonce
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)

	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.Transfer(caller, to, value)
}

func (n *Node) TransferFrom(uid uint64, sig []byte, tAddr, caller, from, to utils.Address, value *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	cn := n.getAndIncNonce(caller)
	if cn != uid {
		return ErrNonce
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)

	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.TransferFrom(caller, from, to, value)
}

func (n *Node) MintToken(uid uint64, sig []byte, tAddr, caller, target utils.Address, mintedAmount *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	cn := n.getAndIncNonce(caller)
	if cn != uid {
		return ErrNonce
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)

	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.MintToken(caller, target, mintedAmount)
}

func (n *Node) Burn(uid uint64, sig []byte, tAddr, caller utils.Address, burnAmount *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	cn := n.getAndIncNonce(caller)
	if cn != uid {
		return ErrNonce
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)

	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.Burn(caller, burnAmount)
}

func (n *Node) AirDrop(uid uint64, sig []byte, tAddr, caller utils.Address, addrs []utils.Address, money *big.Int) error {
	n.Lock()
	defer n.Unlock()

	n.count++
	er, err := n.getTokenMgr(tAddr)
	if err != nil {
		return err
	}

	cn := n.getAndIncNonce(caller)
	if cn != uid {
		return ErrNonce
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)

	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return er.AirDrop(caller, addrs, money)
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
