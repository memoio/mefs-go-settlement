package api

import (
	"context"
	"math/big"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/contract"
	"github.com/memoio/go-settlement/utils"
)

// common API permissions constraints
type CommonStruct struct {
	Internal struct {
		AuthVerify func(ctx context.Context, token string) ([]auth.Permission, error) `perm:"read"`
		AuthNew    func(ctx context.Context, erms []auth.Permission) ([]byte, error)  `perm:"admin"`
	}
}

type FullNodeStruct struct {
	CommonStruct

	Internal struct {
		CreateErcToken func(uuid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error)
		TotalSupply    func(tAddr, caller utils.Address) *big.Int
		BalanceOf      func(tAddr, caller, tokenOwner utils.Address) *big.Int
		Allowance      func(tAddr, caller, tokenOwner, spender utils.Address) *big.Int
		Approve        func(uid uuid.UUID, sig []byte, tAddr, caller, spender utils.Address, value *big.Int) error
		Transfer       func(uid uuid.UUID, sig []byte, tAddr, caller, to utils.Address, value *big.Int) error
		TransferFrom   func(uid uuid.UUID, sig []byte, tAddr, caller, from, to utils.Address, value *big.Int) error
		MintToken      func(uid uuid.UUID, sig []byte, tAddr, caller, target utils.Address, mintedAmount *big.Int) error
		Burn           func(uid uuid.UUID, sig []byte, tAddr, caller utils.Address, burnAmount *big.Int) error
		AirDrop        func(uid uuid.UUID, sig []byte, tAddr, caller utils.Address, addrs []utils.Address, money *big.Int) error

		CreateRoleMgr      func(uid uuid.UUID, sig []byte, caller, founder, token utils.Address) (utils.Address, error)
		Register           func(uid uuid.UUID, sig []byte, caller, addr utils.Address, sign []byte) error
		RegisterToken      func(uid uuid.UUID, sig []byte, caller, taddr utils.Address, asign []byte) error
		RegisterKeeper     func(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, blsKey, signature []byte) error
		RegisterProvider   func(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, signature []byte) error
		RegisterUser       func(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, token uint32, blsKey, usign []byte) error
		Pledge             func(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, money *big.Int, signature []byte) error
		Withdraw           func(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, money *big.Int, signature []byte) error
		CreateGroup        func(uid uuid.UUID, sig []byte, caller utils.Address, inds []uint64, level uint16, asign []byte) error
		AddKeeperToGroup   func(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, ksign, asign []byte) error
		AddProviderToGroup func(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, psign []byte) error
		Recharge           func(uid uuid.UUID, sig []byte, caller utils.Address, user uint64, tokenIndex uint32, money *big.Int, sign []byte) error
		ProWithdraw        func(uid uuid.UUID, sig []byte, caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, ksigns [][]byte) error
		WithdrawFromFs     func(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, amount *big.Int, sign []byte) error
		AddOrder           func(uid uuid.UUID, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error
		SubOrder           func(uid uuid.UUID, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error

		GetIndex          func(caller, addr utils.Address) (uint64, error)
		GetAddr           func(caller utils.Address, index uint64) (utils.Address, error)
		GetInfo           func(caller utils.Address, index uint64) (*contract.BaseInfo, error)
		GetTokenIndex     func(caller, taddr utils.Address) (uint32, error)
		GetTokenAddress   func(caller utils.Address, index uint32) (utils.Address, error)
		GetGroupInfo      func(caller utils.Address, gindex uint64) (*contract.GroupInfo, error)
		GetBalance        func(caller utils.Address, index uint64) ([]*big.Int, error)
		GetBalanceInFs    func(caller utils.Address, index uint64, tIndex uint32) ([]*big.Int, error)
		GetPledgeAddress  func(caller utils.Address) utils.Address
		GetKeeperPledge   func(caller utils.Address) *big.Int
		GetProviderPledge func(caller utils.Address) *big.Int
		GetPledgeBalance  func(caller utils.Address) []*big.Int
		GetAllTokens      func(caller utils.Address) []utils.Address
		GetAllAddrs       func(caller utils.Address) []utils.Address
		GetAllGroups      func(caller utils.Address) []*contract.GroupInfo
		GetFoundation     func(caller utils.Address) utils.Address
	}
}

func (s *CommonStruct) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	return s.Internal.AuthVerify(ctx, token)
}

func (s *CommonStruct) AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error) {
	return s.Internal.AuthNew(ctx, perms)
}

func (s *FullNodeStruct) CreateErcToken(uuid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error) {
	return s.Internal.CreateErcToken(uuid, sig, caller)
}

func (s *FullNodeStruct) TotalSupply(tAddr, caller utils.Address) *big.Int {
	return s.Internal.TotalSupply(tAddr, caller)
}

func (s *FullNodeStruct) BalanceOf(tAddr, caller, tokenOwner utils.Address) *big.Int {
	return s.Internal.BalanceOf(tAddr, caller, tokenOwner)
}

func (s *FullNodeStruct) Allowance(tAddr, caller, tokenOwner, spender utils.Address) *big.Int {
	return s.Internal.Allowance(tAddr, caller, tokenOwner, spender)
}

func (s *FullNodeStruct) Approve(uid uuid.UUID, sig []byte, tAddr, caller, spender utils.Address, value *big.Int) error {
	return s.Internal.Approve(uid, sig, tAddr, caller, spender, value)
}

func (s *FullNodeStruct) Transfer(uid uuid.UUID, sig []byte, tAddr, caller, to utils.Address, value *big.Int) error {
	return s.Internal.Transfer(uid, sig, tAddr, caller, to, value)
}

func (s *FullNodeStruct) TransferFrom(uid uuid.UUID, sig []byte, tAddr, caller, from, to utils.Address, value *big.Int) error {
	return s.Internal.TransferFrom(uid, sig, tAddr, caller, from, to, value)
}

func (s *FullNodeStruct) MintToken(uid uuid.UUID, sig []byte, tAddr, caller, target utils.Address, mintedAmount *big.Int) error {
	return s.Internal.MintToken(uid, sig, tAddr, caller, target, mintedAmount)
}

func (s *FullNodeStruct) Burn(uid uuid.UUID, sig []byte, tAddr, caller utils.Address, burnAmount *big.Int) error {
	return s.Internal.Burn(uid, sig, tAddr, caller, burnAmount)
}

func (s *FullNodeStruct) AirDrop(uid uuid.UUID, sig []byte, tAddr, caller utils.Address, addrs []utils.Address, money *big.Int) error {
	return s.Internal.AirDrop(uid, sig, tAddr, caller, addrs, money)
}

func (s *FullNodeStruct) CreateRoleMgr(uid uuid.UUID, sig []byte, caller, founder, token utils.Address) (utils.Address, error) {
	return s.Internal.CreateRoleMgr(uid, sig, caller, founder, token)
}

func (s *FullNodeStruct) Register(uid uuid.UUID, sig []byte, caller, addr utils.Address, sign []byte) error {
	return s.Internal.Register(uid, sig, caller, addr, sign)
}

func (s *FullNodeStruct) RegisterToken(uid uuid.UUID, sig []byte, caller, taddr utils.Address, asign []byte) error {
	return s.Internal.RegisterToken(uid, sig, caller, taddr, asign)
}

func (s *FullNodeStruct) RegisterKeeper(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, blsKey, signature []byte) error {
	return s.Internal.RegisterKeeper(uid, sig, caller, index, blsKey, signature)
}

func (s *FullNodeStruct) RegisterProvider(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, signature []byte) error {
	return s.Internal.RegisterProvider(uid, sig, caller, index, signature)
}

func (s *FullNodeStruct) RegisterUser(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, token uint32, blsKey, usign []byte) error {
	return s.Internal.RegisterUser(uid, sig, caller, index, gIndex, token, blsKey, usign)
}

func (s *FullNodeStruct) Pledge(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, money *big.Int, signature []byte) error {
	return s.Internal.Pledge(uid, sig, caller, index, money, signature)
}

func (s *FullNodeStruct) Withdraw(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, money *big.Int, signature []byte) error {
	return s.Internal.Withdraw(uid, sig, caller, index, tokenIndex, money, signature)
}

func (s *FullNodeStruct) CreateGroup(uid uuid.UUID, sig []byte, caller utils.Address, inds []uint64, level uint16, asign []byte) error {
	return s.Internal.CreateGroup(uid, sig, caller, inds, level, asign)
}

func (s *FullNodeStruct) AddKeeperToGroup(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, ksign, asign []byte) error {
	return s.Internal.AddKeeperToGroup(uid, sig, caller, index, gIndex, ksign, asign)
}

func (s *FullNodeStruct) AddProviderToGroup(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, psign []byte) error {
	return s.Internal.AddProviderToGroup(uid, sig, caller, index, gIndex, psign)
}

func (s *FullNodeStruct) Recharge(uid uuid.UUID, sig []byte, caller utils.Address, user uint64, tokenIndex uint32, money *big.Int, sign []byte) error {
	return s.Internal.Recharge(uid, sig, caller, user, tokenIndex, money, sign)
}

func (s *FullNodeStruct) ProWithdraw(uid uuid.UUID, sig []byte, caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, ksigns [][]byte) error {
	return s.Internal.ProWithdraw(uid, sig, caller, proIndex, tokenIndex, pay, lost, ksigns)
}

func (s *FullNodeStruct) WithdrawFromFs(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, amount *big.Int, sign []byte) error {
	return s.Internal.WithdrawFromFs(uid, sig, caller, index, tokenIndex, amount, sign)
}

func (s *FullNodeStruct) AddOrder(uid uuid.UUID, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	return s.Internal.AddOrder(uid, sig, caller, user, proIndex, start, end, size, nonce, tokenIndex, sprice, usign, psign, ksigns)
}

func (s *FullNodeStruct) SubOrder(uid uuid.UUID, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	return s.Internal.SubOrder(uid, sig, caller, user, proIndex, start, end, size, nonce, tokenIndex, sprice, usign, psign, ksigns)
}

func (s *FullNodeStruct) GetIndex(caller, addr utils.Address) (uint64, error) {
	return s.Internal.GetIndex(caller, addr)
}

func (s *FullNodeStruct) GetAddr(caller utils.Address, index uint64) (utils.Address, error) {
	return s.Internal.GetAddr(caller, index)
}

func (s *FullNodeStruct) GetInfo(caller utils.Address, index uint64) (*contract.BaseInfo, error) {
	return s.Internal.GetInfo(caller, index)
}

func (s *FullNodeStruct) GetTokenIndex(caller, taddr utils.Address) (uint32, error) {
	return s.Internal.GetTokenIndex(caller, taddr)
}

func (s *FullNodeStruct) GetTokenAddress(caller utils.Address, index uint32) (utils.Address, error) {
	return s.Internal.GetTokenAddress(caller, index)
}

func (s *FullNodeStruct) GetGroupInfo(caller utils.Address, gindex uint64) (*contract.GroupInfo, error) {
	return s.Internal.GetGroupInfo(caller, gindex)
}

func (s *FullNodeStruct) GetBalance(caller utils.Address, index uint64) ([]*big.Int, error) {
	return s.Internal.GetBalance(caller, index)
}

func (s *FullNodeStruct) GetBalanceInFs(caller utils.Address, index uint64, tIndex uint32) ([]*big.Int, error) {
	return s.Internal.GetBalanceInFs(caller, index, tIndex)
}

func (s *FullNodeStruct) GetPledgeAddress(caller utils.Address) utils.Address {
	return s.Internal.GetPledgeAddress(caller)
}

func (s *FullNodeStruct) GetKeeperPledge(caller utils.Address) *big.Int {
	return s.Internal.GetKeeperPledge(caller)
}

func (s *FullNodeStruct) GetProviderPledge(caller utils.Address) *big.Int {
	return s.Internal.GetProviderPledge(caller)
}

func (s *FullNodeStruct) GetPledgeBalance(caller utils.Address) []*big.Int {
	return s.Internal.GetPledgeBalance(caller)
}

func (s *FullNodeStruct) GetAllTokens(caller utils.Address) []utils.Address {
	return s.Internal.GetAllTokens(caller)
}

func (s *FullNodeStruct) GetAllAddrs(caller utils.Address) []utils.Address {
	return s.Internal.GetAllAddrs(caller)
}

func (s *FullNodeStruct) GetAllGroups(caller utils.Address) []*contract.GroupInfo {
	return s.Internal.GetAllGroups(caller)
}

func (s *FullNodeStruct) GetFoundation(caller utils.Address) utils.Address {
	return s.Internal.GetFoundation(caller)
}
