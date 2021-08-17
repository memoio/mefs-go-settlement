package contract

import (
	"math/big"
	"time"

	"github.com/memoio/go-settlement/utils"
)

const (
	roleUser     uint8 = 1
	roleProvider uint8 = 2
	roleKeeper   uint8 = 3
)

type baseInfo struct {
	isActive bool   // 是否激活
	isBanned bool   // 是否禁用
	roleType uint8  // 0 account, 1, user, 2, provider, 3 keeper
	index    uint64 // 序列号
	gIndex   uint64 // 所属group
	extra    []byte // for offline

	rewardAccum []*big.Int // 质押开始时的accumulator，索引和代币对应; 0号为主代币
	rewards     []*big.Int // 已结算的奖励，索引和代币对应
}

type tokenInfo struct {
	index            uint32
	rewardAccum      *big.Int // 本代币的accumulator
	lastRewardSupply *big.Int // 上一次变更时的本代币奖励总量
}

// groupInfo has
type groupInfo struct {
	isActive   bool
	isBanned   bool     // 组是否已被禁用
	level      uint16   // security level
	keepers    []uint64 // 里面有哪些Keeper
	providers  []uint64 // 有哪些provider
	size       *big.Int // storeSize
	price      *big.Int
	period     uint64
	lastTime   uint64              // 上次分润时间
	tAcc       map[uint32]*big.Int // 每次分润后归0
	totalCount uint64
	count      map[uint64]uint64 //   记录keeper触发的次数，用于分润
	fsAddr     utils.Address     // fs contract addr
}

var _ RoleMgr = (*roleMgr)(nil)

type roleMgr struct {
	local      utils.Address // contract of this mgr
	admin      utils.Address // owner
	foundation utils.Address // foundation address

	addrs []utils.Address // all
	info  map[utils.Address]*baseInfo

	// manage group
	groups []*groupInfo

	// 代币信息,0为主代币的代币地址
	tokens []utils.Address
	tInfo  map[utils.Address]*tokenInfo

	pledgeKeeper *big.Int // pledgeMoney for keeper
	pledgePro    *big.Int // pledgeMoney for provider
}

// NewRoleMgr can be admin by mutiple signatures
func NewRoleMgr(caller, foundation, primaryToken utils.Address, kPledge, pPledge *big.Int) RoleMgr {
	// generate local utils.Address from
	local := utils.GetContractAddress(caller, []byte("RoleMgr"))

	rm := &roleMgr{
		admin:      caller,
		local:      local,
		foundation: foundation,

		addrs:  make([]utils.Address, 0, 128),
		info:   make(map[utils.Address]*baseInfo),
		groups: make([]*groupInfo, 0, 1),

		tokens: make([]utils.Address, 0, 1),
		tInfo:  make(map[utils.Address]*tokenInfo),

		pledgeKeeper: kPledge,
		pledgePro:    pPledge,
	}

	ti := &tokenInfo{
		index:            0,
		rewardAccum:      big.NewInt(0),
		lastRewardSupply: getBalance(primaryToken, local),
	}

	rm.tokens = append(rm.tokens, primaryToken)
	rm.tInfo[primaryToken] = ti

	globalMap[local] = rm
	return rm
}

func (r *roleMgr) GetContractAddress() utils.Address {
	return r.local
}

func (r *roleMgr) GetOwnerAddress() utils.Address {
	return r.admin
}

func (r *roleMgr) GetFoundation() utils.Address {
	return r.foundation
}

func (r *roleMgr) GetIndex(caller, addr utils.Address) (uint64, error) {
	bi, ok := r.info[addr]
	if ok {
		return bi.index, nil
	}

	return 0, ErrNoSuchAddr
}

func (r *roleMgr) GetTokenIndex(caller, addr utils.Address) (uint32, error) {
	ti, ok := r.tInfo[addr]
	if ok {
		return ti.index, nil
	}

	return 0, ErrNoSuchAddr
}

func (r *roleMgr) GetAllTokens(caller utils.Address) []utils.Address {
	return r.tokens
}

func (r *roleMgr) GetAllAddrs(caller utils.Address) []utils.Address {
	return r.addrs
}

func (r *roleMgr) GetAllGroups(caller utils.Address) []*groupInfo {
	return r.groups
}
func (r *roleMgr) GetPledge(caller utils.Address) (*big.Int, *big.Int, []*big.Int) {
	var res []*big.Int
	for _, taddr := range r.tokens {
		ti := r.tInfo[taddr]
		res = append(res, ti.lastRewardSupply)
	}
	return r.pledgeKeeper, r.pledgePro, res
}

func (r *roleMgr) GetGroupInfoByIndex(caller utils.Address, index uint64) (*groupInfo, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}
	return r.groups[index], nil
}

func (r *roleMgr) getGroupInfo(index uint64) (*groupInfo, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}

	gi := r.groups[index]

	if !gi.isActive || gi.isBanned {
		return nil, ErrPermission
	}

	return gi, nil
}

func (r *roleMgr) GetInfoByIndex(caller utils.Address, index uint64) (*baseInfo, utils.Address, error) {
	var res utils.Address
	if int(index) >= len(r.addrs) {
		return nil, res, ErrInput
	}

	addr := r.addrs[index]
	ki, ok := r.info[addr]
	if !ok {
		return nil, res, ErrEmpty
	}

	return ki, addr, nil
}

func (r *roleMgr) getInfo(index uint64) (*baseInfo, error) {
	if int(index) >= len(r.addrs) {
		return nil, ErrInput
	}

	addr := r.addrs[index]
	bi, ok := r.info[addr]
	if !ok {
		return nil, ErrEmpty
	}

	if bi.isBanned {
		return nil, ErrPermission
	}

	return bi, nil
}

func (r *roleMgr) GetInfo(caller, addr utils.Address) (*baseInfo, error) {
	bi, ok := r.info[addr]
	if !ok {
		return nil, ErrNoSuchAddr
	}

	return bi, nil
}

func (r *roleMgr) RegisterToken(caller, taddr utils.Address, sign []byte) error {
	// verify sign
	_, ok := r.tInfo[taddr]
	if ok {
		// exist
		return ErrExist
	}

	bal := getBalance(taddr, r.local)
	ti := &tokenInfo{
		index:            uint32(len(r.tokens)),
		rewardAccum:      big.NewInt(0),
		lastRewardSupply: bal,
	}

	r.tokens = append(r.tokens, taddr)
	r.tInfo[taddr] = ti

	return nil
}

func (r *roleMgr) Register(caller, addr utils.Address, signature []byte) error {
	// verify sign
	_, ok := r.info[addr]
	if ok {
		return ErrExist
	}

	bi := &baseInfo{
		roleType:    0,
		index:       uint64(len(r.addrs)),
		rewardAccum: make([]*big.Int, len(r.tokens)),
		rewards:     make([]*big.Int, len(r.tokens)),
	}

	for i, tAddr := range r.tokens {
		ti, ok := r.tInfo[tAddr]
		if !ok {
			return ErrEmpty
		}

		bi.rewardAccum[i] = new(big.Int).Set(ti.rewardAccum)
		bi.rewards[i] = new(big.Int)
	}

	r.addrs = append(r.addrs, addr)
	r.info[addr] = bi

	return nil
}

func (r *roleMgr) GetBalance(caller utils.Address, index uint64) ([]*big.Int, error) {
	bi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return nil, err
	}

	t0 := r.tInfo[r.tokens[0]]
	totalPledge := new(big.Int).Set(t0.lastRewardSupply)
	amount := new(big.Int).Set(bi.rewards[0])

	res := make([]*big.Int, len(bi.rewards))
	for i := 0; i < len(bi.rewards); i++ {
		taddr := r.tokens[i]
		ti, ok := r.tInfo[taddr]
		if !ok {
			return nil, ErrEmpty
		}

		val := new(big.Int).Set(ti.rewardAccum)
		bal := getBalance(taddr, r.local)
		bal.Sub(bal, ti.lastRewardSupply)
		if bal.Cmp(zero) > 0 {
			bal.Div(bal, totalPledge)
			val.Add(val, bal)
		}
		val.Sub(val, bi.rewardAccum[i])
		val.Mul(val, amount)
		val.Add(val, bi.rewards[i])
		res[i] = val
	}

	return res, nil
}

func (r *roleMgr) update(index uint64) error {
	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	t0 := r.tInfo[r.tokens[0]]
	totalPledge := new(big.Int).Set(t0.lastRewardSupply)
	amount := new(big.Int)
	if len(bi.rewards) > 0 {
		amount.Set(bi.rewards[0])
	}

	// 更新token acc，结算奖励
	for i, taddr := range r.tokens {
		// update tokenInfo
		ti := r.tInfo[taddr]

		bal := getBalance(taddr, r.local)

		tv := new(big.Int).Sub(bal, ti.lastRewardSupply)
		if tv.Cmp(zero) > 0 && totalPledge.Cmp(zero) > 0 {
			tv.Div(tv, totalPledge)
			ti.rewardAccum.Add(ti.rewardAccum, tv)
		}

		ti.lastRewardSupply = bal // update to lastest

		// update
		if len(bi.rewardAccum) <= i {
			bi.rewardAccum = append(bi.rewardAccum, new(big.Int))
			bi.rewards = append(bi.rewards, new(big.Int))
		}

		res := new(big.Int).Sub(ti.rewardAccum, bi.rewardAccum[i])
		res.Mul(res, amount)
		bi.rewards[i].Add(bi.rewards[i], res)                // 添加奖励
		bi.rewardAccum[i] = new(big.Int).Set(ti.rewardAccum) // 更新acc
	}
	return nil
}

func (r *roleMgr) Pledge(caller utils.Address, index uint64, money *big.Int, sign []byte) error {
	// chekc money > 0
	return r.Recharge(caller, index, 0, money, sign)
}

func (r *roleMgr) Recharge(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int, sign []byte) error {
	// verify sign
	// chekc money > 0
	if money.Cmp(zero) < 0 {
		return ErrValue
	}

	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrEmpty
	}

	if tokenIndex == 0 {
		err := r.update(index)
		if err != nil {
			return err
		}
	}

	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	// update value
	addr := r.addrs[bi.index]
	err = sendBalanceFrom(r.tokens[tokenIndex], r.local, addr, r.local, money)
	if err != nil {
		return err
	}
	bi.rewards[tokenIndex].Add(bi.rewards[tokenIndex], money)
	ti := r.tInfo[r.tokens[tokenIndex]]
	ti.lastRewardSupply.Add(ti.lastRewardSupply, money)

	return nil
}

func (r *roleMgr) SetPledgeMoney(caller utils.Address, kPledge, pPledge *big.Int, signature []byte) error {
	// verify sign(hash(kPLedge,pPledge))

	r.pledgeKeeper = kPledge
	r.pledgePro = pPledge
	return nil
}

func (r *roleMgr) RegisterKeeper(caller utils.Address, index uint64, blsKey, signature []byte) error {
	bi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	// registered
	if bi.roleType != 0 {
		return ErrRoleType
	}

	if bi.rewards[0].Cmp(r.pledgeKeeper) < 0 {
		return ErrBalanceNotEnough
	}

	bi.roleType = roleKeeper
	bi.extra = blsKey

	return nil
}

func (r *roleMgr) RegisterProvider(caller utils.Address, index uint64, signature []byte) error {
	// verify sign
	bi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	// registered
	if bi.roleType != 0 {
		return ErrRoleType
	}

	if bi.rewards[0].Cmp(r.pledgePro) < 0 {
		return ErrBalanceNotEnough
	}

	bi.roleType = roleProvider

	return nil
}

func (r *roleMgr) RegisterUser(caller utils.Address, index, gIndex uint64, payToken uint32, blsKey, sign []byte) error {
	// verify sign

	bi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	// registered
	if bi.roleType != 0 {
		return ErrRoleType
	}

	// verify payToken

	if gIndex > uint64(len(r.groups)) {
		return ErrInput
	}

	gi := r.groups[gIndex]

	fm, err := getFsMgr(gi.fsAddr)
	if err != nil {
		return err
	}

	err = fm.CreateFs(r.local, index, payToken)
	if err != nil {
		return err
	}

	bi.roleType = roleUser
	bi.gIndex = gIndex
	bi.extra = blsKey

	return nil
}

func (r *roleMgr) CreateGroup(caller utils.Address, inds []uint64, level uint16, signature []byte) error {
	// verify utils.Address
	for _, index := range inds {
		// verify each keepr
		ki, _, err := r.GetInfoByIndex(caller, index)
		if err != nil {
			return err
		}

		// have been in some group
		if ki.isActive {
			return ErrPermission
		}

		if ki.roleType != roleKeeper {
			return ErrRoleType
		}
	}

	gIndex := len(r.groups)

	gi := &groupInfo{
		level:      level,
		keepers:    inds,
		providers:  make([]uint64, 0, 1),
		size:       big.NewInt(0),
		price:      big.NewInt(0),
		period:     10,
		totalCount: 0,
		lastTime:   uint64(time.Now().Unix()),
		tAcc:       make(map[uint32]*big.Int),
		count:      make(map[uint64]uint64),
	}

	r.groups = append(r.groups, gi)

	for _, index := range inds {
		ki, _, _ := r.GetInfoByIndex(caller, index)
		ki.isActive = true
		ki.gIndex = uint64(gIndex)
		gi.count[index] = 1
		gi.totalCount++
	}

	fs, err := NewFsMgr(r.local, r.local, uint64(gIndex))
	if err != nil {
		return err
	}

	gi.fsAddr = fs.GetContractAddress()

	if len(gi.keepers) >= int(level) {
		gi.isActive = true
	}

	return nil
}

func (r *roleMgr) AddKeeperToGroup(caller utils.Address, index, gIndex uint64, ksign, asign []byte) error {
	if len(r.groups) <= int(gIndex) {
		return ErrInput
	}

	// verify asign

	ki, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	if ki.roleType != roleKeeper {
		return ErrRoleType
	}

	// have been in some group
	if ki.isActive {
		return ErrPermission
	}

	// verify psign

	gi := r.groups[gIndex]

	gi.keepers = append(gi.keepers, index)
	ki.gIndex = gIndex
	ki.isActive = true

	gi.count[index] = 1
	gi.totalCount++

	if len(gi.keepers) >= int(gi.level) {
		gi.isActive = true
	}

	return nil
}

func (r *roleMgr) AddProviderToGroup(caller utils.Address, index, gIndex uint64, sign []byte) error {
	// verify sign by addr[index]
	if len(r.groups) <= int(gIndex) {
		return ErrInput
	}

	pi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	// have been in some group
	if pi.isActive {
		return ErrPermission
	}

	if pi.roleType != roleProvider {
		return ErrRoleType
	}

	gi := r.groups[gIndex]
	gi.providers = append(gi.providers, index)
	pi.gIndex = gIndex
	pi.isActive = true

	return nil
}

func (r *roleMgr) GetTokenByIndex(caller utils.Address, index uint32) (utils.Address, error) {
	if index >= uint32(len(r.tokens)) {
		return utils.Address{}, ErrInput
	}

	return r.tokens[index], nil
}

func (r *roleMgr) GetAddressByIndex(caller utils.Address, index uint64) (utils.Address, error) {
	if index >= uint64(len(r.addrs)) {
		return utils.Address{}, ErrInput
	}

	return r.addrs[index], nil
}

func (r *roleMgr) GetKeepersByIndex(caller utils.Address, index uint64) ([]uint64, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}

	g := r.groups[index]

	return g.keepers, nil
}

func (r *roleMgr) GetProvidersByIndex(caller utils.Address, index uint64) ([]uint64, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}

	g := r.groups[index]

	return g.providers, nil
}

func (r *roleMgr) GetGroupByIndex(caller utils.Address, index uint64) (uint64, error) {
	if index >= uint64(len(r.addrs)) {
		return 0, ErrInput
	}

	addr := r.addrs[index]
	ki, ok := r.info[addr]
	if ok {
		return ki.gIndex, nil
	}

	return 0, ErrRes
}

func (r *roleMgr) AddOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	ui, err := r.getInfo(user)
	if err != nil {
		return err
	}

	pi, err := r.getInfo(proIndex)
	if err != nil {
		return err
	}

	if pi.gIndex != ui.gIndex {
		return ErrInput
	}

	gi, err := r.getGroupInfo(ui.gIndex)
	if err != nil {
		return err
	}

	fm, err := getFsMgr(gi.fsAddr)
	if err != nil {
		return err
	}

	if tokenIndex >= uint32(len(ui.rewards)) {
		return ErrBalanceNotEnough
	}

	// update?
	err = r.update(user)
	if err != nil {
		return err
	}

	bal := ui.rewards[tokenIndex]
	pay := new(big.Int).Mul(sprice, new(big.Int).SetUint64(end-start))
	tax := new(big.Int).Mul(pay, big.NewInt(5))
	tax.Div(tax, big.NewInt(100))

	payAndTax := new(big.Int).Add(pay, tax)
	if bal.Cmp(payAndTax) < 0 {
		return ErrBalanceNotEnough
	}

	err = fm.AddOrder(caller, user, proIndex, start, end, size, nonce, tokenIndex, sprice, pay, tax, usign, psign, ksigns)
	if err != nil {
		return err
	}

	bal.Sub(bal, payAndTax)
	gi.size.Add(gi.size, new(big.Int).SetUint64(size))
	gi.price.Add(gi.price, sprice)

	// reduce from user
	ui.rewards[tokenIndex].Sub(ui.rewards[tokenIndex], payAndTax)

	ki, ok := r.info[caller]
	if ok {
		cnt, ok := gi.count[ki.index]
		if ok {
			cnt++
			gi.count[ki.index] = cnt
			gi.totalCount++
		}
	}

	return nil
}

func (r *roleMgr) SubOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	ui, err := r.getInfo(user)
	if err != nil {
		return err
	}

	if ui.roleType != roleUser {
		return ErrRoleType
	}

	pi, err := r.getInfo(proIndex)
	if err != nil {
		return err
	}

	if pi.roleType != roleProvider {
		return ErrRoleType
	}

	if pi.gIndex != ui.gIndex {
		return ErrInput
	}

	gi, err := r.getGroupInfo(ui.gIndex)
	if err != nil {
		return err
	}

	fm, err := getFsMgr(gi.fsAddr)
	if err != nil {
		return err
	}

	res, err := fm.SubOrder(caller, user, proIndex, start, end, size, nonce, tokenIndex, sprice, usign, psign, ksigns)
	if err != nil {
		return err
	}

	// update group profit
	ti := gi.tAcc[tokenIndex]
	ti.Add(ti, res)

	gi.size.Sub(gi.size, new(big.Int).SetUint64(size))
	gi.price.Sub(gi.price, sprice)

	ki, ok := r.info[caller]
	if ok {
		cnt, ok := gi.count[ki.index]
		if ok {
			cnt++
			gi.count[ki.index] = cnt
			gi.totalCount++
		}
	}

	return nil
}

func (r *roleMgr) ProWithdraw(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int) error {
	return nil
}

func (r *roleMgr) Withdraw(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int) error {
	err := r.update(index)
	if err != nil {
		return err
	}

	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	rw := new(big.Int).Set(bi.rewards[tokenIndex])

	if bi.roleType == roleKeeper {
		gi, err := r.getGroupInfo(bi.gIndex)
		if err != nil {
			return err
		}
		ntime := uint64(time.Now().Unix())
		if ntime-gi.lastTime > gi.period {
			if gi.totalCount <= 0 {
				return ErrRes
			}

			for tindex := range r.tokens {
				ti, ok := gi.tAcc[uint32(tindex)]
				if !ok {
					return ErrEmpty
				}

				per := new(big.Int).Div(ti, new(big.Int).SetUint64(gi.totalCount))
				for _, kindex := range gi.keepers {
					kc, ok := gi.count[kindex]
					if ok {
						pro := new(big.Int).Mul(per, new(big.Int).SetUint64(kc))
						bal := bi.rewards[tindex]

						bal.Add(bal, pro)
						ti.Sub(ti, pro)
					} else {
						gi.count[kindex] = 1
					}
				}
			}

			gi.lastTime = ntime
		}
	}

	if tokenIndex == 0 {
		if bi.roleType == roleKeeper {
			rw.Sub(rw, r.pledgeKeeper)
		} else if bi.roleType == roleProvider {
			rw.Sub(rw, r.pledgePro)
		}
	}

	if money.Cmp(zero) > 0 && money.Cmp(rw) < 0 {
		rw.Set(money)
	}

	tAddr := r.tokens[tokenIndex]
	err = sendBalance(tAddr, r.local, r.addrs[bi.index], rw)
	if err != nil {
		return err
	}

	ti := r.tInfo[tAddr]
	ti.lastRewardSupply.Sub(ti.lastRewardSupply, rw)
	bi.rewards[tokenIndex].Sub(bi.rewards[tokenIndex], rw)

	return nil
}
