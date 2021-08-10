package contract

import (
	"fmt"
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

type balance struct {
	amount *big.Int // total
	locked *big.Int // 锁定
}

type TokenInfo struct {
	index            uint32
	rewardAccum      *big.Int // 本代币的accumulator
	lastRewardSupply *big.Int // 上一次变更时的本代币奖励总量
}

func (t TokenInfo) update(amount, totalPledge *big.Int) {
	tv := new(big.Int)
	tv.Add(tv, amount)
	tv.Sub(tv, t.lastRewardSupply)
	if tv.Cmp(zero) > 0 && totalPledge.Cmp(zero) > 0 {
		tv.Div(tv, totalPledge)
		t.rewardAccum.Add(t.rewardAccum, tv)
		fmt.Println("acc has: ", t.rewardAccum.String(), amount.String())
	}

	t.lastRewardSupply = amount // update to lastest
}

type PledgeInfo struct {
	amount      *big.Int // 映射代币的余额
	locked      *big.Int
	rewardAccum []*big.Int // 质押开始时的accumulator，索引和代币对应
	rewards     []*big.Int // 已结算的奖励，索引和代币对应
}

// PledgePool is for stake and withdraw
type PledgePool interface {
	AddToken(addr utils.Address) error
	Stake(addr utils.Address, money *big.Int) error
	Withdraw(addr utils.Address, force bool) error
	WithdrawToken(addr, tokenAddr utils.Address, money *big.Int) error

	GetPledgeInfo(addr utils.Address) (*PledgeInfo, error)
	GetAddressByIndex(index uint32) (utils.Address, error)

	info
}

var _ PledgePool = (*pledgeMgr)(nil)

type pledgeMgr struct {
	admin       utils.Address
	local       utils.Address                 // contract utils.Address
	totalPledge *big.Int                      // 映射代币的发行总量
	pledges     map[utils.Address]*PledgeInfo // 所有质押的人的信息
	tokens      []utils.Address               // 所有用作使用代币的信息,0为主代币的代币地址
	tInfo       map[utils.Address]*TokenInfo
}

func NewPledgeMgr(caller, ptoken utils.Address) *pledgeMgr {
	local := utils.GetContractAddress(caller, []byte("PledgePool"))

	pm := &pledgeMgr{
		admin:       caller,
		local:       local,
		totalPledge: new(big.Int),
		pledges:     make(map[utils.Address]*PledgeInfo),
		tokens:      make([]utils.Address, 0, 1),
		tInfo:       make(map[utils.Address]*TokenInfo),
	}

	pm.tokens = append(pm.tokens, ptoken)

	globalMap[local] = pm

	return pm
}

func (p *pledgeMgr) GetContractAddress() utils.Address {
	return p.local
}

func (p *pledgeMgr) GetOwnerAddress() utils.Address {
	return p.admin
}

func (p *pledgeMgr) GetTokenAddress(index uint32) (utils.Address, error) {
	if index > uint32(len(p.tokens)) {
		return utils.Address{}, ErrRes
	}
	return p.tokens[index], nil
}

// by owner
func (p *pledgeMgr) AddToken(addr utils.Address) error {
	_, ok := p.tInfo[addr]
	if ok {
		// exist
		return ErrRes
	}

	bal := getBalance(addr, p.local)
	ti := &TokenInfo{
		index:            uint32(len(p.tokens)),
		rewardAccum:      big.NewInt(0),
		lastRewardSupply: bal,
	}

	p.tokens = append(p.tokens, addr)
	p.tInfo[addr] = ti

	return nil
}

func (p *pledgeMgr) Stake(addr utils.Address, money *big.Int) error {
	// chekc money > 0

	if money.Cmp(zero) < 0 {
		return ErrRes
	}

	um := getBalance(p.tokens[0], addr)
	if um.Cmp(money) < 0 {
		fmt.Println(um)
		return ErrRes
	}

	tlen := len(p.tokens)

	// update tInfo first
	for taddr, ti := range p.tInfo {
		bal := getBalance(taddr, p.local)

		ti.update(bal, p.totalPledge)
	}

	pi, ok := p.pledges[addr]
	if ok {
		// 先结算奖励
		if pi.amount.Cmp(zero) > 0 {
			for i, acc := range pi.rewardAccum {
				taddr := p.tokens[i+1]
				ti, ok := p.tInfo[taddr]
				if !ok {
					return ErrRes
				}

				res := new(big.Int)
				res.Add(res, ti.rewardAccum)
				res.Sub(res, acc)
				res.Mul(res, pi.amount)
				pi.rewards[i].Add(pi.rewards[i], res) // 添加奖励

				ra := new(big.Int)
				ra.Add(ra, ti.rewardAccum)
				pi.rewardAccum[i] = ra // 更新acc
				fmt.Println("rewards has: ", pi.rewards[i], pi.rewardAccum[i])
			}
		}
	} else {
		pi = &PledgeInfo{
			amount:      new(big.Int),
			rewardAccum: make([]*big.Int, 0, tlen),
			rewards:     make([]*big.Int, 0, tlen),
		}

		p.pledges[addr] = pi
	}

	err := sendBalance(p.tokens[0], addr, p.local, money)
	if err != nil {
		return err
	}
	pi.amount.Add(pi.amount, money)
	p.totalPledge.Add(p.totalPledge, money)

	// 填充rewardAccm
	for i, taddr := range p.tokens[1:] {
		ti, ok := p.tInfo[taddr]
		if !ok {
			return nil
		}

		ra := new(big.Int)
		ra.Add(ra, ti.rewardAccum)

		if len(pi.rewardAccum) <= i {
			pi.rewardAccum = append(pi.rewardAccum, ra)
			pi.rewards = append(pi.rewards, new(big.Int))
		}
	}

	fmt.Println("has: ", pi.amount.String(), money.String())

	return nil
}

// force means transfer for each token
func (p *pledgeMgr) Withdraw(addr utils.Address, force bool) error {
	// update tInfo first
	for taddr, ti := range p.tInfo {
		bal := getBalance(taddr, p.local)

		ti.update(bal, p.totalPledge)
	}

	pi, ok := p.pledges[addr]
	if ok {
		// 结算奖励
		if pi.amount.Cmp(zero) > 0 || force {
			for i, acc := range pi.rewardAccum {
				taddr := p.tokens[i+1]
				ti, ok := p.tInfo[taddr]
				if !ok {
					return ErrRes
				}

				if pi.amount.Cmp(zero) > 0 {
					res := new(big.Int)
					res.Add(res, ti.rewardAccum)
					res.Sub(res, acc)
					res.Mul(res, pi.amount)
					pi.rewards[i].Add(pi.rewards[i], res) // 添加奖励
				}

				fmt.Println("addr: ", taddr.String(), " has: ", pi.rewards[i])

				if force && pi.rewards[i].Cmp(zero) > 0 {
					sendBalance(taddr, p.local, addr, pi.rewards[i])
					ti.lastRewardSupply.Sub(ti.lastRewardSupply, pi.rewards[i]) // query balance?
					pi.rewards[i] = new(big.Int)
				}

				ra := new(big.Int)
				ra.Add(ra, ti.rewardAccum)
				pi.rewardAccum[i] = ra // 更新acc
			}

			// 转账
			if pi.amount.Cmp(zero) > 0 {
				sendBalance(p.tokens[0], p.local, addr, pi.amount)
				p.totalPledge.Sub(p.totalPledge, pi.amount)
				pi.amount = big.NewInt(0)
			}
		}
	}

	return nil
}

func (p *pledgeMgr) Lock(addr utils.Address, money *big.Int) error {
	pi, ok := p.pledges[addr]
	if ok {
		if money.Cmp(zero) >= 0 && pi.locked.Cmp(zero) >= 0 {
			money.Add(money, pi.locked)
			if pi.amount.Cmp(money) >= 0 {
				pi.locked = money
				return nil
			}
		}
	}

	return ErrRes
}

// by contract in
func (p *pledgeMgr) UnLock(addr utils.Address, money *big.Int) error {
	pi, ok := p.pledges[addr]
	if ok {
		if money.Cmp(zero) >= 0 && pi.locked.Cmp(money) >= 0 {
			pi.locked.Sub(pi.locked, money)
			return nil
		}
	}

	return ErrRes
}

func (p *pledgeMgr) WithdrawToken(addr, tokenAddr utils.Address, money *big.Int) error {
	pi, ok := p.pledges[addr]
	if !ok {
		return ErrRes
	}

	ti, ok := p.tInfo[tokenAddr]
	if !ok {
		return ErrRes
	}

	rw := pi.rewards[ti.index]

	res := big.NewInt(0)

	if money.Cmp(rw) > 0 {
		res.Add(res, rw)
	} else {
		res.Add(res, money)
	}

	err := sendBalance(tokenAddr, p.local, addr, res)
	if err != nil {
		return err
	}
	pi.rewards[ti.index] = big.NewInt(0)

	return nil
}

func (p *pledgeMgr) GetPledgeInfo(addr utils.Address) (*PledgeInfo, error) {
	pi, ok := p.pledges[addr]
	if ok {
		return pi, nil
	}

	return nil, ErrRes
}

func (p *pledgeMgr) GetPledgeMoney(addr utils.Address) *big.Int {
	res := big.NewInt(0)
	pi, ok := p.pledges[addr]
	if ok {
		res.Add(res, pi.amount)
	}

	return res
}

func (p *pledgeMgr) GetAddressByIndex(index uint32) (utils.Address, error) {
	if index >= uint32(len(p.tokens)) {
		return utils.Address{}, ErrRes
	}

	return p.tokens[index], nil
}
