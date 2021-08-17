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
	local    utils.Address // contract of this mgr
	admin    utils.Address // owner
	roleAddr utils.Address // address of roleMgr

	taxRate int    // %5 for group, 4% linear and 1% at end;
	gIndex  uint64 // belongs to which group?

	balance map[multiKey]*big.Int // available

	users []uint64
	fs    map[uint64]*fsInfo

	providers []uint64
	proInfo   map[multiKey]*settlement

	tokens []uint32 // user使用某token时候加进来
}

// NewFsMgr creates an instance
func NewFsMgr(caller, rAddr utils.Address, gIndex uint64) (FsMgr, error) {
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

		providers: make([]uint64, 0, 1),
		proInfo:   make(map[multiKey]*settlement),

		tokens: make([]uint32, 0, 1),
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

func (f *fsMgr) CreateFs(caller utils.Address, user uint64, payToken uint32) error {
	// call by roleMgr
	if caller != f.roleAddr {
		return ErrPermission
	}

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

func (f *fsMgr) AddOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice, pay, tax *big.Int, usign, psign []byte, ksigns [][]byte) error {
	if caller != f.roleAddr {
		return ErrPermission
	}

	// verify ksigns
	// verify usign
	// verify psign

	// check params
	if size <= 0 {
		return ErrInput
	}

	fi, err := f.getFsInfo(user)
	if err != nil {
		return err
	}

	// verify sign
	pi, ok := fi.ao[proIndex]
	if !ok {
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
	pi.nonce++

	return nil
}

func (f *fsMgr) SubOrder(caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) (*big.Int, error) {
	if caller != f.roleAddr {
		return nil, ErrPermission
	}

	// verify ksigns
	// verify usign
	// verify psign

	// check params
	if size <= 0 {
		return nil, ErrInput
	}

	if end <= start {
		return nil, ErrInput
	}

	// time.N
	if time.Now().Unix() < int64(end) {
		return nil, ErrRes
	}

	fi, err := f.getFsInfo(user)
	if err != nil {
		return nil, err
	}

	pi, ok := fi.ao[proIndex]
	if !ok {
		return nil, ErrEmpty
	}

	if pi.subNonce != nonce {
		return nil, ErrNonce
	}

	si, ok := pi.sInfo[tokenIndex]
	if !ok {
		return nil, ErrEmpty
	}

	if si.size < size {
		return nil, ErrRes
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
		return nil, ErrEmpty
	}

	se.sub(start, end, size, sprice)

	// pay to keeper, 1% for endpay
	endPaid := new(big.Int).Mul(sprice, new(big.Int).SetUint64(end-start))
	endPaid.Div(endPaid, new(big.Int).SetUint64(100))
	se.endPaid.Add(se.endPaid, endPaid)

	// pay to keeper, 4% for linear; due to pro no trigger pay
	// todo

	pi.subNonce++

	return endPaid, nil
}

func (f *fsMgr) ProWithdraw(caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, signs [][]byte) (*big.Int, *big.Int, error) {
	// verify ksign?
	pKey := multiKey{
		roleIndex:  proIndex,
		tokenIndex: tokenIndex,
	}

	se, ok := f.proInfo[pKey]
	if !ok {
		return nil, nil, ErrEmpty
	}

	// pay to provider
	thisPay, err := se.calc(pay, lost)
	if err != nil {
		return nil, nil, err
	}

	// linear pay to keepers
	lpay := new(big.Int).Div(se.hasPaid, big.NewInt(100))
	lpay.Mul(lpay, big.NewInt(4))
	if lpay.Cmp(se.linearPaid) > 0 {
		lpay.Sub(lpay, se.linearPaid)
		se.linearPaid.Add(se.linearPaid, lpay)
	}

	return thisPay, lpay, nil
}
