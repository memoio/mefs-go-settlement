package node

import (
	"math/big"
	"sync"

	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/contract"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
)

type Node struct {
	sync.RWMutex
	count    uint64
	rm       contract.RoleMgr
	ercMap   map[utils.Address]contract.ErcToken
	nonceMap map[utils.Address]uint64
}

func NewNode() *Node {
	n := &Node{
		count:    0,
		ercMap:   make(map[utils.Address]contract.ErcToken),
		nonceMap: make(map[utils.Address]uint64),
	}

	// todo: persist and load

	return n
}

func (n *Node) CreateRoleMgr(uid uuid.UUID, sig []byte, caller, founder, token utils.Address) (utils.Address, error) {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return utils.NilAddress, ErrRes
	}

	kposit := new(big.Int).Mul(new(big.Int).SetUint64(contract.KeeperDeposit), new(big.Int).SetUint64(contract.Token))
	pposit := new(big.Int).Mul(new(big.Int).SetUint64(contract.ProviderDeposit), new(big.Int).SetUint64(contract.Token))

	rm := contract.NewRoleMgr(caller, founder, token, kposit, pposit)

	n.rm = rm

	log.Info("create roleMgr for: ", caller.String())

	return rm.GetContractAddress(), nil
}

// 注册地址，获取序号
func (n *Node) Register(uid uuid.UUID, sig []byte, caller, addr utils.Address, sign []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.Register(caller, addr, sign)
}

// by admin, 注册erc20代币地址
func (n *Node) RegisterToken(uid uuid.UUID, sig []byte, caller, taddr utils.Address, asign []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.RegisterToken(caller, taddr, asign)
}

// 注册成为keeper角色
func (n *Node) RegisterKeeper(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, blsKey, signature []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.RegisterKeeper(caller, index, blsKey, signature)
}

// 注册成为prvider角色
func (n *Node) RegisterProvider(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, signature []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.RegisterProvider(caller, index, signature)
}

// 注册成为user角色，从fs contract调用
func (n *Node) RegisterUser(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, token uint32, blsKey, usign []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.RegisterUser(caller, index, gIndex, token, blsKey, usign)
}

// 质押,
func (n *Node) Pledge(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, money *big.Int, signature []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.Pledge(caller, index, money, signature)
}

// 取回token对应的代币, money zero means all
func (n *Node) Withdraw(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, money *big.Int, signature []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.Withdraw(caller, index, tokenIndex, money, signature)
}

// 创建组，by admin
func (n *Node) CreateGroup(uid uuid.UUID, sig []byte, caller utils.Address, inds []uint64, level uint16, asign []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.CreateGroup(caller, inds, level, asign)
}

// 向组中添加keeper，by keeper and admin
func (n *Node) AddKeeperToGroup(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, ksign, asign []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.AddKeeperToGroup(caller, index, gIndex, ksign, asign)
}

// 向组中添加provider
func (n *Node) AddProviderToGroup(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, psign []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.AddProviderToGroup(caller, index, gIndex, psign)
}

func (n *Node) Recharge(uid uuid.UUID, sig []byte, caller utils.Address, user uint64, tokenIndex uint32, money *big.Int, sign []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.Recharge(caller, user, tokenIndex, money, sign)
}

func (n *Node) ProWithdraw(uid uuid.UUID, sig []byte, caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, ksigns [][]byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.ProWithdraw(caller, proIndex, tokenIndex, pay, lost, ksigns)
}

func (n *Node) WithdrawFromFs(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, amount *big.Int, sign []byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.WithdrawFromFs(caller, index, tokenIndex, amount, sign)
}

func (n *Node) AddOrder(uid uuid.UUID, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.AddOrder(caller, user, proIndex, start, end, size, nonce, tokenIndex, sprice, usign, psign, ksigns)
}

func (n *Node) SubOrder(uid uuid.UUID, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	n.Lock()
	defer n.Unlock()

	n.count++

	msg := blake2b.Sum256(uid[:])
	ok := utils.Verify(caller, msg[:], sig)
	if !ok {
		return ErrRes
	}

	return n.rm.SubOrder(caller, user, proIndex, start, end, size, nonce, tokenIndex, sprice, usign, psign, ksigns)
}

func (n *Node) GetIndex(caller, addr utils.Address) (uint64, error) {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetIndex(caller, addr)
}

func (n *Node) GetInfo(caller utils.Address, index uint64) (*contract.BaseInfo, utils.Address, error) {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetInfo(caller, index)
}

func (n *Node) GetTokenIndex(caller, taddr utils.Address) (uint32, error) {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetTokenIndex(caller, taddr)
}

func (n *Node) GetTokenAddress(caller utils.Address, index uint32) (utils.Address, error) {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetTokenAddress(caller, index)
}

func (n *Node) GetGroupInfo(caller utils.Address, gindex uint64) (*contract.GroupInfo, error) {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetGroupInfo(caller, gindex)
}

func (n *Node) GetBalance(caller utils.Address, index uint64) ([]*big.Int, error) {
	n.RLock()
	defer n.RUnlock()

	paddr := n.rm.GetPledgeAddress(caller)

	pp, err := contract.GetPledgePool(paddr)
	if err != nil {
		return nil, err
	}

	return pp.GetBalance(caller, index), nil
}

func (n *Node) GetBalanceInFs(caller utils.Address, index uint64, tIndex uint32) (*big.Int, *big.Int, *big.Int, error) {
	n.RLock()
	defer n.RUnlock()

	ui, _, err := n.rm.GetInfo(caller, index)
	if err != nil {
		return nil, nil, nil, err
	}

	gi, err := n.rm.GetGroupInfo(caller, ui.GIndex)
	if err != nil {
		return nil, nil, nil, err
	}

	fm, err := contract.GetFsMgr(gi.FsAddr)
	if err != nil {
		return nil, nil, nil, err
	}

	avail, lock, paid := fm.GetBalance(caller, index, tIndex)
	return avail, lock, paid, nil
}

func (n *Node) GetPledgeAddress(caller utils.Address) utils.Address {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetPledgeAddress(caller)
}

func (n *Node) GetPledge(caller utils.Address) (*big.Int, *big.Int, []*big.Int) {
	n.RLock()
	defer n.RUnlock()

	paddr := n.rm.GetPledgeAddress(caller)

	pp, err := contract.GetPledgePool(paddr)
	if err != nil {
		return nil, nil, nil
	}

	k, p := n.rm.GetPledge(caller)

	return k, p, pp.GetPledge(caller)
}

func (n *Node) GetAllTokens(caller utils.Address) []utils.Address {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetAllTokens(caller)
}

func (n *Node) GetAllAddrs(caller utils.Address) []utils.Address {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetAllAddrs(caller)
}

func (n *Node) GetAllGroups(caller utils.Address) []*contract.GroupInfo {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetAllGroups(caller)
}

func (n *Node) GetFoundation(caller utils.Address) utils.Address {
	n.RLock()
	defer n.RUnlock()

	return n.rm.GetFoundation(caller)
}
