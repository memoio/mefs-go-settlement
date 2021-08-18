package contract

import (
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

type rewardInfo struct {
	rewardAccum *big.Int // 本代币的accumulator
	lastReward  *big.Int // 上一次变更时的本代币奖励总量
}

var _ PledgePool = (*pledgeMgr)(nil)

type pledgeMgr struct {
	owner       utils.Address
	local       utils.Address            // contract utils.Address
	token       uint32                   // largest token
	tokens      []utils.Address          // 所有用作使用代币的信息,0为主代币的代币地址
	totalPledge *big.Int                 // 映射代币的发行总量
	amount      map[multiKey]*rewardInfo // 所有质押的人的信息
	tInfo       map[uint32]*rewardInfo
}

func NewPledgeMgr(caller, ptoken utils.Address) *pledgeMgr {

	local := utils.GetContractAddress(caller, []byte("PledgePool"))

	pm := &pledgeMgr{
		owner:       caller,
		local:       local,
		totalPledge: new(big.Int),
		token:       1,
		tokens:      make([]utils.Address, 0, 1),
		tInfo:       make(map[uint32]*rewardInfo),
		amount:      make(map[multiKey]*rewardInfo),
	}

	bal := getBalance(ptoken, local)
	pm.tInfo[0] = &rewardInfo{
		rewardAccum: big.NewInt(0),
		lastReward:  bal,
	}
	pm.tokens = append(pm.tokens, ptoken)

	globalMap[local] = pm

	return pm
}

func (p *pledgeMgr) GetContractAddress() utils.Address {
	return p.local
}

func (p *pledgeMgr) GetOwnerAddress() utils.Address {
	return p.owner
}

func (p *pledgeMgr) GetPledge(caller utils.Address) []*big.Int {
	var res []*big.Int
	for i := range p.tokens {
		ti := p.tInfo[uint32(i)]
		res = append(res, ti.lastReward)
	}
	return res
}

func (p *pledgeMgr) GetBalance(caller utils.Address, index uint64) []*big.Int {
	totalPledge := new(big.Int).Set(p.totalPledge)
	mk := multiKey{
		roleIndex:  index,
		tokenIndex: 0,
	}

	p0, ok := p.amount[mk]
	if !ok {
		return nil
	}

	amount := new(big.Int).Set(p0.lastReward)

	res := make([]*big.Int, p.token)
	for i, taddr := range p.tokens {
		// update tokenInfo
		ti := p.tInfo[uint32(i)]

		val := new(big.Int).Set(ti.rewardAccum)
		bal := getBalance(taddr, p.local)
		bal.Sub(bal, ti.lastReward)
		if bal.Cmp(zero) > 0 && totalPledge.Cmp(zero) > 0 {
			bal.Div(bal, totalPledge)
			val.Add(val, bal)
		}

		mki := multiKey{
			roleIndex:  index,
			tokenIndex: uint32(i),
		}

		rew, ok := p.amount[mki]
		if ok {
			val.Sub(val, rew.rewardAccum)
			val.Mul(val, amount)
			val.Add(val, rew.lastReward)
		}

		res[i] = val
	}

	return res
}

// by owner
func (p *pledgeMgr) AddToken(caller, tAddr utils.Address, tokenIndex uint32) error {
	if caller != p.owner {
		return ErrPermission
	}

	if p.token != tokenIndex {
		return ErrInput
	}

	bal := getBalance(tAddr, p.local)
	ti := &rewardInfo{
		rewardAccum: big.NewInt(0),
		lastReward:  bal,
	}

	p.tInfo[tokenIndex] = ti
	p.tokens = append(p.tokens, tAddr)
	p.token = tokenIndex + 1

	return nil
}

func (p *pledgeMgr) Pledge(caller utils.Address, index uint64, money *big.Int) error {
	if caller != p.owner {
		return ErrPermission
	}

	// chekc money > 0
	if money.Cmp(zero) < 0 {
		return ErrRes
	}

	rm, err := getRoleMgr(p.owner)
	if err != nil {
		return err
	}

	// update value
	_, addr, err := rm.GetInfo(p.local, index)
	if err != nil {
		return err
	}

	mk := multiKey{
		roleIndex:  index,
		tokenIndex: 0,
	}

	p0, ok := p.amount[mk]
	if !ok {
		p0 = &rewardInfo{
			rewardAccum: big.NewInt(0),
			lastReward:  big.NewInt(0),
		}
		p.amount[mk] = p0
	}

	if p.totalPledge.Cmp(zero) > 0 {
		totalPledge := new(big.Int).Set(p.totalPledge)
		amount := new(big.Int).Set(p0.lastReward)

		// 更新token acc，结算奖励
		for i, taddr := range p.tokens {
			// update tokenInfo
			ti := p.tInfo[uint32(i)]

			mki := multiKey{
				roleIndex:  index,
				tokenIndex: uint32(i),
			}

			rew, ok := p.amount[mki]
			if !ok {
				rew = &rewardInfo{
					rewardAccum: big.NewInt(0),
					lastReward:  big.NewInt(0),
				}
				p.amount[mki] = rew
			}

			bal := getBalance(taddr, p.local)
			tv := new(big.Int).Sub(bal, ti.lastReward)
			if tv.Cmp(zero) > 0 && totalPledge.Cmp(zero) > 0 {
				tv.Div(tv, totalPledge)
				ti.rewardAccum.Add(ti.rewardAccum, tv)
			}

			ti.lastReward = bal // update to lastest

			res := new(big.Int).Sub(ti.rewardAccum, rew.rewardAccum)
			res.Mul(res, amount)
			rew.lastReward.Add(rew.lastReward, res)            // 添加奖励
			rew.rewardAccum = new(big.Int).Set(ti.rewardAccum) // 更新acc
		}
	}

	err = sendBalanceFrom(p.tokens[0], p.local, addr, p.local, money)
	if err != nil {
		return err
	}

	// update
	ti := p.tInfo[0]
	ti.lastReward.Add(ti.lastReward, money)
	p0.lastReward.Add(p0.lastReward, money)
	p.totalPledge.Add(p.totalPledge, money)

	return nil
}

// Withdraw tokens
func (p *pledgeMgr) Withdraw(caller utils.Address, index uint64, tokenIndex uint32, money, lock *big.Int) error {
	if money.Cmp(zero) < 0 {
		return ErrInput
	}

	if tokenIndex > uint32(len(p.tokens)) {
		return ErrInput
	}

	rm, err := getRoleMgr(p.owner)
	if err != nil {
		return err
	}

	// update value
	_, addr, err := rm.GetInfo(p.local, index)
	if err != nil {
		return err
	}

	mk := multiKey{
		roleIndex:  index,
		tokenIndex: 0,
	}

	if p.totalPledge.Cmp(zero) > 0 {
		p0, ok := p.amount[mk]
		if !ok {
			return ErrEmpty
		}

		totalPledge := new(big.Int).Set(p.totalPledge)
		amount := new(big.Int).Set(p0.lastReward)

		// 更新token acc，结算奖励
		for i, taddr := range p.tokens {
			if tokenIndex != 0 && tokenIndex != uint32(i) {
				continue
			}

			// update tokenInfo
			ti := p.tInfo[uint32(i)]

			bal := getBalance(taddr, p.local)
			tv := new(big.Int).Sub(bal, ti.lastReward)
			if tv.Cmp(zero) > 0 && totalPledge.Cmp(zero) > 0 {
				tv.Div(tv, totalPledge)
				ti.rewardAccum.Add(ti.rewardAccum, tv)
			}

			ti.lastReward = bal // update to lastest

			mki := multiKey{
				roleIndex:  index,
				tokenIndex: uint32(i),
			}

			rew, ok := p.amount[mki]
			if !ok {
				rew = &rewardInfo{
					rewardAccum: big.NewInt(0),
					lastReward:  big.NewInt(0),
				}
				p.amount[mki] = rew
			}

			res := new(big.Int).Sub(ti.rewardAccum, rew.rewardAccum)
			res.Mul(res, amount)
			rew.lastReward.Add(rew.lastReward, res)            // 添加奖励
			rew.rewardAccum = new(big.Int).Set(ti.rewardAccum) // 更新acc
		}
	}

	mki := multiKey{
		roleIndex:  index,
		tokenIndex: tokenIndex,
	}

	pi, ok := p.amount[mki]
	if !ok {
		return ErrEmpty
	}

	rw := new(big.Int).Set(pi.lastReward)
	if tokenIndex == 0 {
		rw.Sub(rw, lock)
	}
	if money.Cmp(zero) > 0 && money.Cmp(rw) < 0 {
		rw.Set(money)
	}

	if rw.Cmp(zero) > 0 {
		tAddr := p.tokens[tokenIndex]
		err = sendBalance(tAddr, p.local, addr, rw)
		if err != nil {
			return err
		}

		// update token
		ti := p.tInfo[tokenIndex]
		ti.lastReward.Sub(ti.lastReward, rw)
		// update value
		pi.lastReward.Sub(pi.lastReward, rw)

		if tokenIndex == 0 {
			p.totalPledge.Sub(p.totalPledge, rw)
		}
	}

	return nil
}
