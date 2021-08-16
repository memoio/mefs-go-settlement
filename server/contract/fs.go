package contract

import (
	"math/big"
	"strconv"
	"time"

	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
)

// storeInfo is at some time
type storeInfo struct {
	time  uint64   // 什么时刻的状态，start time of each cycle
	size  uint64   // 在该存储节点上的存储总量，byte
	price *big.Int // 按周期计费; per cycle
}

// channelInfo for user pay read to provider
type channelInfo struct {
	amount *big.Int // available amount
	nonce  uint64   // 防止channel重复提交，pro提交后+1
	expire uint64   // 用于channel到期，user取回
}

// aggOrder is AggregatedOrder is user->provider order and channel
type aggOrder struct {
	nonce    uint64                  // 防止order重复提交
	subNonce uint64                  // 用于订单到期
	sInfo    map[uint32]*storeInfo   // 不同代币的支付信息
	channel  map[uint32]*channelInfo // tokenaddr->channel
}

// fsInfo each user have at most one per group
type fsInfo struct {
	isActive   bool
	tokenIndex uint32               // 当前使用哪种token计费?记录
	providers  []uint64             // provider地址的数组
	ao         map[uint64]*aggOrder // 该User对每个Provider的订单信息
}

// settlement is
type settlement struct {
	time  uint64   // store状态改变或pay时间, align to epoch
	size  uint64   // 在该存储节点上的存储总量
	price *big.Int // per byte*second

	maxPay  *big.Int // 对此provider所有user聚合总额度；expected 加和
	hasPaid *big.Int // 已经支付
	lost    *big.Int // lost due to unable response to chal
	canPay  *big.Int // 最近一次store/pay时刻，可以支付的金额

	taxPay     *big.Int // pay for group keepers >= endPaid+linearPaid
	endPaid    *big.Int // release when order expire
	linearPaid *big.Int // release when pay for provider
}

func newSettlement() *settlement {
	return &settlement{
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
func (s *settlement) add(start, size uint64, sprice, pay, tax *big.Int) {
	// update canPay
	hp := new(big.Int)
	if s.time < start {
		hp.SetUint64(start - s.time)
		s.time = start
	} else if s.time > start {
		// add
		hp.SetUint64(s.time - start)
	}

	hp.Mul(hp, s.price)
	s.canPay.Add(s.canPay, hp)

	// update price and size
	s.price.Add(s.price, sprice)
	s.size += s.size

	s.maxPay.Add(s.maxPay, pay)

	// pay to keeper, 5% of pay
	s.taxPay.Add(s.taxPay, tax)
}

// Sub ends
func (s *settlement) sub(start, end, size uint64, sprice *big.Int) {
	// update canPay
	hp := new(big.Int).SetUint64(end - s.time)
	hp.Mul(hp, s.price)
	s.canPay.Add(s.canPay, hp)

	if s.time < end {
		s.time = end
	}

	// update size and price
	s.price.Sub(s.price, sprice)
	s.size -= size
}

// Calc ends called by withdraw
func (s *settlement) calc(pay, lost *big.Int) (*big.Int, error) {
	res := new(big.Int)
	// has paid
	if s.hasPaid.Cmp(pay) > 0 {
		return res, ErrRes
	}
	// lost is not rigth
	if lost.Cmp(s.lost) < 0 {
		return res, ErrRes
	}
	s.lost.Set(lost)

	ntime := uint64(time.Now().Unix())
	if s.time < ntime {
		hp := new(big.Int).SetUint64(ntime - s.time)
		hp.Mul(hp, s.price)
		s.canPay.Add(s.canPay, hp)
		s.time = ntime
	}

	// can pay is right
	if s.canPay.Cmp(pay) < 0 {
		return res, ErrRes
	}

	if s.hasPaid.Cmp(pay) > 0 {
		return res, ErrRes
	}

	res.Sub(pay, s.hasPaid)
	s.hasPaid.Set(pay)

	return res, nil
}

type multiKey struct {
	roleIndex  uint64
	tokenIndex uint32
}

var _ FsMgr = (*fsMgr)(nil)

type fsMgr struct {
	local      utils.Address // contract of this mgr
	admin      utils.Address // owner
	roleAddr   utils.Address // address of roleMgr
	foundation utils.Address // foundation address

	taxRate int    // %5 for group, 4% linear and 1% at end;
	gIndex  uint64 // belongs to which group?

	balance map[multiKey]*big.Int // available

	users []uint64
	fs    map[uint64]*fsInfo

	keepers    []uint64
	period     uint64
	lastTime   uint64              // 上次分润时间
	tAcc       map[uint32]*big.Int // 每次分润后归0
	totalCount uint64
	count      map[uint64]uint64 //   记录keeper触发的次数，用于分润

	providers []uint64
	proInfo   map[multiKey]*settlement

	tokens []uint32 // user使用某token时候加进来
}

// NewFsMgr creates an instance
func NewFsMgr(caller, rAddr, founder utils.Address, gIndex uint64) (FsMgr, error) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		return nil, err
	}

	keepers, err := rm.GetKeepersByIndex(caller, gIndex)
	if err != nil {
		return nil, err
	}

	h := blake2b.New256()
	h.Write([]byte("FsMgr"))
	h.Write([]byte(strconv.FormatUint(gIndex, 10)))

	local := utils.GetContractAddress(caller, h.Sum(nil))
	fm := &fsMgr{
		local:    local,
		admin:    caller,
		roleAddr: rAddr,
		gIndex:   gIndex,
		taxRate:  5, // 5% for keeper
		balance:  make(map[multiKey]*big.Int),

		users: make([]uint64, 0, 1),
		fs:    make(map[uint64]*fsInfo),

		keepers:    keepers,
		period:     10,
		lastTime:   uint64(time.Now().Unix()),
		tAcc:       make(map[uint32]*big.Int),
		totalCount: 0,
		count:      make(map[uint64]uint64),

		providers: make([]uint64, 0, 1),
		proInfo:   make(map[multiKey]*settlement),

		tokens: make([]uint32, 0, 1),
	}

	for _, kp := range keepers {
		fm.count[kp] = 1
		fm.totalCount++
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

func (f *fsMgr) GetTokens(caller utils.Address) []uint32 {
	return f.tokens
}

func (f *fsMgr) GetBalance(caller utils.Address, index uint64, tIndex uint32) (*big.Int, *big.Int, *big.Int) {
	res := new(big.Int)
	lock := new(big.Int)
	paid := new(big.Int)
	mk := multiKey{
		roleIndex:  index,
		tokenIndex: tIndex,
	}
	val, ok := f.balance[mk]
	if ok {
		res.Set(val)
	}

	se, ok := f.proInfo[mk]
	if ok {
		canPay := new(big.Int).Set(se.canPay)
		nt := uint64(time.Now().Unix())
		tmp := new(big.Int).SetUint64(nt - se.time)
		tmp.Mul(tmp, new(big.Int).SetUint64(se.size))
		tmp.Mul(tmp, se.price)

		canPay.Add(canPay, tmp)
		hardlimit := new(big.Int).Sub(se.maxPay, se.lost)

		if canPay.Cmp(hardlimit) > 0 {
			canPay.Set(hardlimit)
		}

		lock.Sub(canPay, se.hasPaid)
		paid.Set(se.hasPaid)
	} else {
		cnt, ok := f.count[index]
		if ok {
			acc, ok := f.tAcc[tIndex]
			if ok {
				tmp := new(big.Int).Div(acc, new(big.Int).SetUint64(f.totalCount))
				tmp.Mul(tmp, new(big.Int).SetUint64(cnt))
				res.Add(res, tmp)
			}
		}
	}

	return res, lock, paid
}

func (f *fsMgr) GetInfo(caller utils.Address) uint64 {
	return f.gIndex
}

func (f *fsMgr) GetFsInfo(caller utils.Address, user uint64) (uint32, []uint64, error) {
	fi, ok := f.fs[user]
	if !ok {
		return 0, nil, ErrEmpty
	}

	return fi.tokenIndex, fi.providers, nil
}

func (f *fsMgr) AddKeeper(caller utils.Address, kindex uint64) error {
	if caller != f.roleAddr {
		return ErrPermission
	}

	f.keepers = append(f.keepers, kindex)
	f.count[kindex] = 1
	f.totalCount++

	return nil
}

func (f *fsMgr) CreateFs(caller utils.Address, user uint64, payToken uint32, blsKey, sign []byte) error {
	// valid addr is user
	// valid paytoken is valid

	// valid fs is not exist
	fi, ok := f.fs[user]
	if ok {
		if fi.isActive {
			return ErrExist
		}
	} else {
		// add fs
		fi = &fsInfo{
			tokenIndex: payToken,
			providers:  make([]uint64, 0, 1),
			ao:         make(map[uint64]*aggOrder),
		}

		f.fs[user] = fi
		f.users = append(f.users, user)
	}

	rm, err := getRoleMgr(f.roleAddr)
	if err != nil {
		return err
	}

	// valid payToken
	_, ok = f.tAcc[payToken]
	if !ok {
		_, err = rm.GetTokenByIndex(f.local, payToken)
		if err != nil {
			return err
		}

		f.tAcc[payToken] = big.NewInt(0)
		f.tokens = append(f.tokens, payToken)
	}

	// notify rm
	err = rm.RegisterUser(f.local, user, f.gIndex, blsKey)
	if err != nil {
		return err
	}

	fi.isActive = true

	return nil
}

func (f *fsMgr) getFsInfo(user uint64) (*fsInfo, error) {
	fi, ok := f.fs[user]
	if !ok {
		return nil, ErrEmpty
	}

	if !fi.isActive {
		return nil, ErrPermission
	}

	return fi, nil
}

// 充值
func (f *fsMgr) Recharge(caller utils.Address, user uint64, tokenIndex uint32, money *big.Int, sign []byte) error {
	_, err := f.getFsInfo(user)
	if err != nil {
		return err
	}

	rm, err := getRoleMgr(f.roleAddr)
	if err != nil {
		return err
	}

	tAddr, err := rm.GetTokenByIndex(f.local, tokenIndex)
	if err != nil {
		return err
	}

	// add tokenIndex
	_, ok := f.tAcc[tokenIndex]
	if !ok {
		f.tAcc[tokenIndex] = big.NewInt(0)
		f.tokens = append(f.tokens, tokenIndex)
	}

	uAddr, err := rm.GetAddressByIndex(f.local, user)
	if err != nil {
		return err
	}

	err = sendBalanceFrom(tAddr, f.local, uAddr, f.local, new(big.Int).Set(money))
	if err != nil {
		return err
	}

	mk := multiKey{
		roleIndex:  user,
		tokenIndex: tokenIndex,
	}

	bal, ok := f.balance[mk]
	if ok {
		bal.Add(bal, money)
	} else {
		f.balance[mk] = money
	}

	return nil
}

func (f *fsMgr) AddOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	// check params
	if size <= 0 {
		return ErrInput
	}

	fi, err := f.getFsInfo(user)
	if err != nil {
		return err
	}

	// get instance by address
	rm, err := getRoleMgr(f.roleAddr)
	if err != nil {
		return err
	}

	_, err = rm.GetAddressByIndex(f.local, user)
	if err != nil {
		return err
	}

	// verify sign

	// verify money is enough
	uKey := multiKey{
		roleIndex:  user,
		tokenIndex: tokenIndex,
	}

	bal, ok := f.balance[uKey]
	if !ok {
		return ErrBalanceNotEnough
	}
	pay := new(big.Int).SetUint64(end - start)
	pay.Mul(pay, sprice)

	tax := new(big.Int).SetUint64(5)
	tax.Mul(tax, pay)
	tax.Div(tax, big.NewInt(100))

	payAndTax := new(big.Int)
	payAndTax.Add(pay, tax)
	if bal.Cmp(payAndTax) < 0 {
		return ErrBalanceNotEnough
	}

	pi, ok := fi.ao[proIndex]
	if !ok {
		gIndex, err := rm.GetGroupByIndex(f.local, proIndex)
		if err != nil {
			return err
		}

		if gIndex != f.gIndex {
			return ErrInput
		}

		fi.providers = append(fi.providers, proIndex)

		pi = &aggOrder{
			nonce:    0,
			subNonce: 0,
			sInfo:    make(map[uint32]*storeInfo),
			channel:  make(map[uint32]*channelInfo),
		}
		fi.ao[proIndex] = pi

		ch := &channelInfo{
			nonce:  0,
			amount: big.NewInt(0), // pay from amount
			expire: end,           // added?
		}
		pi.channel[tokenIndex] = ch
	}

	if pi.nonce != nonce {
		return ErrNonce
	}

	si, ok := pi.sInfo[tokenIndex]
	if !ok {
		si = &storeInfo{
			time:  0,
			size:  0,
			price: big.NewInt(0),
		}
		pi.sInfo[tokenIndex] = si
	}

	if si.time > start {
		return ErrInput
	}

	si.price.Add(si.price, sprice)

	si.size += size

	pKey := multiKey{
		roleIndex:  proIndex,
		tokenIndex: tokenIndex,
	}

	se, ok := f.proInfo[pKey]
	if !ok {
		se = newSettlement()
		f.proInfo[pKey] = se
	}

	se.add(start, size, sprice, pay, tax)

	bal.Sub(bal, payAndTax)
	pi.nonce++

	ind, err := rm.GetIndex(f.local, caller)
	if err != nil {
		return err
	}

	cnt, ok := f.count[ind]
	if ok {
		cnt++
		f.count[ind] = cnt
		f.totalCount++
	}

	return nil
}

func (f *fsMgr) SubOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error {
	// check params
	if size <= 0 {
		return ErrInput
	}

	fi, err := f.getFsInfo(user)
	if err != nil {
		return err
	}

	// get instance by address
	rm, err := getRoleMgr(f.roleAddr)
	if err != nil {
		return err
	}

	// for verify sign
	_, err = rm.GetAddressByIndex(f.local, user)
	if err != nil {
		return err
	}

	pi, ok := fi.ao[proIndex]
	if !ok {
		return ErrEmpty
	}

	if pi.subNonce != nonce {
		return ErrNonce
	}

	si, ok := pi.sInfo[tokenIndex]
	if !ok {
		return ErrEmpty
	}

	// time.N
	if time.Now().Unix() < int64(end) {
		return ErrRes
	}

	if si.size < size {
		return ErrRes
	}

	// verify size and price
	si.price.Sub(si.price, sprice)
	si.size -= size

	pKey := multiKey{
		roleIndex:  proIndex,
		tokenIndex: tokenIndex,
	}

	se, ok := f.proInfo[pKey]
	if !ok {
		return ErrEmpty
	}

	se.sub(start, end, size, sprice)

	// pay to keeper, 4% for linear; due to pro no trigger pay
	// todo

	// pay to keeper, 1% for endpay
	endPaid := new(big.Int).SetUint64(end - start)
	endPaid.Mul(endPaid, sprice)
	endPaid.Div(endPaid, new(big.Int).SetUint64(100))

	f.addToGroup(tokenIndex, endPaid)

	se.endPaid.Add(se.endPaid, endPaid)
	pi.subNonce++

	ind, err := rm.GetIndex(f.local, caller)
	if err != nil {
		return err
	}

	cnt, ok := f.count[ind]
	if ok {
		cnt++
		f.count[ind] = cnt
		f.totalCount++
	}

	return nil
}

func (f *fsMgr) ProWithdraw(caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, sign []byte) error {
	pKey := multiKey{
		roleIndex:  proIndex,
		tokenIndex: tokenIndex,
	}

	se, ok := f.proInfo[pKey]
	if !ok {
		return ErrEmpty
	}

	// pay to provider
	thisPay, err := se.calc(pay, lost)
	if err != nil {
		return err
	}

	// linear pay to keepers
	lpay := new(big.Int).Div(se.hasPaid, big.NewInt(100))
	lpay.Mul(lpay, big.NewInt(4))
	if lpay.Cmp(se.linearPaid) > 0 {
		lpay.Sub(lpay, se.linearPaid)
		f.addToGroup(tokenIndex, lpay) // 交管理费
		se.linearPaid.Add(se.linearPaid, lpay)
	}

	pb, ok := f.balance[pKey]
	if ok {
		thisPay.Add(thisPay, pb)
	} else {
		pb = new(big.Int).Set(thisPay)
		f.balance[pKey] = pb
	}

	// get instance by address
	rm, err := getRoleMgr(f.roleAddr)
	if err != nil {
		return err
	}

	proAddr, err := rm.GetAddressByIndex(f.local, proIndex)
	if err != nil {
		return err
	}

	tAddr, err := rm.GetTokenByIndex(f.local, tokenIndex)
	if err != nil {
		return err
	}

	err = sendBalance(tAddr, f.local, proAddr, thisPay)
	if err != nil {
		return err
	}

	pb.Sub(pb, thisPay)

	return nil
}

func (f *fsMgr) KeeperWithdraw(caller utils.Address, keeper uint64, tokenIndex uint32, amount *big.Int, sign []byte) error {
	ntime := uint64(time.Now().Unix())
	if ntime-f.lastTime > f.period {
		if f.totalCount <= 0 {
			return ErrRes
		}

		for _, tindex := range f.tokens {
			ti, ok := f.tAcc[tindex]
			if !ok {
				return ErrEmpty
			}

			per := new(big.Int).Div(ti, new(big.Int).SetUint64(f.totalCount))
			for _, kindex := range f.keepers {
				kc, ok := f.count[kindex]
				if ok {
					pro := new(big.Int).Mul(per, new(big.Int).SetUint64(kc))
					nk := multiKey{
						tokenIndex: tindex,
						roleIndex:  kindex,
					}
					bal, ok := f.balance[nk]
					if ok {
						bal.Add(bal, pro)
					} else {
						f.balance[nk] = pro
					}
					ti.Sub(ti, pro)
				} else {
					f.count[kindex] = 1
				}
			}
		}

		f.lastTime = ntime
	}

	// update count?

	nk := multiKey{
		tokenIndex: tokenIndex,
		roleIndex:  keeper,
	}

	bal, ok := f.balance[nk]
	if !ok {
		bal = big.NewInt(0)
		f.balance[nk] = big.NewInt(0)
	}

	if amount.Cmp(zero) == 0 || amount.Cmp(bal) > 0 {
		amount.Set(bal)
	}

	rm, err := getRoleMgr(f.roleAddr)
	if err != nil {
		return err
	}

	tAddr, err := rm.GetTokenByIndex(f.local, tokenIndex)
	if err != nil {
		return err
	}

	kAddr, err := rm.GetAddressByIndex(f.local, keeper)
	if err != nil {
		return err
	}

	err = sendBalance(tAddr, f.local, kAddr, amount)
	if err != nil {
		return err
	}
	bal.Sub(bal, amount)
	return nil
}

func (f *fsMgr) addToGroup(tokenIndex uint32, amount *big.Int) error {
	ti, ok := f.tAcc[tokenIndex]
	if !ok {
		return ErrEmpty
	}

	tv := new(big.Int).Set(amount)
	tv.Div(tv, new(big.Int).SetInt64(int64(len(f.keepers))))
	ti.Add(ti, tv)

	return nil
}
