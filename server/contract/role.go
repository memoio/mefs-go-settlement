package contract

import (
	"math/big"

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

	amount      *big.Int   // 映射代币的余额
	rewardAccum []*big.Int // 质押开始时的accumulator，索引和代币对应
	rewards     []*big.Int // 已结算的奖励，索引和代币对应
}

type tokenInfo struct {
	index            uint32
	rewardAccum      *big.Int // 本代币的accumulator
	lastRewardSupply *big.Int // 上一次变更时的本代币奖励总量
}

func (t tokenInfo) update(amount, totalPledge *big.Int) {
	tv := new(big.Int).Set(amount)
	tv.Sub(tv, t.lastRewardSupply)
	if tv.Cmp(zero) > 0 && totalPledge.Cmp(zero) > 0 {
		tv.Div(tv, totalPledge)
		t.rewardAccum.Add(t.rewardAccum, tv)
	}

	t.lastRewardSupply = amount // update to lastest
}

// groupInfo has
type groupInfo struct {
	isActive  bool
	isBanned  bool          // 组是否已被禁用
	level     uint16        // security level
	keepers   []uint64      // 里面有哪些Keeper
	providers []uint64      // 有哪些provider
	fsAddr    utils.Address // fs contract addr
}

var _ RoleMgr = (*roleMgr)(nil)

type roleMgr struct {
	local utils.Address // contract of this mgr
	admin utils.Address // owner

	addrs []utils.Address // all
	info  map[utils.Address]*baseInfo

	// manage group
	groups []*groupInfo

	// 代币信息,0为主代币的代币地址
	tokens []utils.Address
	tInfo  map[utils.Address]*tokenInfo

	pledgeKeeper *big.Int // pledgeMoney for keeper
	pledgePro    *big.Int // pledgeMoney for provider
	totalPledge  *big.Int
}

// NewRoleMgr can be admin by mutiple signatures
func NewRoleMgr(caller, primaryToken utils.Address, kPledge, pPledge *big.Int) RoleMgr {
	// generate local utils.Address from
	local := utils.GetContractAddress(caller, []byte("RoleMgr"))

	rm := &roleMgr{
		admin: caller,
		local: local,

		addrs:  make([]utils.Address, 0, 128),
		info:   make(map[utils.Address]*baseInfo),
		groups: make([]*groupInfo, 0, 1),

		tokens: make([]utils.Address, 0, 1),
		tInfo:  make(map[utils.Address]*tokenInfo),

		pledgeKeeper: kPledge,
		pledgePro:    pPledge,
		totalPledge:  big.NewInt(0),
	}
	rm.tokens = append(rm.tokens, primaryToken)
	globalMap[local] = rm
	return rm
}

func (r *roleMgr) GetContractAddress() utils.Address {
	return r.local
}

func (r *roleMgr) GetOwnerAddress() utils.Address {
	return r.admin
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
func (r *roleMgr) GetPledge(caller utils.Address) (*big.Int, *big.Int, *big.Int) {
	return r.totalPledge, r.pledgeKeeper, r.pledgePro
}

func (r *roleMgr) GetGroupInfoByIndex(caller utils.Address, index uint64) (*groupInfo, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}
	return r.groups[index], nil
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
		amount:      big.NewInt(0),
		rewardAccum: make([]*big.Int, len(r.tokens)-1),
		rewards:     make([]*big.Int, len(r.tokens)-1),
	}

	for i, tAddr := range r.tokens[1:] {
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

	res := make([]*big.Int, len(bi.rewards)+1)
	res[0] = new(big.Int).Set(bi.amount)
	for i := 0; i < len(bi.rewards); i++ {
		taddr := r.tokens[i+1]
		ti, ok := r.tInfo[taddr]
		if !ok {
			return nil, ErrEmpty
		}

		val := new(big.Int).Set(ti.rewardAccum)
		bal := getBalance(taddr, r.local)
		bal.Sub(bal, ti.lastRewardSupply)
		if bal.Cmp(zero) > 0 && r.totalPledge.Cmp(zero) > 0 {
			bal.Div(bal, r.totalPledge)
			val.Add(val, bal)
		}
		val.Sub(val, bi.rewardAccum[i])
		val.Mul(val, bi.amount)
		val.Add(val, bi.rewards[i])
		res[i+1] = val
	}

	return res, nil
}

func (r *roleMgr) Pledge(caller utils.Address, index uint64, money *big.Int, sign []byte) error {
	// verify sign

	// chekc money > 0
	if money.Cmp(zero) < 0 {
		return ErrValue
	}

	bi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	// 更新token acc，结算奖励
	for i, taddr := range r.tokens[1:] {
		ti, ok := r.tInfo[taddr]
		if !ok {
			return ErrEmpty
		}

		bal := getBalance(taddr, r.local)

		ti.update(bal, r.totalPledge)

		if len(bi.rewardAccum) <= i {
			bi.rewardAccum = append(bi.rewardAccum, new(big.Int))
			bi.rewards = append(bi.rewards, new(big.Int))
		}

		res := new(big.Int).Set(ti.rewardAccum)
		res.Sub(res, bi.rewardAccum[i])
		res.Mul(res, bi.amount)
		bi.rewards[i].Add(bi.rewards[i], res) // 添加奖励

		bi.rewardAccum[i] = new(big.Int).Set(ti.rewardAccum) // 更新acc
	}

	addr := r.addrs[bi.index]
	err = sendBalance(r.tokens[0], addr, r.local, money)
	if err != nil {
		return err
	}
	bi.amount.Add(bi.amount, money)
	r.totalPledge.Add(r.totalPledge, money)

	return nil
}

func (r *roleMgr) Withdraw(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int) error {

	if tokenIndex > 0 {
		return r.withdrawToken(caller, index, tokenIndex, money)
	}

	bi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	rw := new(big.Int).Set(bi.amount)

	if bi.roleType == roleKeeper {
		rw.Sub(rw, r.pledgeKeeper)
	} else if bi.roleType == roleProvider {
		rw.Sub(rw, r.pledgePro)
	}

	if money.Cmp(zero) > 0 && money.Cmp(rw) < 0 {
		rw.Set(money)
	}

	// update token
	for i, taddr := range r.tokens[1:] {
		ti, ok := r.tInfo[taddr]
		if !ok {
			return ErrEmpty
		}

		bal := getBalance(taddr, r.local)

		ti.update(bal, r.totalPledge)

		if len(bi.rewardAccum) <= i {
			bi.rewardAccum = append(bi.rewardAccum, new(big.Int))
			bi.rewards = append(bi.rewards, new(big.Int))
		}

		res := new(big.Int).Set(ti.rewardAccum)
		res.Sub(res, bi.rewardAccum[i])
		res.Mul(res, bi.amount)
		bi.rewards[i].Add(bi.rewards[i], res)                // 添加奖励
		bi.rewardAccum[i] = new(big.Int).Set(ti.rewardAccum) // 更新acc
	}

	err = sendBalance(r.tokens[0], r.local, r.addrs[bi.index], rw)
	if err != nil {
		return err
	}
	bi.amount.Sub(bi.amount, rw)
	r.totalPledge.Sub(r.totalPledge, rw)

	return nil
}

func (r *roleMgr) withdrawToken(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int) error {
	bi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	taddr, err := r.GetTokenByIndex(caller, tokenIndex)
	if err != nil {
		return err
	}

	ti, ok := r.tInfo[taddr]
	if !ok {
		return nil
	}

	bal := getBalance(taddr, r.local)

	ti.update(bal, r.totalPledge)
	if uint32(len(bi.rewardAccum)) <= tokenIndex-1 {
		bi.rewardAccum = append(bi.rewardAccum, new(big.Int))
		bi.rewards = append(bi.rewards, new(big.Int))
	}

	res := new(big.Int).Set(ti.rewardAccum)
	res.Sub(res, bi.rewardAccum[tokenIndex-1])
	res.Mul(res, bi.amount)
	bi.rewards[tokenIndex-1].Add(bi.rewards[tokenIndex-1], res)     // 添加奖励
	bi.rewardAccum[tokenIndex-1] = new(big.Int).Set(ti.rewardAccum) // 更新acc

	val := new(big.Int).Set(bi.rewards[tokenIndex-1])

	if money.Cmp(val) < 0 {
		res.Set(money)
	}

	err = sendBalance(taddr, r.local, r.addrs[bi.index], val)
	if err != nil {
		return err
	}
	bi.rewards[tokenIndex-1].Sub(bi.rewards[tokenIndex-1], val)
	ti.lastRewardSupply = getBalance(taddr, r.local)

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

	if bi.amount.Cmp(r.pledgeKeeper) < 0 {
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

	if bi.amount.Cmp(r.pledgePro) < 0 {
		return ErrBalanceNotEnough
	}

	bi.roleType = roleProvider

	return nil
}

func (r *roleMgr) RegisterUser(caller utils.Address, index, gIndex uint64, blsKey []byte) error {
	// verify sender is contract
	if gIndex > uint64(len(r.groups)) {
		return ErrInput
	}
	if r.groups[gIndex].fsAddr != caller {
		return ErrInput
	}

	bi, _, err := r.GetInfoByIndex(caller, index)
	if err != nil {
		return err
	}

	// registered
	if bi.roleType != 0 {
		return ErrRoleType
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

	gi := &groupInfo{
		level:   level,
		keepers: inds,
	}

	gIndex := len(r.groups)

	r.groups = append(r.groups, gi)

	for _, index := range inds {
		ki, _, _ := r.GetInfoByIndex(caller, index)
		ki.isActive = true
		ki.gIndex = uint64(gIndex)
	}

	if len(gi.keepers) >= int(level) {
		gi.isActive = true
	}

	return nil
}

func (r *roleMgr) SetFsAddrForGroup(caller utils.Address, fAddr utils.Address, asign []byte) error {
	fm, err := getFsMgr(fAddr)
	if err != nil {
		return err
	}

	gIndex := fm.GetInfo(r.local)

	if gIndex >= uint64(len(r.groups)) {
		return ErrInput
	}

	gi := r.groups[gIndex]
	//verify r.groups[gIndex].fsAddr is not set
	if gi.fsAddr != utils.NilAddress {
		return ErrExist
	}

	// need verify faddr.gindex?
	gi.fsAddr = fAddr

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

func getAddressByIndex(rAddr, caller utils.Address, index uint64) (utils.Address, error) {
	ri, ok := globalMap[rAddr]
	if ok {
		r, ok := ri.(RoleMgr)
		if ok {
			return r.GetAddressByIndex(caller, index)
		}
	}

	return utils.NilAddress, ErrRes
}

func getTokenByIndex(rAddr, caller utils.Address, index uint32) (utils.Address, error) {
	ri, ok := globalMap[rAddr]
	if ok {
		r, ok := ri.(RoleMgr)
		if ok {
			return r.GetTokenByIndex(caller, index)
		}
	}

	return utils.NilAddress, ErrRes
}
