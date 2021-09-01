package api

import (
	"context"
	"math/big"

	"github.com/filecoin-project/go-jsonrpc/auth"
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
		GetNonce func(caller, addr utils.Address) uint64

		CreateErcToken func(uid uint64, sig []byte, caller utils.Address) (utils.Address, error)
		TotalSupply    func(tAddr, caller utils.Address) *big.Int
		BalanceOf      func(tAddr, caller, tokenOwner utils.Address) *big.Int
		Allowance      func(tAddr, caller, tokenOwner, spender utils.Address) *big.Int
		Approve        func(uid uint64, sig []byte, tAddr, caller, spender utils.Address, value *big.Int) error
		Transfer       func(uid uint64, sig []byte, tAddr, caller, to utils.Address, value *big.Int) error
		TransferFrom   func(uid uint64, sig []byte, tAddr, caller, from, to utils.Address, value *big.Int) error
		MintToken      func(uid uint64, sig []byte, tAddr, caller, target utils.Address, mintedAmount *big.Int) error
		Burn           func(uid uint64, sig []byte, tAddr, caller utils.Address, burnAmount *big.Int) error
		AirDrop        func(uid uint64, sig []byte, tAddr, caller utils.Address, addrs []utils.Address, money *big.Int) error

		CreateRoleMgr      func(uid uint64, sig []byte, caller, founder, token utils.Address) (utils.Address, error)
		Register           func(uid uint64, sig []byte, caller, addr utils.Address, sign []byte) error
		RegisterToken      func(uid uint64, sig []byte, caller, taddr utils.Address) error
		RegisterKeeper     func(uid uint64, sig []byte, caller utils.Address, index uint64, blsKey, signature []byte) error
		RegisterProvider   func(uid uint64, sig []byte, caller utils.Address, index uint64, signature []byte) error
		RegisterUser       func(uid uint64, sig []byte, caller utils.Address, index, gIndex uint64, blsKey []byte) error
		Pledge             func(uid uint64, sig []byte, caller utils.Address, index uint64, money *big.Int) error
		Withdraw           func(uid uint64, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, money *big.Int) error
		CreateGroup        func(uid uint64, sig []byte, caller utils.Address, level uint16) error
		AddKeeperToGroup   func(uid uint64, sig []byte, caller utils.Address, index, gIndex uint64, asign []byte) error
		AddProviderToGroup func(uid uint64, sig []byte, caller utils.Address, index, gIndex uint64) error
		Recharge           func(uid uint64, sig []byte, caller utils.Address, user uint64, tokenIndex uint32, money *big.Int) error
		ProWithdraw        func(uid uint64, sig []byte, caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, ksigns [][]byte) error
		WithdrawFromFs     func(uid uint64, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, amount *big.Int) error
		AddOrder           func(uid uint64, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error
		SubOrder           func(uid uint64, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error

		GetIndex          func(caller, addr utils.Address) (uint64, error)
		GetAddr           func(caller utils.Address, index uint64) (utils.Address, error)
		GetInfo           func(caller utils.Address, index uint64) (*contract.BaseInfo, error)
		GetTokenIndex     func(caller, taddr utils.Address) (uint32, error)
		GetTokenAddress   func(caller utils.Address, index uint32) (utils.Address, error)
		GetGroupInfo      func(caller utils.Address, gindex uint64) (*contract.GroupInfo, error)
		GetBalance        func(caller utils.Address, index uint64) ([]*big.Int, error)
		GetBalanceInFs    func(caller utils.Address, index uint64, tIndex uint32) ([]*big.Int, error)
		GetSettleInfo     func(caller utils.Address, index uint64, tIndex uint32) (*contract.Settlement, error)
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

func (s *FullNodeStruct) GetNonce(caller, addr utils.Address) uint64 {
	return s.Internal.GetNonce(caller, addr)
}

func (s *FullNodeStruct) CreateErcToken(uid uint64, sig []byte, caller utils.Address) (utils.Address, error) {
	return s.Internal.CreateErcToken(uid, sig, caller)
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

func (s *FullNodeStruct) Approve(uid uint64, sig []byte, tAddr, caller, spender utils.Address, value *big.Int) error {
	return s.Internal.Approve(uid, sig, tAddr, caller, spender, value)
}

func (s *FullNodeStruct) Transfer(uid uint64, sig []byte, tAddr, caller, to utils.Address, value *big.Int) error {
	return s.Internal.Transfer(uid, sig, tAddr, caller, to, value)
}

func (s *FullNodeStruct) TransferFrom(uid uint64, sig []byte, tAddr, caller, from, to utils.Address, value *big.Int) error {
	return s.Internal.TransferFrom(uid, sig, tAddr, caller, from, to, value)
}

func (s *FullNodeStruct) MintToken(uid uint64, sig []byte, tAddr, caller, target utils.Address, mintedAmount *big.Int) error {
	return s.Internal.MintToken(uid, sig, tAddr, caller, target, mintedAmount)
}

func (s *FullNodeStruct) Burn(uid uint64, sig []byte, tAddr, caller utils.Address, burnAmount *big.Int) error {
	return s.Internal.Burn(uid, sig, tAddr, caller, burnAmount)
}

func (s *FullNodeStruct) AirDrop(uid uint64, sig []byte, tAddr, caller utils.Address, addrs []utils.Address, money *big.Int) error {
	return s.Internal.AirDrop(uid, sig, tAddr, caller, addrs, money)
}

func (s *FullNodeStruct) CreateRoleMgr(uid uint64, sig []byte, caller, founder, token utils.Address) (utils.Address, error) {
	return s.Internal.CreateRoleMgr(uid, sig, caller, founder, token)
}

func (s *FullNodeStruct) Register(uid uint64, sig []byte, caller, addr utils.Address, sign []byte) error {
	return s.Internal.Register(uid, sig, caller, addr, sign)
}

func (s *FullNodeStruct) RegisterToken(uid uint64, sig []byte, caller, taddr utils.Address) error {
	return s.Internal.RegisterToken(uid, sig, caller, taddr)
}

func (s *FullNodeStruct) RegisterKeeper(uid uint64, sig []byte, caller utils.Address, index uint64, blsKey, signature []byte) error {
	return s.Internal.RegisterKeeper(uid, sig, caller, index, blsKey, signature)
}

func (s *FullNodeStruct) RegisterProvider(uid uint64, sig []byte, caller utils.Address, index uint64, signature []byte) error {
	return s.Internal.RegisterProvider(uid, sig, caller, index, signature)
}

func (s *FullNodeStruct) RegisterUser(uid uint64, sig []byte, caller utils.Address, index, gIndex uint64, blsKey []byte) error {
	return s.Internal.RegisterUser(uid, sig, caller, index, gIndex, blsKey)
}

func (s *FullNodeStruct) Pledge(uid uint64, sig []byte, caller utils.Address, index uint64, money *big.Int) error {
	return s.Internal.Pledge(uid, sig, caller, index, money)
}

func (s *FullNodeStruct) Withdraw(uid uint64, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, money *big.Int) error {
	return s.Internal.Withdraw(uid, sig, caller, index, tokenIndex, money)
}

func (s *FullNodeStruct) CreateGroup(uid uint64, sig []byte, caller utils.Address, level uint16) error {
	return s.Internal.CreateGroup(uid, sig, caller, level)
}

func (s *FullNodeStruct) AddKeeperToGroup(uid uint64, sig []byte, caller utils.Address, index, gIndex uint64, asign []byte) error {
	return s.Internal.AddKeeperToGroup(uid, sig, caller, index, gIndex, asign)
}

func (s *FullNodeStruct) AddProviderToGroup(uid uint64, sig []byte, caller utils.Address, index, gIndex uint64) error {
	return s.Internal.AddProviderToGroup(uid, sig, caller, index, gIndex)
}

func (s *FullNodeStruct) Recharge(uid uint64, sig []byte, caller utils.Address, user uint64, tokenIndex uint32, money *big.Int) error {
	return s.Internal.Recharge(uid, sig, caller, user, tokenIndex, money)
}

func (s *FullNodeStruct) ProWithdraw(uid uint64, sig []byte, caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, ksigns [][]byte) error {
	return s.Internal.ProWithdraw(uid, sig, caller, proIndex, tokenIndex, pay, lost, ksigns)
}

func (s *FullNodeStruct) WithdrawFromFs(uid uint64, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, amount *big.Int) error {
	return s.Internal.WithdrawFromFs(uid, sig, caller, index, tokenIndex, amount)
}

func (s *FullNodeStruct) AddOrder(uid uint64, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	return s.Internal.AddOrder(uid, sig, caller, user, proIndex, start, end, size, nonce, tokenIndex, sprice, usign, psign, ksigns)
}

func (s *FullNodeStruct) SubOrder(uid uint64, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
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

func (s *FullNodeStruct) GetSettleInfo(caller utils.Address, index uint64, tIndex uint32) (*contract.Settlement, error) {
	return s.Internal.GetSettleInfo(caller, index, tIndex)
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
