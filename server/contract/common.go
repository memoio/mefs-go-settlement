package contract

import (
	"errors"
	"math/big"
	"time"

	"github.com/memoio/go-settlement/utils"
)

var log = utils.Logger("contract")

// for compare
var zero = new(big.Int).SetInt64(0)

// erros
var (
	// ErrRes is
	ErrRes              = errors.New("error result")
	ErrInput            = errors.New("input is wrong")
	ErrEmpty            = errors.New("empty, no such value")
	ErrExist            = errors.New("is existed")
	ErrValue            = errors.New("value is less than zero")
	ErrNoSuchAddr       = errors.New("not resgister")
	ErrMisType          = errors.New("mistype contract")
	ErrRoleType         = errors.New("mistype contract")
	ErrBalanceNotEnough = errors.New("balance is insufficient")
	ErrPermission       = errors.New("permission is not right")
	ErrNonce            = errors.New("nonce is not right")
)

// default value

const (
	KiB = 1024
	MiB = 1048576
	GiB = 1073741824
	TiB = 1099511627776

	KB = 1e3
	MB = 1e6
	GB = 1e9
	TB = 1e12

	Day    = 86400
	Hour   = 3600
	Minute = 60

	Wei   = 1
	GWei  = 1e9
	Token = 1e18
)

const (
	DefaultSecurity uint16 = 7

	// DefaultCycle is default cycle: 1 day
	DefaultCycle uint64 = Day

	// DefaultCapacity is default store capacity： 1GB
	DefaultCapacity uint64 = GiB
	// DefaultDuration is default store days： 100 days
	DefaultDuration uint64 = 100 * Day

	// DepositCapacity is provider deposit capacity, 1TB
	DepositCapacity uint64 = TiB

	// ProviderDeposit is provider deposit
	ProviderDeposit uint64 = 1 // Token
	// KeeperDeposit is keeper deposit；
	KeeperDeposit uint64 = 100 // Token

	// ReadPrice is read price 0.01-0.04 $/GB(?)(0.25 rmb-0.5rmb/GB in aliyun oss)
	ReadPrice uint64 = 1e4 * GWei // per MB

	// StorePrice is stored price 1-4$/TB*Month (33 rmb/TB*Month in aliyun oss)
	StorePrice uint64 = 100 * GWei // per MB*day
)

var (
	globalMap map[utils.Address]interface{}
	gtime     uint64
	realTime  bool
)

func init() {
	globalMap = make(map[utils.Address]interface{})
	gtime = uint64(time.Now().Unix())
	realTime = true
}

func SetRealTime(flag bool) {
	realTime = flag
}

func SetTime(t uint64) {
	if t > 0 {
		gtime = t
	}

	gtime = uint64(time.Now().Unix())
}

func GetTime() uint64 {
	if realTime {
		return uint64(time.Now().Unix())
	}

	return gtime
}

func GetMap() map[utils.Address]interface{} {
	return globalMap
}

func getErcToken(addr utils.Address) (ErcToken, error) {
	ri, ok := globalMap[addr]
	if ok {
		r, ok := ri.(ErcToken)
		if ok {
			return r, nil
		}
	}

	return nil, ErrEmpty
}

func getPledgePool(addr utils.Address) (PledgePool, error) {
	pi, ok := globalMap[addr]
	if ok {
		r, ok := pi.(PledgePool)
		if ok {
			return r, nil
		}
	}

	return nil, ErrEmpty
}

func getRoleMgr(addr utils.Address) (RoleMgr, error) {
	ri, ok := globalMap[addr]
	if ok {
		r, ok := ri.(RoleMgr)
		if ok {
			return r, nil
		}
	}

	return nil, ErrEmpty
}

func getFsMgr(addr utils.Address) (FsMgr, error) {
	ri, ok := globalMap[addr]
	if ok {
		r, ok := ri.(FsMgr)
		if ok {
			return r, nil
		}
	}

	return nil, ErrEmpty
}

type info interface {
	GetContractAddress() utils.Address
	GetOwnerAddress() utils.Address
}

// ErcToken is
// 参考：https://zhuanlan.zhihu.com/p/391837660
type ErcToken interface {
	// 查询
	TotalSupply(caller utils.Address) *big.Int
	BalanceOf(caller, tokenOwner utils.Address) *big.Int
	Allowance(caller, tokenOwner, spender utils.Address) *big.Int

	// 处理， caller is from msg.Sender in real contract
	Approve(caller, spender utils.Address, value *big.Int)
	Transfer(caller, to utils.Address, value *big.Int) error
	TransferFrom(caller, from, to utils.Address, value *big.Int) error

	MintToken(caller, target utils.Address, mintedAmount *big.Int) error
	Burn(caller utils.Address, burnAmount *big.Int) error
	AirDrop(caller utils.Address, addrs []utils.Address, money *big.Int) error

	// 额外的辅助接口
	info
}

// PledgePool is for stake and withdraw
type PledgePool interface {
	AddToken(caller, tAddr utils.Address, tIndex uint32) error
	Pledge(caller utils.Address, index uint64, money *big.Int) error
	Withdraw(caller utils.Address, index uint64, tokenIndex uint32, money, lock *big.Int) error

	GetBalance(caller utils.Address, index uint64) []*big.Int
	GetPledge(caller utils.Address) []*big.Int

	info
}

// RoleMgr can be admin by multiple signatures, non-destroy
// now, single sign
type RoleMgr interface {
	// 注册地址，获取序号
	Register(caller, addr utils.Address, sign []byte) error
	// by admin, 注册erc20代币地址
	RegisterToken(caller, taddr utils.Address, asign []byte) error

	// 注册成为keeper角色
	RegisterKeeper(caller utils.Address, index uint64, blsKey, signature []byte) error
	// 注册成为prvider角色
	RegisterProvider(caller utils.Address, index uint64, signature []byte) error
	// 注册成为user角色，从fs contract调用
	RegisterUser(caller utils.Address, index, gIndex uint64, token uint32, blsKey, usign []byte) error

	// 质押,
	Pledge(caller utils.Address, index uint64, money *big.Int, signature []byte) error
	// 取回token对应的代币, money zero means all
	Withdraw(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int, signature []byte) error

	// 创建组，by admin
	CreateGroup(caller utils.Address, inds []uint64, level uint16, asign []byte) error
	// 向组中添加keeper，by keeper and admin
	AddKeeperToGroup(caller utils.Address, index, gIndex uint64, ksign, asign []byte) error
	// 向组中添加provider
	AddProviderToGroup(caller utils.Address, index, gIndex uint64, psign []byte) error

	AddOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error
	SubOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error

	//  查询相关
	// 获取addr地址的相关信息
	GetIndex(caller, addr utils.Address) (uint64, error)
	GetInfo(caller utils.Address, index uint64) (*baseInfo, utils.Address, error)
	GetTokenIndex(caller, taddr utils.Address) (uint32, error)
	GetTokenAddress(caller utils.Address, index uint32) (utils.Address, error)
	GetGroupInfo(caller utils.Address, gindex uint64) (*groupInfo, error)

	GetBalance(caller utils.Address, index uint64) ([]*big.Int, error)

	GetPledgeAddress(caller utils.Address) utils.Address
	GetPledge(caller utils.Address) (*big.Int, *big.Int, []*big.Int)
	GetAllTokens(caller utils.Address) []utils.Address
	GetAllAddrs(caller utils.Address) []utils.Address
	GetAllGroups(caller utils.Address) []*groupInfo

	GetFoundation(caller utils.Address) utils.Address

	info
	// stop service? not allowed
	//KeeperQuit()
	// stop service? allowed ?
	//ProviderQuit()
	// can quit?
	//UserQuit()
}

// FsMgr manage, create by admin
type FsMgr interface {
	// by roleMgr contract
	AddKeeper(caller utils.Address, kindex uint64) error
	CreateFs(caller utils.Address, user uint64, payToken uint32) error
	AddOrder(caller utils.Address, kindex, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error
	SubOrder(caller utils.Address, kindex, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error

	// by user
	Recharge(caller utils.Address, user uint64, tokenIndex uint32, money *big.Int, sign []byte) error
	Withdraw(caller utils.Address, keeperIndex uint64, tokenIndex uint32, amount *big.Int, sign []byte) error
	ProWithdraw(caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, ksigns [][]byte) error

	GetFsInfo(caller utils.Address, user uint64) (*fsInfo, error)
	// return avalilable, locked, paid
	GetBalance(caller utils.Address, index uint64, tIndex uint32) (*big.Int, *big.Int, *big.Int)

	info
}
