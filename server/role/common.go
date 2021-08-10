package role

import (
	"errors"
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

var zero = new(big.Int).SetInt64(0)

var (
	ErrRes = errors.New("error result")
)

var globalMap map[utils.Address]interface{}

func init() {
	globalMap = make(map[utils.Address]interface{})
}

type info interface {
	GetContractAddress() utils.Address
	GetOwnerAddress() utils.Address
}

// ErcToken is
// 参考：https://zhuanlan.zhihu.com/p/391837660
type ErcToken interface {
	// 查询
	TotalSupply() *big.Int
	BalanceOf(tokenOwner utils.Address) *big.Int
	Allowance(tokenOwner, spender utils.Address) *big.Int

	// 处理， caller is from msg.Sender in real contract
	Approve(caller, spender utils.Address, value *big.Int)
	Transfer(caller, to utils.Address, value *big.Int) error
	TransferFrom(caller, from, to utils.Address, value *big.Int) error

	// 额外的辅助接口
	info
}

type PledgePool interface {
	AddToken(addr utils.Address) error
	Stake(addr utils.Address, money *big.Int) error
	Withdraw(addr utils.Address, force bool) error
	WithdrawToken(addr, tokenAddr utils.Address, money *big.Int) error

	GetPledgeInfo(addr utils.Address) (*PledgeInfo, error)
	GetAddressByIndex(index uint32) (utils.Address, error)

	info
}

// RoleMgr can be admin by multiple signatures, non-destroy
// now, single sign
type RoleMgr interface {
	// 质押, 元交易
	Pledge(addr utils.Address, money *big.Int, signature []byte) error
	RegisterKeeper(addr utils.Address, blsKey, signature []byte) error
	RegisterProvider(addr utils.Address, signature []byte) error
	RegisterUser(addr utils.Address, blsKey, signature []byte) error

	// by admin
	RegisterToken(taddr utils.Address, asign []byte) error

	// by admin
	CreateGroup(inds []uint64, level uint16, asign []byte) error
	SetFsAddrForGroup(gIndex uint64, fAddr utils.Address, asign []byte) error
	// by keeper and admin
	AddKeeperToGroup(index, gIndex uint64, ksign, asign []byte) error
	// by provider self; from == addrs[index]
	AddProviderToGroup(index, gIndex uint64, psign []byte) error

	// zero means all
	WithdrawToken(caller, tokenAddr utils.Address, money *big.Int) error

	GetTokenByIndex(index uint32) (utils.Address, error)
	GetAddressByIndex(index uint64) (utils.Address, error)
	GetGroupByIndex(index uint64) (uint64, error)
	GetKeepersByIndex(index uint64) ([]uint64, error)

	GetPledgeInfo(addr utils.Address) (*PledgeInfo, error)
	// stop service? not allowed
	//KeeperQuit()
	// stop service? allowed ?
	//ProviderQuit()
	// can quit?
	//UserQuit()
}

// FsMgr manage, create by admin
type FsMgr interface {
	// by user
	CreateFs(user uint64, payToken uint32, asign []byte) error
	// by user
	Recharge(fsIndex uint64, tokenIndex uint32, money *big.Int, sign []byte) error

	// by user sign and keepers sign
	AddOrder(fsIndex, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, sign []byte) error
	SubOrder(fsIndex, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, sign []byte) error
	ProWithdraw(fsIndex, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, sign []byte) error

	KeeperWithdraw(keeperIndex uint64, tokenIndex uint32, amount *big.Int, sign []byte) error

	GetFsIndex(user uint64) (uint64, error)
	GetFsInfo(fsIndex uint64) (uint32, []uint64, error)

	info
}
