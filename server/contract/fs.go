package contract

import (
	"math/big"
	"time"

	"github.com/memoio/go-settlement/utils"
)

// StoreInfo is at some time
type StoreInfo struct {
	time  uint64   // 什么时刻的状态，start time of each cycle
	size  uint64   // 在该存储节点上的存储总量，byte
	price *big.Int // 按周期计费，比如一周期为一个区块?
}

func newStoreInfo() *StoreInfo {
	return &StoreInfo{
		time:  0,
		size:  0,
		price: big.NewInt(1),
	}
}

func (si *StoreInfo) add(time, size uint64, sprice *big.Int) error {
	if si.time < time {
		return ErrRes
	}

	if si.size > 0 {
		si.price.Mul(si.price, new(big.Int).SetUint64(si.size))
	}

	si.price.Add(si.price, sprice)

	si.size += size

	si.price.Div(si.price, new(big.Int).SetUint64(si.size))
	return nil
}

func (si *StoreInfo) sub(end, size uint64, sprice *big.Int) error {
	// time.N
	if time.Now().Unix() < int64(end) {
		return ErrRes
	}

	if si.size < size {
		return ErrRes
	}

	si.price.Mul(si.price, new(big.Int).SetUint64(si.size))

	si.price.Sub(si.price, sprice)

	si.size -= size

	if si.size > 0 {
		si.price.Div(si.price, new(big.Int).SetUint64(si.size))
	}
	return nil
}

// ChannelInfo for user pay read to provider
type ChannelInfo struct {
	amount *big.Int // available amount
	nonce  uint64   // 防止channel重复提交，pro提交后+1
	expire uint64   // 用于channel到期，user取回
}

// AggregatedOrder is user->provider order and channel
type AggregatedOrder struct {
	nonce    uint64                // 防止order重复提交
	subNonce uint64                // 用于订单到期
	sInfo    map[uint32]*StoreInfo // 不同代币的支付信息

	channel map[uint32]*ChannelInfo // tokenaddr->channel
}

func newAggregatedOrder() *AggregatedOrder {
	return &AggregatedOrder{
		nonce:    0,
		subNonce: 0,
		sInfo:    make(map[uint32]*StoreInfo),
		channel:  make(map[uint32]*ChannelInfo),
	}
}

// FSInfo each user have at most one per group
type FSInfo struct {
	tokenIndex uint32              // 当前使用哪种token计费?记录
	user       uint64              // 哪个user
	amount     map[uint32]*big.Int // available money in token

	providers []uint64                    // provider地址的数组
	ao        map[uint64]*AggregatedOrder // 该User对每个Provider的订单信息
}

// Settlement is
type Settlement struct {
	time  uint64   // store状态改变或pay时间, align to epoch
	size  uint64   // 在该存储节点上的存储总量
	price *big.Int // per b*second

	maxPay  *big.Int // 对此provider所有user聚合总额度；expected 加和
	hasPaid *big.Int // 已经支付
	lost    *big.Int // lost due to unable response to chal
	canPay  *big.Int // 最近一次store/pay时刻，可以支付的金额

	taxPay     *big.Int // pay for group keepers > endPaid+linearPaid
	endPaid    *big.Int // release when order expire
	linearPaid *big.Int // release when pay for provider
}

func newSettlement() *Settlement {
	return &Settlement{
		maxPay:  big.NewInt(0),
		hasPaid: big.NewInt(0),
		lost:    big.NewInt(0),
		canPay:  big.NewInt(0),
		price:   big.NewInt(1),

		taxPay:     big.NewInt(0),
		linearPaid: big.NewInt(0),
		endPaid:    big.NewInt(0),
	}
}

// Add is
func (s *Settlement) add(time, size uint64, sprice, pay, tax *big.Int) {
	// update canPay
	hp := new(big.Int)
	if s.time < time {
		hp.SetUint64(time - s.time)
		hp.Mul(hp, new(big.Int).SetUint64(s.size))
		hp.Mul(hp, s.price)

		s.time = time
	} else if s.time > time {
		// add
		hp.SetUint64(s.time - time)
		hp.Mul(hp, sprice)
	}
	s.canPay.Add(s.canPay, hp)

	// update price and size
	s.price.Mul(s.price, new(big.Int).SetUint64(s.size))
	s.price.Add(s.price, sprice)
	s.size += size
	s.price.Div(s.price, new(big.Int).SetUint64(s.size))

	s.maxPay.Add(s.maxPay, pay)

	// pay to keeper, 5% of pay
	s.taxPay.Add(s.taxPay, tax)
}

// Sub ends
func (s *Settlement) sub(start, end, size uint64, sprice *big.Int) {
	// update canPay
	hp := new(big.Int)
	if s.time < end {
		hp.SetUint64(end - s.time)
		hp.Mul(hp, new(big.Int).SetUint64(s.size))
		hp.Mul(hp, s.price)
		s.canPay.Add(s.canPay, hp)
		s.time = end
	} else if s.time > end {
		// sub
		hp.SetUint64(s.time - end)
		hp.Mul(hp, sprice)
		s.canPay.Sub(s.canPay, hp)
	}

	// update size and price
	s.price.Mul(s.price, new(big.Int).SetUint64(s.size))
	s.price.Sub(s.price, sprice)
	s.size -= size
	if s.size > 0 {
		s.price.Div(s.price, new(big.Int).SetUint64(s.size))
	}
}

// Calc ends called by withdraw
func (s *Settlement) calc(pay, lost *big.Int) (*big.Int, error) {
	res := new(big.Int)
	// has paid
	if s.hasPaid.Cmp(pay) > 0 {
		return res, ErrRes
	}
	// lost is not rigth
	if lost.Cmp(s.lost) < 0 {
		return res, ErrRes
	}

	ntime := uint64(time.Now().Unix())
	if s.time < ntime {
		hp := new(big.Int).SetUint64(ntime - s.time)
		hp.Mul(hp, new(big.Int).SetUint64(s.size))
		hp.Mul(hp, s.price)
		s.canPay.Add(s.canPay, hp)
		s.time = ntime
	}

	// can pay is right
	if s.canPay.Cmp(pay) < 0 {
		return res, ErrRes
	}

	return res.Sub(pay, s.hasPaid), nil
}

type nodeKey struct {
	id  uint64 // node
	tid uint32 // token
}

// pay info per token
type tokenPay struct {
	index int
	acc   *big.Int
}

type kAmount struct {
	acc    *big.Int
	amount *big.Int
}

type kSettle struct {
	keepers []uint64
	pay     map[nodeKey]*kAmount
	tokens  []uint32
	tAcc    map[uint32]*big.Int
}

func (k *kSettle) add(tokenIndex uint32, amount *big.Int) error {
	ti, ok := k.tAcc[tokenIndex]
	if !ok {
		// verify tokenIndex first
		ti = big.NewInt(0)
		k.tAcc[tokenIndex] = ti
		k.tokens = append(k.tokens, tokenIndex)

		for _, keeper := range k.keepers {
			nk := nodeKey{
				tid: tokenIndex,
				id:  keeper,
			}
			ki, ok := k.pay[nk]
			if !ok {
				ki = &kAmount{
					acc:    big.NewInt(0),
					amount: big.NewInt(0),
				}
				k.pay[nk] = ki
			}
		}
	}

	tv := new(big.Int).Set(amount)
	tv.Div(tv, new(big.Int).SetInt64(int64(len(k.keepers))))
	ti.Add(ti, tv)

	return nil
}

func (k *kSettle) addKeeper(keeper uint64) error {
	for _, tindex := range k.tokens {
		ti, ok := k.tAcc[tindex]
		if !ok {
			return ErrRes
		}

		nk := nodeKey{
			tid: tindex,
			id:  keeper,
		}

		ki := &kAmount{
			acc:    new(big.Int).Set(ti),
			amount: big.NewInt(0),
		}

		k.pay[nk] = ki
	}

	return nil
}

func (k *kSettle) withdraw(keeper uint64, tokenIndex uint32) error {
	nk := nodeKey{
		tid: tokenIndex,
		id:  keeper,
	}

	ki, ok := k.pay[nk]
	if !ok {
		return ErrRes
	}

	ti, ok := k.tAcc[tokenIndex]
	if !ok {
		return ErrRes
	}

	tmp := new(big.Int).Sub(ti, ki.acc)

	ki.amount.Add(ki.amount, tmp)
	ki.acc.Set(ti)

	return nil
}

var _ FsMgr = (*fsMgr)(nil)

type fsMgr struct {
	local    utils.Address // contract of this mgr
	admin    utils.Address // owner
	roleAddr utils.Address // address of roleMgr

	taxRate int // %5 for group, 4% linear and 1% at end;

	gIndex uint64 // belongs to which group?

	users   map[uint64]uint64 // user -> fs
	fsInfo  []*FSInfo
	kInfo   *kSettle
	proInfo map[nodeKey]*Settlement
}

func NewFsMgr(caller, rAddr utils.Address, gIndex uint64) (*fsMgr, error) {
	rm, err := getRoleMgrByAddress(rAddr)
	if err != nil {
		return nil, err
	}

	keepers, err := rm.GetKeepersByIndex(gIndex)
	if err != nil {
		return nil, err
	}

	ks := &kSettle{
		keepers: keepers,
		pay:     make(map[nodeKey]*kAmount),
		tokens:  make([]uint32, 0, 1),
		tAcc:    make(map[uint32]*big.Int),
	}

	local := utils.GetContractAddress(caller, []byte("FsMgr"))
	fm := &fsMgr{
		local:    local,
		admin:    caller,
		roleAddr: rAddr,
		gIndex:   gIndex,
		taxRate:  5, // 5% for keeper
		users:    make(map[uint64]uint64),
		fsInfo:   make([]*FSInfo, 0, 1),
		proInfo:  make(map[nodeKey]*Settlement),
		kInfo:    ks,
	}

	globalMap[local] = fm

	return fm, nil
}

func (f *fsMgr) GetContractAddress() utils.Address {
	return f.local
}

func (f *fsMgr) GetOwnerAddress() utils.Address {
	return f.admin
}

func (f *fsMgr) GetFsIndex(user uint64) (uint64, error) {
	fs, ok := f.users[user]
	if ok {
		return fs, nil
	}

	return 0, ErrRes
}

func (f *fsMgr) GetFsInfo(user uint64) (uint32, []uint64, error) {
	fsIndex, err := f.GetFsIndex(user)
	if err != nil {
		return 0, nil, ErrRes
	}

	fi := f.fsInfo[fsIndex]
	return fi.tokenIndex, fi.providers, nil
}

func (f *fsMgr) CreateFs(user uint64, payToken uint32, sign []byte) error {
	// valid addr is user
	// valid paytoken is valid

	// valid is not exist
	_, ok := f.users[user]
	if ok {
		return ErrRes
	}

	// valid
	fi := &FSInfo{
		tokenIndex: payToken,
		user:       user,
		amount:     make(map[uint32]*big.Int),
		ao:         make(map[uint64]*AggregatedOrder),
	}

	fsIndex := len(f.fsInfo)
	f.users[user] = uint64(fsIndex)

	f.fsInfo = append(f.fsInfo, fi)

	return nil
}

// 充值
func (f *fsMgr) Recharge(user uint64, tokenIndex uint32, money *big.Int, sign []byte) error {
	fsIndex, err := f.GetFsIndex(user)
	if err != nil {
		return ErrRes
	}
	fi := f.fsInfo[fsIndex]

	rm, err := getRoleMgrByAddress(f.roleAddr)
	if err != nil {
		return err
	}

	tAddr, err := rm.GetTokenByIndex(tokenIndex)
	if err != nil {
		return err
	}

	uAddr, err := rm.GetAddressByIndex(fi.user)
	if err != nil {
		return err
	}

	sendM := big.NewInt(0)
	sendM.Add(sendM, money)

	err = sendBalance(tAddr, uAddr, f.local, sendM)
	if err != nil {
		return err
	}

	bal, ok := fi.amount[tokenIndex]
	if ok {
		bal.Add(bal, money)
	} else {
		fi.amount[tokenIndex] = money
	}

	return nil
}

func (f *fsMgr) AddOrder(user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, sign []byte) error {
	// check params
	if size <= 0 {
		return ErrRes
	}

	fsIndex, err := f.GetFsIndex(user)
	if err != nil {
		return ErrRes
	}

	// get instance by address
	rm, err := getRoleMgrByAddress(f.roleAddr)
	if err != nil {
		return err
	}

	fi := f.fsInfo[fsIndex]

	_, err = rm.GetAddressByIndex(fi.user)
	if err != nil {
		return err
	}

	// verify sign

	// verify money is enough
	bal, ok := fi.amount[tokenIndex]
	if !ok {
		return ErrRes
	}

	pay := new(big.Int).SetUint64(end - start)
	pay.Mul(pay, sprice)

	tax := new(big.Int).SetUint64(5)
	tax.Mul(tax, pay)
	tax.Div(tax, big.NewInt(100))

	payAndTax := new(big.Int)
	payAndTax.Add(pay, tax)
	if payAndTax.Cmp(bal) < 0 {
		return ErrRes
	}

	pi, ok := fi.ao[proIndex]
	if !ok {
		gIndex, err := rm.GetGroupByIndex(proIndex)
		if err != nil {
			return err
		}

		if gIndex != f.gIndex {
			return ErrRes
		}

		fi.providers = append(fi.providers, proIndex)

		pi = newAggregatedOrder()
		fi.ao[proIndex] = pi

		ch := &ChannelInfo{
			nonce:  0,
			amount: big.NewInt(0), // pay from amount
			expire: end,           // added?
		}
		pi.channel[tokenIndex] = ch
	}

	if pi.nonce != nonce {
		return ErrRes
	}

	si, ok := pi.sInfo[tokenIndex]
	if !ok {
		si = newStoreInfo()
		pi.sInfo[tokenIndex] = si
	}

	err = si.add(start, size, sprice)
	if err != nil {
		return ErrRes
	}

	nKey := nodeKey{
		id:  proIndex,
		tid: tokenIndex,
	}

	se, ok := f.proInfo[nKey]
	if !ok {
		se := newSettlement()
		f.proInfo[nKey] = se
	}

	se.add(start, size, sprice, pay, tax)

	fi.amount[tokenIndex].Sub(fi.amount[tokenIndex], payAndTax)
	pi.nonce++
	return nil
}

func (f *fsMgr) SubOrder(user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, sign []byte) error {
	// check params
	if size <= 0 {
		return ErrRes
	}

	fsIndex, err := f.GetFsIndex(user)
	if err != nil {
		return ErrRes
	}

	// get instance by address
	rm, err := getRoleMgrByAddress(f.roleAddr)
	if err != nil {
		return err
	}

	fi := f.fsInfo[fsIndex]

	_, err = rm.GetAddressByIndex(fi.user)
	if err != nil {
		return err
	}

	pi, ok := fi.ao[proIndex]
	if !ok {
		return ErrRes
	}

	if pi.subNonce != nonce {
		return ErrRes
	}

	si, ok := pi.sInfo[tokenIndex]
	if !ok {
		return ErrRes
	}

	err = si.sub(end, size, sprice)
	if err != nil {
		return err
	}

	nKey := nodeKey{
		id:  proIndex,
		tid: tokenIndex,
	}

	se, ok := f.proInfo[nKey]
	if !ok {
		return ErrRes
	}

	se.sub(start, end, size, sprice)

	// pay to keeper, 4% for linear; due to pro no trigger pay
	// todo

	// pay to keeper, 1% for endpay
	endPaid := new(big.Int).SetUint64(end - start)
	endPaid.Mul(endPaid, sprice)
	endPaid.Div(endPaid, new(big.Int).SetUint64(100))

	f.kInfo.add(tokenIndex, endPaid)

	se.endPaid.Add(se.endPaid, endPaid)

	return nil
}

func (f *fsMgr) ProWithdraw(user, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, sign []byte) error {
	fsIndex, err := f.GetFsIndex(user)
	if err != nil {
		return ErrRes
	}

	// get instance by address
	rm, err := getRoleMgrByAddress(f.roleAddr)
	if err != nil {
		return err
	}

	fi := f.fsInfo[fsIndex]

	_, err = rm.GetAddressByIndex(fi.user)
	if err != nil {
		return err
	}

	nKey := nodeKey{
		id:  proIndex,
		tid: tokenIndex,
	}

	se, ok := f.proInfo[nKey]
	if !ok {
		return ErrRes
	}

	// pay to provider
	thisPay, err := se.calc(pay, lost)
	if err != nil {
		return err
	}

	proAddr, err := rm.GetAddressByIndex(proIndex)
	if err != nil {
		return err
	}

	tAddr, err := rm.GetTokenByIndex(tokenIndex)
	if err != nil {
		return err
	}

	err = sendBalance(tAddr, f.local, proAddr, thisPay)
	if err != nil {
		return err
	}

	se.lost = lost
	se.hasPaid = pay

	// linear pay to keepers
	lpay := new(big.Int).Mul(se.hasPaid, big.NewInt(4))
	lpay.Div(lpay, big.NewInt(4))
	if lpay.Cmp(se.linearPaid) > 0 {
		lpay.Sub(lpay, se.linearPaid)
		f.kInfo.add(tokenIndex, lpay)
		se.linearPaid.Add(se.linearPaid, lpay)
	}

	return nil
}

func (f *fsMgr) KeeperWithdraw(keeper uint64, tokenIndex uint32, amount *big.Int, sign []byte) error {
	err := f.kInfo.withdraw(keeper, tokenIndex)
	if err != nil {
		return err
	}

	nk := nodeKey{
		tid: tokenIndex,
		id:  keeper,
	}

	ki, ok := f.kInfo.pay[nk]
	if !ok {
		return ErrRes
	}

	if amount.Cmp(zero) == 0 {
		amount.Set(ki.amount)
	}

	if amount.Cmp(ki.amount) > 0 {
		amount.Set(ki.amount)
	}

	rm, err := getRoleMgrByAddress(f.roleAddr)
	if err != nil {
		return err
	}

	tAddr, err := rm.GetTokenByIndex(tokenIndex)
	if err != nil {
		return err
	}

	kAddr, err := rm.GetAddressByIndex(keeper)
	if err != nil {
		return err
	}

	err = sendBalance(tAddr, f.local, kAddr, new(big.Int).Set(amount))
	if err != nil {
		return err
	}

	ki.amount.Sub(ki.amount, amount)
	return nil
}
