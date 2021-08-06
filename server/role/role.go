package role

import (
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

type baseInfo struct {
	isActive bool   // 是否激活
	isBanned bool   // 是否禁用
	index    uint64 // 序列号
}

type KeeperInfo struct {
	baseInfo
	gIndex uint64 // 所属group
	blsKey []byte // for verify
}

type ProInfo struct {
	baseInfo
	gIndex uint64 // 所属group
}

type UserInfo struct {
	baseInfo
	gIndex uint64 // 所属group
	blsKey []byte // for verify
}

type GroupInfo struct {
	isActive  bool
	isBanned  bool          // 组是否已被禁用
	level     uint16        // security level
	keepers   []uint64      // 里面有哪些Keeper
	providers []uint64      // 有哪些provider
	fsAddr    utils.Address // fs contract addr?
}

var _ RoleMgr = (*roleMgr)(nil)

type roleMgr struct {
	local utils.Address // contract of this mgr
	admin utils.Address // owner

	pledgeAddr utils.Address

	addrs []utils.Address // all

	// manage keepers
	pledgeKeeper *big.Int // pledgeMoney for keeper
	kInfo        map[utils.Address]*KeeperInfo

	// manage provider
	pledgePro *big.Int
	pInfo     map[utils.Address]*ProInfo

	// manage user
	uInfo map[utils.Address]*UserInfo

	groups []*GroupInfo

	// pledge pool
	pledgePool PledgePool
}

// can be admin by mutiple signatures
func NewRoleMgr(caller, primaryToken utils.Address, kPledge, pPledge *big.Int) RoleMgr {
	// generate local utils.Address from
	local := utils.GetContractAddress(caller, []byte("RoleMgr"))

	ppool := &pledgeMgr{
		admin:       caller,
		local:       local,
		totalPledge: new(big.Int),
		pledges:     make(map[utils.Address]*PledgeInfo),
		tokens:      make([]utils.Address, 0, 1),
		tInfo:       make(map[utils.Address]*TokenInfo),
	}

	ppool.tokens = append(ppool.tokens, primaryToken)

	rm := &roleMgr{
		admin: caller,
		local: local,

		addrs: make([]utils.Address, 0, 128),

		pledgeKeeper: kPledge,
		kInfo:        make(map[utils.Address]*KeeperInfo),

		pledgePro: pPledge,
		pInfo:     make(map[utils.Address]*ProInfo),

		uInfo: make(map[utils.Address]*UserInfo),

		groups: make([]*GroupInfo, 0, 1),

		pledgePool: ppool,
	}

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
	return r.pledgePool.AddToken(addr)
}

func (r *roleMgr) Pledge(addr utils.Address, money *big.Int, sign []byte) error {
	// verify sign
	return r.pledgePool.Stake(addr, money)
}

func (r *roleMgr) WithdrawToken(addr, tokenAddr utils.Address, money *big.Int) error {
	return r.pledgePool.WithdrawToken(addr, tokenAddr, money)
}

func (r *roleMgr) GetPledgeInfo(addr utils.Address) (*PledgeInfo, error) {
	return r.pledgePool.GetPledgeInfo(addr)
}

func (r *roleMgr) getKeeper(index uint64) (*KeeperInfo, error) {
	if int(index) >= len(r.addrs) {
		return nil, nil
	}

	addr := r.addrs[index]
	ki, ok := r.kInfo[addr]
	if !ok {
		return nil, nil
	}

	if ki.isBanned {
		return nil, nil
	}

	return ki, nil
}

func (r *roleMgr) getProvider(index uint64) (*ProInfo, error) {
	if int(index) >= len(r.addrs) {
		return nil, nil
	}

	addr := r.addrs[index]
	pi, ok := r.pInfo[addr]
	if !ok {
		return nil, nil
	}

	if pi.isBanned {
		return nil, nil
	}

	return pi, nil
}

func (r *roleMgr) getUser(index uint64) (*UserInfo, error) {
	if int(index) >= len(r.addrs) {
		return nil, nil
	}

	addr := r.addrs[index]
	ui, ok := r.uInfo[addr]
	if !ok {
		return nil, nil
	}

	if ui.isBanned {
		return nil, nil
	}

	return ui, nil
}

func (r *roleMgr) SetPledgeMoney(kPledge, pPledge *big.Int, admin utils.Address, signature []byte) error {
	// verify sign(hash(kPLedge,pPledge))

	r.pledgeKeeper = kPledge
	r.pledgePro = pPledge
	return nil
}

func (r *roleMgr) hasRegister(addr utils.Address) bool {
	_, ok := r.kInfo[addr]
	if ok {
		return ok
	}

	_, ok = r.pInfo[addr]
	if ok {
		return ok
	}

	_, ok = r.uInfo[addr]
	if ok {
		return ok
	}

	return false
}

func (r *roleMgr) RegisterKeeper(addr utils.Address, blsKey, signature []byte) error {
	has := r.hasRegister(addr)
	if has {
		return ErrRes
	}

	pi, err := r.GetPledgeInfo(addr)
	if err != nil {
		return err
	}

	bal := big.NewInt(0)
	bal.Add(bal, pi.amount)
	bal.Sub(bal, pi.locked)

	if bal.Cmp(r.pledgeKeeper) < 0 {
		return ErrRes
	}

	pi.locked.Add(pi.locked, r.pledgeKeeper)

	bi := baseInfo{
		index: uint64(len(r.addrs)),
	}
	ki := &KeeperInfo{
		baseInfo: bi,
		blsKey:   blsKey,
	}

	// addr has money
	// sent money from addr to k.local
	r.addrs = append(r.addrs, addr)
	r.kInfo[addr] = ki

	return nil
}

func (r *roleMgr) RegisterProvider(addr utils.Address, signature []byte) error {
	// verify sign(hash(blsKey, money))
	has := r.hasRegister(addr)
	if has {
		return ErrRes
	}

	ppi, err := r.GetPledgeInfo(addr)
	if err != nil {
		return err
	}

	bal := big.NewInt(0)
	bal.Add(bal, ppi.amount)
	bal.Sub(bal, ppi.locked)

	if bal.Cmp(r.pledgePro) < 0 {
		return ErrRes
	}

	ppi.locked.Add(ppi.locked, r.pledgeKeeper)

	bi := baseInfo{
		index: uint64(len(r.addrs)),
	}
	pi := &ProInfo{
		baseInfo: bi,
	}

	r.addrs = append(r.addrs, addr)
	r.pInfo[addr] = pi

	return nil
}

func (r *roleMgr) RegisterUser(addr utils.Address, blsKey, signature []byte) error {
	// verify sign(hash(blsKey))
	has := r.hasRegister(addr)
	if has {
		return ErrRes
	}

	bi := baseInfo{
		index: uint64(len(r.addrs)),
	}
	ui := &UserInfo{
		baseInfo: bi,
	}

	r.addrs = append(r.addrs, addr)
	r.uInfo[addr] = ui

	return nil
}

func (r *roleMgr) CreateGroup(inds []uint64, level uint16, signature []byte) error {
	// verify utils.Address
	for _, index := range inds {
		// verify each keepr
		ki, err := r.getKeeper(index)
		if err != nil {
			return err
		}

		// have been in some group
		if ki.isActive {
			return nil
		}
	}

	gi := &GroupInfo{
		level:   level,
		keepers: inds,
	}

	gIndex := len(r.groups)

	r.groups = append(r.groups, gi)

	for _, index := range inds {
		ki, _ := r.getKeeper(index)
		ki.isActive = true
		ki.gIndex = uint64(gIndex)
	}

	if len(gi.keepers) > int(level) {
		gi.isActive = true
	}

	return nil
}

func (r *roleMgr) AddKeeperToGroup(index, gIndex uint64, ksign, asign []byte) error {
	if len(r.groups) <= int(gIndex) {
		return nil
	}

	// verify asign

	ki, err := r.getKeeper(index)
	if err != nil {
		return err
	}

	// have been in some group
	if ki.isActive {
		return nil
	}

	// verify psign

	gi := r.groups[gIndex]
	gi.keepers = append(gi.keepers, index)

	ki.gIndex = gIndex
	ki.isActive = true

	if len(gi.keepers) > int(gi.level) {
		gi.isActive = true
	}

	return nil
}

func (r *roleMgr) AddProviderToGroup(index, gIndex uint64, sign []byte) error {
	// verify sign by addr[index]
	if len(r.groups) <= int(gIndex) {
		return nil
	}

	pi, err := r.getProvider(index)
	if err != nil {
		return err
	}

	// have been in some group
	if pi.isActive {
		return nil
	}

	gi := r.groups[gIndex]
	gi.providers = append(gi.providers, index)
	pi.gIndex = gIndex
	pi.isActive = true

	return nil
}

func (r *roleMgr) GetTokenByIndex(index uint32) (utils.Address, error) {
	return r.pledgePool.GetAddressByIndex(index)
}

func (r *roleMgr) GetAddressByIndex(index uint64) (utils.Address, error) {
	if index >= uint64(len(r.addrs)) {
		return utils.Address{}, ErrRes
	}

	return r.addrs[index], nil
}

func (r *roleMgr) GetGroupByIndex(index uint64) (uint64, error) {
	if index >= uint64(len(r.addrs)) {
		return 0, ErrRes
	}

	addr := r.addrs[index]
	ki, ok := r.kInfo[addr]
	if ok {
		return ki.gIndex, nil
	}

	pi, ok := r.pInfo[addr]
	if ok {
		return pi.gIndex, nil
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

	ri, ok := globalMap[rAddr]
	if ok {
		r, ok := ri.(RoleMgr)
		if ok {
			pi, err := r.GetPledgeInfo(rAddr)
			if err != nil {
				return res
			}

			res.Add(res, pi.amount)
			res.Sub(res, pi.locked)

		}
	}

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

			pi, err := r.GetPledgeInfo(rAddr)
			if err != nil {
				return res
			}

			res.Add(res, pi.rewards[tokenIndex-1])
		}
	}

	return res
}
