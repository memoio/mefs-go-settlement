package contract

import (
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

var _ ErcToken = (*ercToken)(nil)

type twoKey struct {
	owner   utils.Address
	spender utils.Address
}

type ercToken struct {
	local       utils.Address // contract utils.Address
	admin       utils.Address // owner
	totalSupply *big.Int
	money       map[utils.Address]*big.Int
	allowed     map[twoKey]*big.Int
}

// NewErcToken create
func NewErcToken(caller utils.Address) ErcToken {
	// verify
	// get local utils.Address
	local := utils.GetContractAddress(caller, []byte("ErcToken"))

	et := &ercToken{
		admin:       caller,
		local:       local,
		money:       make(map[utils.Address]*big.Int),
		allowed:     make(map[twoKey]*big.Int),
		totalSupply: new(big.Int).Mul(big.NewInt(Token), big.NewInt(1e10)),
	}

	et.money[caller] = new(big.Int).Set(et.totalSupply)

	globalMap[local] = et
	return et
}

func (e *ercToken) TotalSupply(caller utils.Address) *big.Int {
	return new(big.Int).Set(e.totalSupply)
}

func (e *ercToken) BalanceOf(caller utils.Address, from utils.Address) *big.Int {
	res := new(big.Int)
	val, ok := e.money[from]
	if ok {
		res.Add(res, val)
	}

	return res
}

func (e *ercToken) Allowance(caller, tokenOwner, spender utils.Address) *big.Int {
	res := new(big.Int)
	tKey := twoKey{
		owner:   tokenOwner,
		spender: spender,
	}

	val, ok := e.allowed[tKey]
	if ok {
		res.Add(res, val)
	}
	return res
}

func (e *ercToken) Transfer(caller, to utils.Address, value *big.Int) error {
	// verify to is not zero
	// verify value > 0
	if value.Cmp(zero) < 0 {
		return ErrValue
	}

	// verify money is enough
	val, ok := e.money[caller]
	if !ok {
		return ErrEmpty
	}
	if val.Cmp(value) < 0 {
		return ErrBalanceNotEnough
	}

	// sub from caller
	val.Sub(val, value)

	// add to to
	valto, ok := e.money[to]
	if !ok {
		valto = big.NewInt(0)
		e.money[to] = valto
	}
	valto.Add(valto, value)

	return nil
}

// 用于合约账户将erc token转入合约账户中
func (e *ercToken) Approve(caller, spender utils.Address, value *big.Int) {
	if value.Cmp(zero) > 0 {
		tKey := twoKey{
			owner:   caller,
			spender: spender,
		}
		e.allowed[tKey] = new(big.Int).Set(value)
	}
}

func (e *ercToken) TransferFrom(caller, from, to utils.Address, value *big.Int) error {
	// verify from and to is not zero address
	// verify value > 0
	if value.Cmp(zero) < 0 {
		return ErrValue
	}

	// verify money is enough
	val, ok := e.money[from]
	if !ok {
		return ErrEmpty
	}
	if val.Cmp(value) < 0 {
		return ErrBalanceNotEnough
	}

	// verify money is allowed by caller
	tKey := twoKey{
		owner:   from,
		spender: caller,
	}
	aval, ok := e.allowed[tKey]
	if !ok {
		return ErrRes
	}
	if aval.Cmp(value) < 0 {
		return ErrPermission
	}

	// sub from from
	val.Sub(val, value)
	// sub from allowed
	aval.Sub(aval, value)

	// add to to
	valto, ok := e.money[to]
	if !ok {
		valto = big.NewInt(0)
		e.money[to] = valto
	}
	valto.Add(valto, value)

	return nil
}

// 增发
func (e *ercToken) MintToken(caller, target utils.Address, mintedAmount *big.Int) error {
	if caller != e.admin {
		return ErrPermission
	}

	bal, ok := e.money[target]
	if ok {
		bal.Add(bal, mintedAmount)
	} else {
		bal = new(big.Int).Set(mintedAmount)
		e.money[target] = bal
	}

	e.totalSupply.Add(e.totalSupply, mintedAmount)

	return nil
}

// 空投
func (e *ercToken) AirDrop(caller utils.Address, addrs []utils.Address, money *big.Int) error {
	if caller != e.admin {
		return ErrPermission
	}

	for _, addr := range addrs {
		e.Transfer(e.admin, addr, money)
	}

	return nil
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
	return et.BalanceOf(query, query)
}

// taddr对应的erc20上从from到to
func sendBalance(taddr, caller, to utils.Address, money *big.Int) error {
	eti, ok := globalMap[taddr]
	if !ok {
		return ErrEmpty
	}

	et, ok := eti.(ErcToken)
	if !ok {
		return ErrMisType
	}
	return et.Transfer(caller, to, money)
}

func sendBalanceFrom(taddr, caller, from, to utils.Address, money *big.Int) error {
	eti, ok := globalMap[taddr]
	if !ok {
		return ErrEmpty
	}

	et, ok := eti.(ErcToken)
	if !ok {
		return ErrMisType
	}
	return et.TransferFrom(caller, from, to, money)
}

func approve(taddr, caller, spender utils.Address, money *big.Int) error {
	eti, ok := globalMap[taddr]
	if !ok {
		return ErrEmpty
	}

	et, ok := eti.(ErcToken)
	if !ok {
		return ErrMisType
	}
	et.Approve(caller, spender, money)

	return nil
}
