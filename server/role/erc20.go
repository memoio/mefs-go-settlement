package role

import (
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

var base = int64(1000000000)

var _ ErcToken = (*ercToken)(nil)

type ercToken struct {
	local utils.Address // contract utils.Address
	admin utils.Address // owner
	money map[utils.Address]*big.Int
}

func NewErcToken(caller utils.Address) (ErcToken, error) {
	// verify
	// get local utils.Address
	local := utils.GetContractAddress(caller, []byte("ErcToken"))

	et := &ercToken{
		admin: caller,
		local: local,
		money: make(map[utils.Address]*big.Int),
	}

	et.money[caller] = new(big.Int).Mul(big.NewInt(base), big.NewInt(base))

	globalMap[local] = et
	return et, nil
}

func (e *ercToken) Transfer(caller, to utils.Address, value *big.Int) error {
	// verify from
	if value.Cmp(zero) < 0 {
		return ErrRes
	}

	val, ok := e.money[caller]
	if !ok {
		return ErrRes
	}

	if val.Cmp(value) < 0 {
		return ErrRes
	}
	val.Sub(val, value)

	valto, ok := e.money[to]
	if !ok {
		valto = big.NewInt(0)
		e.money[to] = valto
	}

	valto.Add(valto, value)

	return nil
}

func (e *ercToken) Balance(from utils.Address) *big.Int {
	res := new(big.Int)

	val, ok := e.money[from]
	if ok {
		res.Add(res, val)
	}

	return res
}

func (e *ercToken) GetContractAddress() utils.Address {
	return e.local
}

func (e *ercToken) GetOwnerAddress() utils.Address {
	return e.admin
}

// 获取taddr对应的erc20上local地址的余额
func getBalance(taddr, query utils.Address) *big.Int {
	eti, ok := globalMap[taddr]
	if !ok {
		return big.NewInt(0)
	}

	et, ok := eti.(ErcToken)
	if !ok {
		return big.NewInt(0)
	}
	return et.Balance(query)
}

// taddr对应的erc20上从from到to
func sendBalance(taddr, caller, to utils.Address, money *big.Int) error {
	eti, ok := globalMap[taddr]
	if !ok {
		return ErrRes
	}

	et, ok := eti.(ErcToken)
	if !ok {
		return ErrRes
	}
	return et.Transfer(caller, to, money)
}
