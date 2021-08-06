package role

import (
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

// 某时刻的存储信息
type StoreInfo struct {
	time       uint64   // 什么时刻的状态，start time of each cycle
	size       uint64   // 在该存储节点上的存储总量，byte
	price      *big.Int // 按周期计费，比如一周期为一个区块
	tokenIndex uint32   // 指明使用哪种代币付费
}

// 支付通道信息
type ChannelInfo struct {
	amount *big.Int // available amount
	nonce  uint64   // 防止channel重复提交，pro提交后+1
	expire uint64   // 用于channel到期，user取回
}

// 为链下某user与某Provider所有订单聚合而成的结算数据
// 自带了Channel合约
type AggregatedOrder struct {
	isActive bool                    // 是否继续可用; add order 时候判断
	index    uint64                  // provider index
	sInfo    map[uint32]*StoreInfo   // 不同代币的支付信息
	nonce    uint64                  // 防止order重复提交
	subNonce uint64                  // 用于订单到期
	channel  map[uint32]*ChannelInfo // tokenaddr->channel
}

// 文件系统
type FSInfo struct {
	tokenIndex uint32                      // 当前使用哪种token计费
	user       uint64                      // 哪个user
	keepers    []uint64                    // keeper地址的数组, 从pledge合约获取
	providers  []uint64                    // provider地址的数组
	amount     map[uint32]*big.Int         // available money in token, order进来的时候，减去相应的值到ProviderOrder中的maxPay
	ao         map[uint64]*AggregatedOrder // 该User对每个Provider的订单信息
}

// users -> provider
type Settlement struct {
	maxPay     *big.Int // 对此provider所有user聚合总额度；expected 加和
	hasPaid    *big.Int // 已经支付
	canPay     *big.Int // 最近一次store/pay时刻，可以支付的金额
	time       uint64   // store状态改变或pay时间
	storeBytes uint64   // 在该存储节点上的存储总量
	price      *big.Int // per epoch
	tokenIndex uint64
}

type ProSettlement struct { //记录某Provider所有订单信息
	index  uint64 // provider序号
	settle Settlement
}

type fsMgr struct {
	local utils.Address // contract of this mgr
	admin utils.Address // owner

	roleAddr utils.Address

	gIndex  uint64 // belongs to which group?
	fsInfo  []*FSInfo
	proInfo map[uint64]*ProSettlement
}

func NewFsMgr(caller, rAddr utils.Address) *fsMgr {
	local := utils.GetContractAddress(caller, []byte("FsMgr"))
	fm := &fsMgr{
		local:    local,
		admin:    caller,
		roleAddr: rAddr,
		fsInfo:   make([]*FSInfo, 0, 1),
	}

	return fm
}

func (f *fsMgr) CreateFs(user uint64, payToken uint32, keeper, provider []uint64, sign []byte) *FSInfo {
	// valid addr is user
	// valid paytoken is valid

	// valid each keeper
	// valid each provider

	// query money in each token

	// valid
	fi := &FSInfo{
		tokenIndex: payToken,
		user:       user,
		keepers:    keeper,
		providers:  provider,
		amount:     make(map[uint32]*big.Int),
		ao:         make(map[uint64]*AggregatedOrder),
	}

	return fi
}

// 充值
func (f *fsMgr) Recharge(fsIndex uint64, tokenIndex uint32, money *big.Int, sign []byte) error {
	if fsIndex >= uint64(len(f.fsInfo)) {
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

func (f *fsMgr) AddOrder(fsIndex, proIndex, start, end, size uint64, tokenIndex uint32, price *big.Int, sign []byte) error {
	// check params
	if size <= 0 {
		return ErrRes
	}

	if fsIndex >= uint64(len(f.fsInfo)) {
		return ErrRes
	}

	fi := f.fsInfo[fsIndex]

	bal, ok := fi.amount[tokenIndex]
	if !ok {
		return ErrRes
	}

	pay := new(big.Int).SetUint64(end - start)
	pay.Mul(pay, price)

	if pay.Cmp(bal) < 0 {
		return ErrRes
	}

	rm, err := getRoleMgrByAddress(f.roleAddr)
	if err != nil {
		return err
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

		pi = &AggregatedOrder{
			isActive: true,
			index:    0,
			nonce:    0,
			subNonce: 0,
			sInfo:    make(map[uint32]*StoreInfo),
			channel:  make(map[uint32]*ChannelInfo),
		}

		fi.ao[proIndex] = pi

		newPay := big.NewInt(10000)
		newPay.Add(newPay, pay)

		if newPay.Cmp(fi.amount[tokenIndex]) >= 0 {
			ch := &ChannelInfo{
				nonce:  0,
				amount: big.NewInt(10000),
				expire: end, // added?
			}

			pi.channel[tokenIndex] = ch
			fi.amount[tokenIndex].Sub(fi.amount[tokenIndex], ch.amount)
		}
	}

	si, ok := pi.sInfo[tokenIndex]
	if !ok {
		si = &StoreInfo{
			time:  start,
			size:  0,
			price: big.NewInt(1),
		}

		pi.sInfo[tokenIndex] = si
	}

	if si.time < start {
		return ErrRes
	}

	si.price.Mul(si.price, new(big.Int).SetUint64(si.size))
	si.price.Add(si.price, price)

	si.size += size

	si.price.Div(si.price, new(big.Int).SetUint64(si.size))

	uAddr, err := rm.GetAddressByIndex(fi.user)
	if err != nil {
		return err
	}

	// verify sign
	uAddr.String()

	return nil
}
