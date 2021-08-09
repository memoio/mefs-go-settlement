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

type ErcToken interface {
	Transfer(caller, to utils.Address, value *big.Int) error
	Balance(from utils.Address) *big.Int

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

// can be admin by multiple signatures, non-destroy
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
