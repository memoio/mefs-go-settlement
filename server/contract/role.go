package contract

import (
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

const (
	RoleUser     uint8 = 1
	RoleProvider uint8 = 2
	RoleKeeper   uint8 = 3
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

// GroupInfo has
type GroupInfo struct {
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
	groups []*GroupInfo

	// 代币信息,0为主代币的代币地址
	tokens []utils.Address
	tInfo  map[utils.Address]*TokenInfo

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
		groups: make([]*GroupInfo, 0, 1),

		tokens: make([]utils.Address, 0, 1),
		tInfo:  make(map[utils.Address]*TokenInfo),

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

func (r *roleMgr) RegisterToken(addr utils.Address, sign []byte) error {
	// verify sign
	_, ok := r.tInfo[addr]
	if ok {
		// exist
		return ErrRes
	}

	bal := getBalance(addr, r.local)
	ti := &TokenInfo{
		index:            uint32(len(r.tokens)),
		rewardAccum:      big.NewInt(0),
		lastRewardSupply: bal,
	}

	r.tokens = append(r.tokens, addr)
	r.tInfo[addr] = ti

	return nil
}

func (r *roleMgr) Register(addr utils.Address, signature []byte) error {
	// verify sign
	_, ok := r.info[addr]
	if ok {
		return ErrRes
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
			return ErrRes
		}

		bi.rewardAccum[i] = new(big.Int).Set(ti.rewardAccum)
		bi.rewards[i] = new(big.Int)
	}

	r.addrs = append(r.addrs, addr)
	r.info[addr] = bi

	return nil
}

func (r *roleMgr) WithdrawToken(index uint64, tokenIndex uint32, money *big.Int) error {

	bi, err := r.GetInfoByIndex(index)
	if err != nil {
		return err
	}

	tokenAddr, err := r.GetTokenByIndex(tokenIndex)
	if err != nil {
		return err
	}

	rw := bi.rewards[tokenIndex-1]

	res := big.NewInt(0)

	if money.Cmp(rw) > 0 {
		res.Set(rw)
	} else {
		res.Set(money)
	}

	err = sendBalance(tokenAddr, r.local, r.addrs[bi.index], res)
	if err != nil {
		return err
	}
	bi.rewards[tokenIndex-1] = big.NewInt(0)

	return nil
}

func (r *roleMgr) Withdraw(index uint64, money *big.Int) error {

	bi, err := r.GetInfoByIndex(index)
	if err != nil {
		return err
	}

	rw := new(big.Int).Set(bi.amount)

	if bi.roleType == RoleKeeper {
		rw.Sub(rw, r.pledgeKeeper)
	} else if bi.roleType == RoleProvider {
		rw.Sub(rw, r.pledgePro)
	}

	if money.Cmp(rw) < 0 {
		rw.Set(money)
	}

	err = sendBalance(r.tokens[0], r.local, r.addrs[bi.index], rw)
	if err != nil {
		return err
	}
	bi.amount.Sub(bi.amount, rw)

	return nil
}

func (r *roleMgr) GetInfoByIndex(index uint64) (*baseInfo, error) {
	if int(index) >= len(r.addrs) {
		return nil, nil
	}

	addr := r.addrs[index]
	ki, ok := r.info[addr]
	if !ok {
		return nil, nil
	}

	if ki.isBanned {
		return nil, nil
	}

	return ki, nil
}

func (r *roleMgr) GetInfo(addr utils.Address) (*baseInfo, error) {
	bi, ok := r.info[addr]
	if !ok {
		return nil, ErrRes
	}

	return bi, nil
}

func (r *roleMgr) Pledge(index uint64, money *big.Int, sign []byte) error {
	// verify sign

	// chekc money > 0
	if money.Cmp(zero) < 0 {
		return ErrRes
	}

	bi, err := r.GetInfoByIndex(index)
	if err != nil {
		return err
	}

	// 更新token acc，结算奖励
	for i, taddr := range r.tokens[1:] {
		ti, ok := r.tInfo[taddr]
		if !ok {
			return nil
		}

		bal := getBalance(taddr, r.local)

		ti.update(bal, r.totalPledge)

		if len(bi.rewardAccum) <= i {
			bi.rewardAccum = append(bi.rewardAccum, new(big.Int))
			bi.rewards = append(bi.rewards, new(big.Int))
		}

		res := new(big.Int).Set(ti.rewardAccum)
		res.Sub(res, bi.rewards[i])
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

func (r *roleMgr) SetPledgeMoney(caller utils.Address, kPledge, pPledge *big.Int, signature []byte) error {
	// verify sign(hash(kPLedge,pPledge))

	r.pledgeKeeper = kPledge
	r.pledgePro = pPledge
	return nil
}

func (r *roleMgr) RegisterKeeper(index uint64, blsKey, signature []byte) error {
	bi, err := r.GetInfoByIndex(index)
	if err != nil {
		return err
	}

	// registered
	if bi.roleType != 0 {
		return ErrRes
	}

	if bi.amount.Cmp(r.pledgeKeeper) < 0 {
		return ErrRes
	}

	bi.roleType = RoleKeeper
	bi.extra = blsKey

	return nil
}

func (r *roleMgr) RegisterProvider(index uint64, signature []byte) error {
	// verify sign
	bi, err := r.GetInfoByIndex(index)
	if err != nil {
		return err
	}

	// registered
	if bi.roleType != 0 {
		return ErrRes
	}

	if bi.amount.Cmp(r.pledgeKeeper) < 0 {
		return ErrRes
	}

	bi.roleType = RoleProvider

	return nil
}

func (r *roleMgr) RegisterUser(index uint64, blsKey, signature []byte) error {
	// verify sign(hash(blsKey))
	bi, err := r.GetInfoByIndex(index)
	if err != nil {
		return err
	}

	// registered
	if bi.roleType != 0 {
		return ErrRes
	}

	bi.roleType = RoleUser
	bi.extra = blsKey

	return nil
}

func (r *roleMgr) CreateGroup(inds []uint64, level uint16, signature []byte) error {
	// verify utils.Address
	for _, index := range inds {
		// verify each keepr
		ki, err := r.GetInfoByIndex(index)
		if err != nil {
			return err
		}

		// have been in some group
		if ki.isActive {
			return ErrRes
		}

		if ki.roleType != RoleKeeper {
			return ErrRes
		}
	}

	gi := &GroupInfo{
		level:   level,
		keepers: inds,
	}

	gIndex := len(r.groups)

	r.groups = append(r.groups, gi)

	for _, index := range inds {
		ki, _ := r.GetInfoByIndex(index)
		ki.isActive = true
		ki.gIndex = uint64(gIndex)
	}

	if len(gi.keepers) >= int(level) {
		gi.isActive = true
	}

	return nil
}

func (r *roleMgr) SetFsAddrForGroup(gIndex uint64, fAddr utils.Address, asign []byte) error {
	if len(r.groups) <= int(gIndex) {
		return ErrRes
	}

	// verify fs.gindex

	//verify r.groups[gIndex].fsAddr is not set

	r.groups[gIndex].fsAddr = fAddr

	return nil
}

func (r *roleMgr) AddKeeperToGroup(index, gIndex uint64, ksign, asign []byte) error {
	if len(r.groups) <= int(gIndex) {
		return ErrRes
	}

	// verify asign

	ki, err := r.GetInfoByIndex(index)
	if err != nil {
		return err
	}

	if ki.roleType != RoleKeeper {
		return ErrRes
	}

	// have been in some group
	if ki.isActive {
		return ErrRes
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

func (r *roleMgr) AddProviderToGroup(index, gIndex uint64, sign []byte) error {
	// verify sign by addr[index]
	if len(r.groups) <= int(gIndex) {
		return nil
	}

	pi, err := r.GetInfoByIndex(index)
	if err != nil {
		return err
	}

	// have been in some group
	if pi.isActive {
		return ErrRes
	}

	if pi.roleType != RoleProvider {
		return ErrRes
	}

	gi := r.groups[gIndex]
	gi.providers = append(gi.providers, index)
	pi.gIndex = gIndex
	pi.isActive = true

	return nil
}

func (r *roleMgr) GetTokenByIndex(index uint32) (utils.Address, error) {
	if index >= uint32(len(r.tokens)) {
		return utils.Address{}, ErrRes
	}

	return r.tokens[index], nil
}

func (r *roleMgr) GetAddressByIndex(index uint64) (utils.Address, error) {
	if index >= uint64(len(r.addrs)) {
		return utils.Address{}, ErrRes
	}

	return r.addrs[index], nil
}

func (r *roleMgr) GetKeepersByIndex(index uint64) ([]uint64, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrRes
	}

	g := r.groups[index]
	if !g.isActive || !g.isBanned {
		return nil, ErrRes
	}

	return g.keepers, nil
}

func (r *roleMgr) GetProvidersByIndex(index uint64) ([]uint64, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrRes
	}

	g := r.groups[index]
	if !g.isActive || !g.isBanned {
		return nil, ErrRes
	}

	return g.providers, nil
}

func (r *roleMgr) GetGroupByIndex(index uint64) (uint64, error) {
	if index >= uint64(len(r.addrs)) {
		return 0, ErrRes
	}

	addr := r.addrs[index]
	ki, ok := r.info[addr]
	if ok {
		return ki.gIndex, nil
	}

	return 0, ErrRes
}

func getRoleMgrByAddress(rAddr utils.Address) (RoleMgr, error) {
	ri, ok := globalMap[rAddr]
	if ok {
		r, ok := ri.(RoleMgr)
		if ok {
			return r, nil
		}
	}

	return nil, ErrRes
}

func getAddressByIndex(rAddr utils.Address, index uint64) (utils.Address, error) {
	ri, ok := globalMap[rAddr]
	if ok {
		r, ok := ri.(RoleMgr)
		if ok {
			return r.GetAddressByIndex(index)
		}
	}

	return utils.Address{}, ErrRes
}

func getTokenByIndex(rAddr utils.Address, index uint32) (utils.Address, error) {
	ri, ok := globalMap[rAddr]
	if ok {
		r, ok := ri.(RoleMgr)
		if ok {
			return r.GetTokenByIndex(index)
		}
	}

	return utils.Address{}, ErrRes
}

func getBalanceByIndex(rAddr utils.Address, query uint64) *big.Int {
	res := big.NewInt(0)

	return res
}

func getTokenBalanceByIndex(rAddr utils.Address, tokenIndex uint32, query uint64) *big.Int {
	res := big.NewInt(0)

	ri, ok := globalMap[rAddr]
	if ok {
		r, ok := ri.(RoleMgr)
		if ok {
			_, err := r.GetTokenByIndex(tokenIndex)
			if err != nil {
				return res
			}

			pi, err := r.GetInfoByIndex(query)
			if err != nil {
				return res
			}

			res.Set(pi.rewards[tokenIndex-1])
		}
	}

	return res
}
