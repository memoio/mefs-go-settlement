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
	IsActive bool   // 是否激活
	IsBanned bool   // 是否禁用
	RoleType uint8  // 0 account, 1, user, 2, provider, 3 keeper
	Index    uint64 // 序列号
	GIndex   uint64 // 所属group
	Extra    []byte // for offline
}

type tokenInfo struct {
	IsBanned bool
	Index    uint32
}

// GroupInfo has
type GroupInfo struct {
	IsActive  bool     // enough keepers?
	IsBanned  bool     // 组是否已被禁用
	IsReady   bool     // 是否已在线下成组; 由签名触发
	Level     uint16   // security level
	Keepers   []uint64 // 里面有哪些Keeper
	Providers []uint64 // 有哪些provider
	Size      *big.Int // storeSize
	Price     *big.Int
	FsAddr    utils.Address // fs contract addr
}

type MintInfo struct {
	Ratio    uint16
	Size     uint64
	Duration uint64
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
	mint      []*MintInfo
	lastMint  uint64
	start     uint64
	size      *big.Int
	price     *big.Int
	spaceTime *big.Int
	totalPaid *big.Int
	totalPay  *big.Int

	subPMap map[uint64]*big.Int
	subSMap map[uint64]*big.Int
}

// NewRoleMgr can be admin by mutiple signatures
func NewRoleMgr(caller, foundation, primaryToken utils.Address, kPledge, pPledge *big.Int) RoleMgr {
	// generate local utils.Address from
	local := utils.GetContractAddress(caller, []byte("RoleMgr"))

	mi := make([]*MintInfo, 4)
	mi[0] = &MintInfo{
		Ratio:    100,
		Size:     1,
		Duration: 1,
	}

	mi[1] = &MintInfo{
		Ratio:    120,
		Size:     100 * 1024 * 1024 * 1024 * 1024, // 100TB
		Duration: 100 * 24 * 60 * 60,              // 100 days
	}

	mi[2] = &MintInfo{
		Ratio:    150,
		Size:     1024 * 1024 * 1024 * 1024 * 1024, // 1PB
		Duration: 100 * 24 * 60 * 60,               // 100 days
	}

	mi[3] = &MintInfo{
		Ratio:    200,
		Size:     10 * 1024 * 1024 * 1024 * 1024 * 1024, // 10PB
		Duration: 100 * 24 * 60 * 60,                    // 100 days
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

		subPMap: make(map[uint64]*big.Int),
		subSMap: make(map[uint64]*big.Int),
	}

	ti := &tokenInfo{
		Index: 0,
	}

	rm.tokens = append(rm.tokens, primaryToken)
	rm.tInfo[primaryToken] = ti

	bi := &BaseInfo{
		RoleType: 0,
		Index:    uint64(len(rm.addrs)),
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
		return bi.Index, nil
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
		return ti.Index, nil
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
		log.Info(index, " is empty")
		return nil, ErrEmpty
	}

	if bi.IsBanned {
		return nil, ErrPermission
	}

	return bi, nil
}

func (r *roleMgr) RegisterToken(caller, taddr utils.Address) error {
	// verify sign
	// chek existence
	_, ok := r.tInfo[taddr]
	if ok {
		// exist
		return ErrExist
	}

	pp, err := GetPledgePool(r.pledge)
	if err != nil {
		return err
	}

	ti := &tokenInfo{
		Index: uint32(len(r.tokens)),
	}

	r.tokens = append(r.tokens, taddr)
	r.tInfo[taddr] = ti

	return pp.AddToken(r.local, taddr, ti.Index)
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
		Index:    uint64(len(r.addrs)),
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

	pp, err := GetPledgePool(r.pledge)
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
	bi.Extra = blsKey

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

	pp, err := GetPledgePool(r.pledge)
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

func (r *roleMgr) RegisterUser(caller utils.Address, index, gIndex uint64, blsKey []byte) error {
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
	if !gi.IsActive || gi.IsBanned {
		return ErrPermission
	}

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	err = fm.CreateFs(r.local, index)
	if err != nil {
		return err
	}

	bi.RoleType = RoleUser
	bi.GIndex = gIndex
	bi.Extra = blsKey

	return nil
}

// group related
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

	if !gi.IsActive || gi.IsBanned {
		return nil, ErrPermission
	}

	return gi, nil
}

// CreateGroup
func (r *roleMgr) CreateGroup(caller utils.Address, level uint16) error {
	if caller != r.admin {
		return ErrPermission
	}

	gIndex := len(r.groups)

	gi := &GroupInfo{
		Level:     level,
		Keepers:   make([]uint64, 0, 1),
		Providers: make([]uint64, 0, 1),
		Size:      big.NewInt(0),
		Price:     big.NewInt(0),
	}

	r.groups = append(r.groups, gi)

	fs, err := NewFsMgr(r.local, 0, uint64(gIndex))
	if err != nil {
		return err
	}

	gi.FsAddr = fs.GetContractAddress()

	return nil
}

func (r *roleMgr) SetReady(caller utils.Address, gIndex uint64, ksigns [][]byte) error {
	if len(r.groups) <= int(gIndex) {
		return ErrInput
	}

	gi := r.groups[gIndex]
	if gi.IsBanned {
		return ErrPermission
	}

	if gi.IsActive {
		return ErrPermission
	}

	if len(ksigns) < int(gi.Level) {
		return ErrPermission
	}

	// verfiy ksigns

	gi.IsReady = true

	return nil
}

func (r *roleMgr) AddKeeperToGroup(caller utils.Address, index, gIndex uint64, asign []byte) error {
	// verify asign
	if len(r.groups) <= int(gIndex) {
		return ErrInput
	}

	gi := r.groups[gIndex]
	if gi.IsBanned {
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
	if ki.IsActive {
		return ErrPermission
	}

	fsMgr, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	err = fsMgr.AddKeeper(r.local, index)
	if err != nil {
		return err
	}

	gi.Keepers = append(gi.Keepers, index)
	ki.GIndex = gIndex
	ki.IsActive = true

	if len(gi.Keepers) >= int(gi.Level) {
		gi.IsActive = true
	}

	return nil
}

func (r *roleMgr) AddProviderToGroup(caller utils.Address, index, gIndex uint64) error {
	// verify sign by addr[index]
	if len(r.groups) <= int(gIndex) {
		return ErrInput
	}

	gi := r.groups[gIndex]
	if gi.IsBanned {
		return ErrPermission
	}

	pi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	// have been in some group
	if pi.IsActive {
		return ErrPermission
	}

	if pi.RoleType != RoleProvider {
		return ErrRoleType
	}

	gi.Providers = append(gi.Providers, index)
	pi.GIndex = gIndex
	pi.IsActive = true

	return nil
}

// balance related

func (r *roleMgr) GetPledge(caller utils.Address) (*big.Int, *big.Int) {
	return r.pledgeKeeper, r.pledgePro
}

func (r *roleMgr) SetPledgeMoney(caller utils.Address, kPledge, pPledge *big.Int, signature []byte) error {
	// verify sign(hash(kPLedge,pPledge))

	r.pledgeKeeper = kPledge
	r.pledgePro = pPledge
	return nil
}

// 质押，非流动性
func (r *roleMgr) Pledge(caller utils.Address, index uint64, money *big.Int) error {
	// verify sign

	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	if bi.IsBanned {
		return ErrPermission
	}

	pp, err := GetPledgePool(r.pledge)
	if err != nil {
		return err
	}

	addr := r.addrs[index]
	if caller == r.admin {
		// air drop
		addr = r.local
		pt, err := getErcToken(r.tokens[0])
		if err != nil {
			return err
		}
		pt.Approve(r.local, pp.GetOwnerAddress(), money)
	} else if caller != addr {
		return ErrPermission
	}

	return pp.Pledge(r.local, addr, index, money)
}

func (r *roleMgr) Withdraw(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int) error {
	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrInput
	}

	bi, err := r.getInfo(index)
	if err != nil {
		return err
	}

	if bi.IsBanned {
		return ErrPermission
	}

	pp, err := GetPledgePool(r.pledge)
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

func (r *roleMgr) Recharge(caller utils.Address, index uint64, tokenIndex uint32, money *big.Int) error {
	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrInput
	}

	ui, err := r.getInfo(index)
	if err != nil {
		return err
	}

	gi, err := r.getGroupInfo(ui.GIndex)
	if err != nil {
		return err
	}

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	addr := r.addrs[index]
	if caller == r.admin {
		// air drop
		addr = r.local
		pt, err := getErcToken(r.tokens[tokenIndex])
		if err != nil {
			return err
		}
		pt.Approve(r.local, fm.GetOwnerAddress(), money)
	} else if caller != addr {
		return ErrPermission
	}

	return fm.Recharge(r.local, addr, index, tokenIndex, money)
}

func (r *roleMgr) ProWithdraw(caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, ksigns [][]byte) error {
	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrInput
	}

	ui, err := r.getInfo(proIndex)
	if err != nil {
		return err
	}

	gi, err := r.getGroupInfo(ui.GIndex)
	if err != nil {
		return err
	}

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	blost := new(big.Int)
	se := fm.GetSettleInfo(r.local, proIndex, tokenIndex)
	if se != nil {
		blost.Set(se.Lost)
	}

	err = fm.ProWithdraw(r.local, proIndex, tokenIndex, pay, lost)
	if err != nil {
		return err
	}

	if tokenIndex == 0 {
		// add lost to paid
		r.totalPaid.Add(r.totalPaid, lost)
		r.totalPaid.Sub(r.totalPaid, blost)
	}

	return nil
}

func (r *roleMgr) WithdrawFromFs(caller utils.Address, index uint64, tokenIndex uint32, amount *big.Int) error {
	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrInput
	}

	ui, err := r.getInfo(index)
	if err != nil {
		return err
	}

	gi, err := r.getGroupInfo(ui.GIndex)
	if err != nil {
		return err
	}

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	return fm.Withdraw(r.local, index, tokenIndex, amount)
}

// order ops
func (r *roleMgr) AddOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	log.Info("AddOrder")

	// check params
	if size <= 0 {
		return ErrInput
	}

	if end <= start {
		return ErrInput
	}

	// align to day
	_end := end
	if (_end/86400)*86400 != end {
		return ErrInput
	}

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

	if pi.RoleType != RoleProvider {
		return ErrRoleType
	}

	kindex := uint64(0)
	ki, ok := r.info[caller]
	if ok {
		kindex = ki.Index
	}

	if pi.GIndex != ui.GIndex {
		return ErrInput
	}

	gi, err := r.getGroupInfo(ui.GIndex)
	if err != nil {
		return err
	}

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrBalanceNotEnough
	}

	err = fm.AddOrder(r.local, kindex, user, proIndex, start, end, size, nonce, tokenIndex, sprice)
	if err != nil {
		return err
	}

	// 增发
	if tokenIndex == 0 {
		// add sub price and size
		subprice, ok := r.subPMap[end]
		if ok {
			subprice.Add(subprice, sprice)
		} else {
			r.subPMap[end] = new(big.Int).Set(sprice)
		}

		subSize, ok := r.subSMap[end]
		if ok {
			subSize.Add(subSize, new(big.Int).SetUint64(size))
		} else {
			r.subSMap[end] = new(big.Int).SetUint64(size)
		}

		// todo: edge case ok?
		ntime := GetTime()
		if ntime-r.lastMint > 86400 {
			ntime = r.lastMint + 86400
		}

		dur := new(big.Int).SetUint64(ntime - r.lastMint)
		paid := new(big.Int).Mul(r.price, dur)
		// cross day
		if r.lastMint/86400 < ntime/86400 {
			midTime := (ntime / 86400) * 86400
			sp, ok := r.subPMap[midTime]
			if ok {
				subPay := new(big.Int).Mul(sp, new(big.Int).SetUint64(ntime-midTime))
				paid.Sub(paid, subPay)
				// update
				r.price.Sub(r.price, sp)
			}
			ssize, ok := r.subSMap[midTime]
			if ok {
				r.size.Sub(r.size, ssize)
			}
		}
		r.totalPaid.Add(r.totalPaid, paid)

		// update info in group
		gi.Size.Add(gi.Size, new(big.Int).SetUint64(size))
		gi.Price.Add(gi.Price, sprice)

		// update info
		// update spacetime for reward ratio
		st := new(big.Int).Mul(new(big.Int).SetUint64(size), new(big.Int).SetUint64(end-start))
		r.spaceTime.Add(r.spaceTime, st)

		// update total pay
		pay := new(big.Int).Mul(sprice, new(big.Int).SetUint64(end-start))
		r.totalPay.Add(r.totalPay, pay)

		r.size.Add(r.size, new(big.Int).SetUint64(size))
		r.price.Add(r.price, sprice)

		if paid.Cmp(zero) <= 0 {
			r.lastMint = ntime
			return nil
		}

		// cal reward
		for i := r.mintLevel + 1; i < len(r.mint); i++ {
			esize := new(big.Int).SetUint64(r.mint[i].Size)
			if esize.Cmp(r.size) < 0 {
				esize.Set(r.size)
			}

			dur := new(big.Int).SetUint64(r.mint[i].Duration)

			if r.spaceTime.Div(r.spaceTime, esize).Cmp(dur) >= 0 {
				r.mintLevel = i
			} else {
				break
			}
		}
		reward := new(big.Int).Mul(paid, new(big.Int).SetUint64(uint64(r.mint[r.mintLevel].Ratio)))
		reward.Div(reward, new(big.Int).SetUint64(100))

		err = sendBalance(r.tokens[0], r.local, r.pledge, reward)
		if err != nil {
			return err
		}
		log.Info("AddOrder: send to pledge: ", reward)

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

	if pi.GIndex != ui.GIndex {
		return ErrInput
	}

	gi, err := r.getGroupInfo(ui.GIndex)
	if err != nil {
		return err
	}

	kindex := uint64(0)
	ki, ok := r.info[caller]
	if ok {
		kindex = ki.Index
	}

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	err = fm.SubOrder(r.local, kindex, user, proIndex, start, end, size, nonce, tokenIndex, sprice)
	if err != nil {
		return err
	}

	gi.Size.Sub(gi.Size, new(big.Int).SetUint64(size))
	gi.Price.Sub(gi.Price, sprice)

	return nil
}

// order ops
func (r *roleMgr) AddRepair(caller utils.Address, proIndex, newPro, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, psign []byte, ksigns [][]byte) error {
	log.Info("AddOrder")
	// verify ksigns
	// verify psign

	pi, err := r.getInfo(proIndex)
	if err != nil {
		return err
	}

	if pi.RoleType != RoleProvider {
		return ErrRoleType
	}

	npi, err := r.getInfo(newPro)
	if err != nil {
		return err
	}

	if npi.RoleType != RoleProvider {
		return ErrRoleType
	}

	kindex := uint64(0)
	ki, ok := r.info[caller]
	if ok {
		kindex = ki.Index
	}

	if pi.GIndex != npi.GIndex {
		return ErrInput
	}

	gi, err := r.getGroupInfo(npi.GIndex)
	if err != nil {
		return err
	}

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	if tokenIndex >= uint32(len(r.tokens)) {
		return ErrBalanceNotEnough
	}

	err = fm.AddRepair(r.local, kindex, proIndex, newPro, start, end, size, nonce, tokenIndex, sprice)
	if err != nil {
		return err
	}

	// modify size

	// 增发
	if tokenIndex == 0 {
		pay := new(big.Int).Mul(sprice, new(big.Int).SetUint64(end-start))
		r.totalPaid.Sub(r.totalPaid, pay) // sub due to lost
	}

	return nil
}

func (r *roleMgr) SubRepair(caller utils.Address, proIndex, newPro, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, psign []byte, ksigns [][]byte) error {
	pi, err := r.getInfo(newPro)
	if err != nil {
		return err
	}

	if pi.RoleType != RoleProvider {
		return ErrRoleType
	}

	gi, err := r.getGroupInfo(pi.GIndex)
	if err != nil {
		return err
	}

	kindex := uint64(0)
	ki, ok := r.info[caller]
	if ok {
		kindex = ki.Index
	}

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		return err
	}

	err = fm.SubRepair(r.local, kindex, proIndex, newPro, start, end, size, nonce, tokenIndex, sprice)
	if err != nil {
		return err
	}

	// modify size

	return nil
}
