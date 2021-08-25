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

type BaseInfo struct {
	isActive bool   // 是否激活
	isBanned bool   // 是否禁用
	RoleType uint8  // 0 account, 1, user, 2, provider, 3 keeper
	index    uint64 // 序列号
	gIndex   uint64 // 所属group
	extra    []byte // for offline
}

type tokenInfo struct {
	index    uint32
	isBanned bool
}

// GroupInfo has
type GroupInfo struct {
	isActive  bool
	isBanned  bool     // 组是否已被禁用
	isReady   bool     // 是否已在线下成组; 由签名触发
	level     uint16   // security level
	keepers   []uint64 // 里面有哪些Keeper
	providers []uint64 // 有哪些provider
	size      *big.Int // storeSize
	price     *big.Int
	FsAddr    utils.Address // fs contract addr
}

type mintInfo struct {
	ratio    uint16
	size     uint64
	duration uint64
}

var _ RoleMgr = (*roleMgr)(nil)

type roleMgr struct {
	local utils.Address // contract of this mgr
	admin utils.Address // owner

	pledge     utils.Address // pledge pool
	foundation utils.Address // foundation address

	addrs []utils.Address // all
	info  map[utils.Address]*BaseInfo

	// manage group
	groups []*GroupInfo

	// 代币信息,0为主代币的代币地址
	tokens []utils.Address
	tInfo  map[utils.Address]*tokenInfo

	pledgeKeeper *big.Int // pledgeMoney for keeper
	pledgePro    *big.Int // pledgeMoney for provider
	totalPledge  *big.Int

	mintLevel int
	mint      []*mintInfo
	lastMint  uint64
	start     uint64
	size      *big.Int
	price     *big.Int
	spaceTime *big.Int
	totalPaid *big.Int
	totalPay  *big.Int
}

// NewRoleMgr can be admin by mutiple signatures
func NewRoleMgr(caller, foundation, primaryToken utils.Address, kPledge, pPledge *big.Int) RoleMgr {
	// generate local utils.Address from
	local := utils.GetContractAddress(caller, []byte("RoleMgr"))

	mi := make([]*mintInfo, 4)
	mi[0] = &mintInfo{
		ratio:    100,
		size:     1,
		duration: 1,
	}

	mi[1] = &mintInfo{
		ratio:    120,
		size:     100 * 1024 * 1024 * 1024 * 1024, // 100TB
		duration: 100 * 24 * 60 * 60,              // 100 days
	}

	mi[2] = &mintInfo{
		ratio:    150,
		size:     1024 * 1024 * 1024 * 1024 * 1024, // 1PB
		duration: 100 * 24 * 60 * 60,               // 100 days
	}

	mi[3] = &mintInfo{
		ratio:    200,
		size:     10 * 1024 * 1024 * 1024 * 1024 * 1024, // 10PB
		duration: 100 * 24 * 60 * 60,                    // 100 days
	}

	rm := &roleMgr{
		admin:      caller,
		local:      local,
		foundation: foundation,

		addrs:  make([]utils.Address, 0, 128),
		info:   make(map[utils.Address]*BaseInfo),
		groups: make([]*GroupInfo, 0, 1),

		tokens: make([]utils.Address, 0, 1),
		tInfo:  make(map[utils.Address]*tokenInfo),

		pledgeKeeper: kPledge,
		pledgePro:    pPledge,
		totalPledge:  big.NewInt(0),

		mint:      mi,
		start:     GetTime(),
		lastMint:  GetTime(),
		mintLevel: 0,
		size:      big.NewInt(0),
		price:     big.NewInt(0),
		totalPaid: big.NewInt(0),
		totalPay:  big.NewInt(0),
		spaceTime: big.NewInt(0),
	}

	ti := &tokenInfo{
		index: 0,
	}

	rm.tokens = append(rm.tokens, primaryToken)
	rm.tInfo[primaryToken] = ti

	bi := &BaseInfo{
		RoleType: 0,
		index:    uint64(len(rm.addrs)),
	}

	rm.addrs = append(rm.addrs, foundation)
	rm.info[foundation] = bi

	pp := NewPledgeMgr(rm.local, primaryToken)
	rm.pledge = pp.GetContractAddress()

	globalMap[local] = rm
	return rm
}

func (r *roleMgr) GetContractAddress() utils.Address {
	return r.local
}

func (r *roleMgr) GetOwnerAddress() utils.Address {
	return r.admin
}

func (r *roleMgr) GetFoundation(caller utils.Address) utils.Address {
	return r.foundation
}

func (r *roleMgr) GetPledgeAddress(caller utils.Address) utils.Address {
	return r.pledge
}

func (r *roleMgr) GetAllTokens(caller utils.Address) []utils.Address {
	return r.tokens
}

func (r *roleMgr) GetAllAddrs(caller utils.Address) []utils.Address {
	return r.addrs
}

func (r *roleMgr) GetAllGroups(caller utils.Address) []*GroupInfo {
	return r.groups
}

// register related

func (r *roleMgr) GetIndex(caller, addr utils.Address) (uint64, error) {
	bi, ok := r.info[addr]
	if ok {
		return bi.index, nil
	}

	return 0, ErrNoSuchAddr
}

func (r *roleMgr) GetTokenAddress(caller utils.Address, index uint32) (utils.Address, error) {
	if index >= uint32(len(r.tokens)) {
		return utils.Address{}, ErrInput
	}

	return r.tokens[index], nil
}

func (r *roleMgr) GetTokenIndex(caller, addr utils.Address) (uint32, error) {
	ti, ok := r.tInfo[addr]
	if ok {
		return ti.index, nil
	}

	return 0, ErrNoSuchAddr
}

func (r *roleMgr) GetInfo(caller utils.Address, index uint64) (*BaseInfo, utils.Address, error) {
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

func (r *roleMgr) getInfo(index uint64) (*BaseInfo, error) {
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

func (r *roleMgr) RegisterToken(caller, taddr utils.Address, sign []byte) error {
	// verify sign
	// chek existence
	_, ok := r.tInfo[taddr]
	if ok {
		// exist
		return ErrExist
	}

	pp, err := getPledgePool(r.pledge)
	if err != nil {
		return err
	}

	ti := &tokenInfo{
		index: uint32(len(r.tokens)),
	}

	r.tokens = append(r.tokens, taddr)
	r.tInfo[taddr] = ti

	return pp.AddToken(r.local, taddr, ti.index)
}

func (r *roleMgr) Register(caller, addr utils.Address, signature []byte) error {
	// verify sign
	// chek existence
	_, ok := r.info[addr]
	if ok {
		return ErrExist
	}

	bi := &BaseInfo{
		RoleType: 0,
		index:    uint64(len(r.addrs)),
	}

	r.addrs = append(r.addrs, addr)
	r.info[addr] = bi

	return nil
}

func (r *roleMgr) RegisterKeeper(caller utils.Address, index uint64, blsKey, signature []byte) error {
	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	// registered
	if bi.RoleType != 0 {
		return ErrRoleType
	}

	pp, err := getPledgePool(r.pledge)
	if err != nil {
		return err
	}

	vals := pp.GetBalance(r.local, index)
	if len(vals) < 1 {
		return ErrBalanceNotEnough
	}

	if vals[0].Cmp(r.pledgeKeeper) < 0 {
		return ErrBalanceNotEnough
	}

	bi.RoleType = RoleKeeper
	bi.extra = blsKey

	return nil
}

func (r *roleMgr) RegisterProvider(caller utils.Address, index uint64, signature []byte) error {
	// verify sign
	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	// registered
	if bi.RoleType != 0 {
		return ErrRoleType
	}

	pp, err := getPledgePool(r.pledge)
	if err != nil {
		return err
	}

	vals := pp.GetBalance(r.local, index)
	if len(vals) < 1 {
		return ErrBalanceNotEnough
	}

	if vals[0].Cmp(r.pledgePro) < 0 {
		return ErrBalanceNotEnough
	}

	bi.RoleType = RoleProvider

	return nil
}

func (r *roleMgr) RegisterUser(caller utils.Address, index, gIndex uint64, payToken uint32, blsKey, sign []byte) error {
	// verify sign

	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	// registered
	if bi.RoleType != 0 {
		return ErrRoleType
	}

	// verify payToken
	if gIndex > uint64(len(r.groups)) {
		return ErrInput
	}

	gi := r.groups[gIndex]
	if !gi.isActive || gi.isBanned {
		return ErrPermission
	}

	fm, err := getFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	err = fm.CreateFs(r.local, index, payToken)
	if err != nil {
		return err
	}

	bi.RoleType = RoleUser
	bi.gIndex = gIndex
	bi.extra = blsKey

	return nil
}

// group related

func (r *roleMgr) GetKeepersInGroup(caller utils.Address, index uint64) ([]uint64, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}

	g := r.groups[index]

	return g.keepers, nil
}

func (r *roleMgr) GetProvidersInGroup(caller utils.Address, index uint64) ([]uint64, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}

	g := r.groups[index]

	return g.providers, nil
}

func (r *roleMgr) GetGroupInfo(caller utils.Address, index uint64) (*GroupInfo, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}
	return r.groups[index], nil
}

func (r *roleMgr) getGroupInfo(index uint64) (*GroupInfo, error) {
	if index >= uint64(len(r.groups)) {
		return nil, ErrInput
	}

	gi := r.groups[index]

	if !gi.isActive || gi.isBanned {
		return nil, ErrPermission
	}

	return gi, nil
}

// CreateGroup
func (r *roleMgr) CreateGroup(caller utils.Address, inds []uint64, level uint16, signature []byte) error {
	// verify utils.Address
	for _, index := range inds {
		// verify each keepr
		ki, err := r.getInfo(index)
		if err != nil {
			return err
		}

		// have been in some group
		if ki.isActive {
			return ErrPermission
		}

		if ki.RoleType != RoleKeeper {
			return ErrRoleType
		}
	}

	gIndex := len(r.groups)

	gi := &GroupInfo{
		level:     level,
		keepers:   inds,
		providers: make([]uint64, 0, 1),
		size:      big.NewInt(0),
		price:     big.NewInt(0),
	}

	r.groups = append(r.groups, gi)

	for _, index := range inds {
		ki, _ := r.getInfo(index)
		ki.isActive = true
		ki.gIndex = uint64(gIndex)
	}

	fs, err := NewFsMgr(r.local, 0, uint64(gIndex))
	if err != nil {
		return err
	}

	gi.FsAddr = fs.GetContractAddress()

	if len(gi.keepers) >= int(level) {
		gi.isActive = true
	}

	return nil
}

func (r *roleMgr) AddKeeperToGroup(caller utils.Address, index, gIndex uint64, ksign, asign []byte) error {
	// verify asign
	if len(r.groups) <= int(gIndex) {
		return ErrInput
	}

	gi := r.groups[gIndex]
	if gi.isBanned {
		return ErrPermission
	}

	ki, err := r.getInfo(index)
	if err != nil {
		return err
	}

	if ki.RoleType != RoleKeeper {
		return ErrRoleType
	}

	// have been in some group
	if ki.isActive {
		return ErrPermission
	}

	fsMgr, err := getFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	err = fsMgr.AddKeeper(r.local, index)
	if err != nil {
		return err
	}

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

	gi := r.groups[gIndex]
	if gi.isBanned {
		return ErrPermission
	}

	pi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	// have been in some group
	if pi.isActive {
		return ErrPermission
	}

	if pi.RoleType != RoleProvider {
		return ErrRoleType
	}

	gi.providers = append(gi.providers, index)
	pi.gIndex = gIndex
	pi.isActive = true

	return nil
}

// balance related

func (r *roleMgr) GetPledge(caller utils.Address) (*big.Int, *big.Int, []*big.Int) {
	pp, err := getPledgePool(r.pledge)
	if err != nil {
		return r.pledgeKeeper, r.pledgePro, nil
	}

	return r.pledgeKeeper, r.pledgePro, pp.GetPledge(caller)
}

func (r *roleMgr) GetBalance(caller utils.Address, index uint64) ([]*big.Int, error) {
	pp, err := getPledgePool(r.pledge)
	if err != nil {
		return nil, err
	}

	return pp.GetBalance(r.local, index), nil
}

func (r *roleMgr) SetPledgeMoney(caller utils.Address, kPledge, pPledge *big.Int, signature []byte) error {
	// verify sign(hash(kPLedge,pPledge))

	r.pledgeKeeper = kPledge
	r.pledgePro = pPledge
	return nil
}

// 质押，非流动性
func (r *roleMgr) Pledge(caller utils.Address, index uint64, money *big.Int, sign []byte) error {
	// verify sign

	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	if bi.isBanned {
		return ErrPermission
	}

	pp, err := getPledgePool(r.pledge)
	if err != nil {
		return err
	}

	return pp.Pledge(r.local, index, money)
}

func (r *roleMgr) Withdraw(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int, sign []byte) error {
	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrInput
	}

	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	if bi.isBanned {
		return ErrPermission
	}

	pp, err := getPledgePool(r.pledge)
	if err != nil {
		return err
	}

	lock := new(big.Int)
	if bi.RoleType == RoleKeeper {
		lock.Set(r.pledgeKeeper)
	} else if bi.RoleType == RoleProvider {
		lock.Set(r.pledgePro)
	}

	return pp.Withdraw(r.local, index, tokenIndex, money, lock)
}

// order ops
func (r *roleMgr) AddOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	log.Info("AddOrder")
	// verify ksigns
	// verify usign
	// verify psign
	ui, err := r.getInfo(user)
	if err != nil {
		return err
	}

	pi, err := r.getInfo(proIndex)
	if err != nil {
		return err
	}

	kindex := uint64(0)
	ki, ok := r.info[caller]
	if ok {
		kindex = ki.index
	}

	if pi.gIndex != ui.gIndex {
		return ErrInput
	}

	gi, err := r.getGroupInfo(ui.gIndex)
	if err != nil {
		return err
	}

	fm, err := getFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrBalanceNotEnough
	}

	err = fm.AddOrder(r.local, kindex, user, proIndex, start, end, size, nonce, tokenIndex, sprice, usign, psign, ksigns)
	if err != nil {
		return err
	}

	// 增发
	if tokenIndex == 0 {
		gi.size.Add(gi.size, new(big.Int).SetUint64(size))
		gi.price.Add(gi.price, sprice)

		ntime := GetTime()
		dur := new(big.Int).SetUint64(ntime - r.lastMint)

		paid := new(big.Int).Mul(r.price, dur)

		reward := new(big.Int).Sub(r.totalPay, r.totalPaid)

		if r.price.Cmp(zero) > 0 {
			length := new(big.Int).Div(reward, r.price) // length
			reward.Div(reward, length)
			reward.Mul(reward, dur)
		}

		log.Info("AddOrder: send to pledge: ", reward)

		for i := r.mintLevel + 1; i < len(r.mint); i++ {
			esize := new(big.Int).SetUint64(r.mint[i].size)
			if esize.Cmp(r.size) < 0 {
				esize.Set(r.size)
			}

			dur := new(big.Int).SetUint64(r.mint[i].duration)

			if r.spaceTime.Div(r.spaceTime, esize).Cmp(dur) >= 0 {
				r.mintLevel = i
			} else {
				break
			}
		}
		log.Info("AddOrder: send to pledge: ", reward)

		st := new(big.Int).Mul(r.size, new(big.Int).SetUint64(end-start))
		r.spaceTime.Add(r.spaceTime, st)

		r.size.Add(r.size, new(big.Int).SetUint64(size))
		r.price.Add(r.price, sprice)
		pay := new(big.Int).Mul(sprice, new(big.Int).SetUint64(end-start))
		r.totalPay.Add(r.totalPay, pay)

		reward.Mul(reward, new(big.Int).SetUint64(uint64(r.mint[r.mintLevel].ratio)))
		reward.Div(reward, new(big.Int).SetUint64(100))

		err = sendBalance(r.tokens[0], r.local, r.pledge, reward)
		if err != nil {
			return err
		}
		log.Info("AddOrder: send to pledge: ", reward)
		r.totalPaid.Add(r.totalPaid, paid)
		log.Info("AddOrder: totalPay: ", r.totalPay, r.totalPaid, r.lastMint, ntime)

		r.lastMint = ntime
	}

	return nil
}

func (r *roleMgr) SubOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	ui, err := r.getInfo(user)
	if err != nil {
		return err
	}

	if ui.RoleType != RoleUser {
		return ErrRoleType
	}

	pi, err := r.getInfo(proIndex)
	if err != nil {
		return err
	}

	if pi.RoleType != RoleProvider {
		return ErrRoleType
	}

	if pi.gIndex != ui.gIndex {
		return ErrInput
	}

	gi, err := r.getGroupInfo(ui.gIndex)
	if err != nil {
		return err
	}

	kindex := uint64(0)
	ki, ok := r.info[caller]
	if ok {
		kindex = ki.index
	}

	fm, err := getFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	err = fm.SubOrder(r.local, kindex, user, proIndex, start, end, size, nonce, tokenIndex, sprice, usign, psign, ksigns)
	if err != nil {
		return err
	}

	gi.size.Sub(gi.size, new(big.Int).SetUint64(size))
	gi.price.Sub(gi.price, sprice)

	if tokenIndex == 0 {
		gi.size.Sub(gi.size, new(big.Int).SetUint64(size))
		gi.price.Sub(gi.price, sprice)

		r.price.Sub(r.price, sprice)
		r.size.Sub(r.size, new(big.Int).SetUint64(size))
	}

	return nil
}
