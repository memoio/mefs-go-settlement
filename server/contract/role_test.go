package contract

import (
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/memoio/go-settlement/utils"
)

func TestErc(t *testing.T) {
	testErc(t)
}

func testErc(t *testing.T) utils.Address {
	adminkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	adminAddr := utils.ToAddress(adminkey.PubKey)

	et := NewErcToken(adminAddr)

	t.Log("et token adminAddr:", et.GetOwnerAddress(), "local:", et.GetContractAddress(), "value:", getBalance(et.GetContractAddress(), adminAddr))

	userkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	userAddr := utils.ToAddress(userkey.PubKey)
	err = sendBalance(et.GetContractAddress(), adminAddr, userAddr, big.NewInt(100000000))
	if err != nil {
		t.Fatal(err)
	}

	if getBalance(et.GetContractAddress(), userAddr).Cmp(big.NewInt(100000000)) != 0 {
		t.Fatal("balance is not right")
	}

	userkey2, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	userAddr2 := utils.ToAddress(userkey2.PubKey)

	err = sendBalanceFrom(et.GetContractAddress(), userAddr2, userAddr, userAddr2, big.NewInt(20000000))
	if err == nil {
		t.Fatal("should fail")
	}

	approve(et.GetContractAddress(), userAddr, userAddr2, big.NewInt(20000000))
	err = sendBalanceFrom(et.GetContractAddress(), userAddr2, userAddr, userAddr2, big.NewInt(20000000))
	if err != nil {
		t.Fatal(err)
	}

	if getBalance(et.GetContractAddress(), userAddr).Cmp(big.NewInt(80000000)) != 0 {
		t.Fatal("balance is not right")
	}

	if getBalance(et.GetContractAddress(), userAddr2).Cmp(big.NewInt(20000000)) != 0 {
		t.Fatal("balance is not right")
	}

	t.Log(getBalance(et.GetContractAddress(), userAddr), getBalance(et.GetContractAddress(), userAddr2))

	return et.GetContractAddress()
}

func testCreateRoleMgr(t *testing.T, tAddr utils.Address) utils.Address {
	adminkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	adminAddr := utils.ToAddress(adminkey.PubKey)

	et, err := getErcToken(tAddr)
	if err != nil {
		t.Fatal(err)
	}

	rm := NewRoleMgr(adminAddr, adminAddr, et.GetContractAddress(), big.NewInt(123450), big.NewInt(12345))

	err = sendBalance(tAddr, et.GetOwnerAddress(), rm.GetContractAddress(), big.NewInt(Token))
	if err != nil {
		t.Fatal(err)
	}

	t.Log("roleMgr has bal:", getBalance(tAddr, rm.GetContractAddress()))

	return rm.GetContractAddress()
}

func testAddToken(t *testing.T, rAddr, tAddr utils.Address) {
	et, err := getErcToken(tAddr)
	if err != nil {
		t.Fatal(err)
	}

	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	ts := rm.GetAllTokens(rm.GetOwnerAddress())

	err = rm.RegisterToken(rm.GetOwnerAddress(), et.GetContractAddress(), nil)
	if err != nil {
		t.Fatal(err)
	}

	ti, err := rm.GetTokenIndex(rm.GetContractAddress(), et.GetContractAddress())
	if err != nil {
		t.Fatal(err)
	}

	if ti != uint32(len(ts)) {
		t.Fatal("add token fail")
	}
}

func testPledge(t *testing.T, rAddr utils.Address, amount *big.Int) uint64 {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	ts := rm.GetAllTokens(rm.GetOwnerAddress())

	if len(ts) != 2 {
		t.Log("wrong token")
	}

	pt, err := getErcToken(ts[0])
	if err != nil {
		t.Fatal(err)
	}

	userkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	userAddr := utils.ToAddress(userkey.PubKey)
	err = pt.Transfer(pt.GetOwnerAddress(), userAddr, amount)
	if err != nil {
		t.Fatal(err)
	}

	addrs := rm.GetAllAddrs(rm.GetOwnerAddress())

	err = rm.Register(userAddr, userAddr, nil)
	if err != nil {
		t.Fatal(err)
	}

	uindex, err := rm.GetIndex(userAddr, userAddr)
	if err != nil {
		t.Fatal(err)
	}

	ui, _, err := rm.GetInfo(userAddr, uindex)
	if err != nil {
		t.Fatal(err)
	}

	if ui.Index != uint64(len(addrs)) {
		t.Fatal("register fails")
	}

	pt.Approve(userAddr, rm.GetPledgeAddress(userAddr), amount)

	t.Log("pledge has:", getBalance(ts[0], rm.GetPledgeAddress(userAddr)))

	err = rm.Pledge(userAddr, ui.Index, amount, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("after pledge has:", getBalance(ts[0], rm.GetPledgeAddress(userAddr)))

	pAddr := rm.GetPledgeAddress(userAddr)

	pp, err := GetPledgePool(pAddr)
	if err != nil {
		t.Fatal(err)
	}

	bal := pp.GetBalance(userAddr, ui.Index)
	if len(bal) == 0 {
		t.Fatal("pledge fail")
	}

	if bal[0].Cmp(amount) != 0 {
		t.Fatal("pledge fail:", bal[0], amount)
	}

	return ui.Index
}

func testWithdrawPledge(t *testing.T, rAddr utils.Address, Index uint64, tIndex uint32, send bool) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	ts := rm.GetAllTokens(rm.GetOwnerAddress())

	et, err := getErcToken(ts[tIndex])
	if err != nil {
		t.Fatal(err)
	}

	ui, userAddr, err := rm.GetInfo(rm.GetOwnerAddress(), Index)
	if err != nil {
		t.Fatal(err)
	}

	if send {
		err = et.Transfer(et.GetOwnerAddress(), rm.GetPledgeAddress(userAddr), big.NewInt(100000000))
		if err != nil {
			t.Fatal(err)
		}
	}

	pAddr := rm.GetPledgeAddress(userAddr)

	pp, err := GetPledgePool(pAddr)
	if err != nil {
		t.Fatal(err)
	}

	bal := pp.GetBalance(userAddr, ui.Index)
	if len(bal) == 0 {
		t.Fatal("pledge fail")
	}

	before := et.BalanceOf(userAddr, userAddr)

	bres := pp.GetPledge(userAddr)

	err = rm.Withdraw(userAddr, ui.Index, tIndex, big.NewInt(0), nil)
	if err != nil {
		t.Fatal(err)
	}

	after := et.BalanceOf(userAddr, userAddr)
	getM := new(big.Int).Sub(after, before)

	res := pp.GetPledge(userAddr)
	val := new(big.Int).Sub(bres[tIndex], res[tIndex])
	// 合约中少的金额和user账户多的金额匹配
	if getM.Cmp(val) != 0 {
		t.Log(Index, before, after)
		t.Fatal(Index, "withdraw fails: ", getM, bres[tIndex], res[tIndex])
	}

	kposit, pposit := rm.GetPledge(userAddr)
	if tIndex == 0 {
		if ui.RoleType == RoleKeeper {
			getM.Add(getM, kposit)
		} else if ui.RoleType == RoleProvider {
			getM.Add(getM, pposit)
		}
	}

	//  Withdraw前用户的余额=取出的余额+最小质押额
	if getM.Cmp(bal[tIndex]) != 0 {
		t.Log(Index, before, after)
		t.Fatal("withdraw fails: ", getM, bal[tIndex])
	}

	t.Log(Index, "withdraw: ", getM)
}

func testCreateKeeper(t *testing.T, rAddr utils.Address) uint64 {

	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	kPledge, _ := rm.GetPledge(rAddr)
	Index := testPledge(t, rAddr, new(big.Int).Mul(kPledge, big.NewInt(10)))

	err = rm.RegisterKeeper(rm.GetOwnerAddress(), Index, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	return Index
}

func testCreateProvider(t *testing.T, rAddr utils.Address) uint64 {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	_, pPledge := rm.GetPledge(rAddr)
	Index := testPledge(t, rAddr, new(big.Int).Mul(pPledge, big.NewInt(10)))

	err = rm.RegisterProvider(rm.GetOwnerAddress(), Index, nil)
	if err != nil {
		t.Fatal(err)
	}

	return Index
}

func testCreateGroup(t *testing.T, rAddr utils.Address, inds []uint64) uint64 {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = rm.CreateGroup(rm.GetOwnerAddress(), inds, 7, nil)
	if err != nil {
		t.Fatal(err)
	}

	gs := rm.GetAllGroups(rm.GetOwnerAddress())
	if len(gs) == 0 || gs[len(gs)-1].Level != 7 || !gs[len(gs)-1].IsActive {
		t.Fatal("create group fails")
	}

	return uint64(len(gs) - 1)
}

func testAddKeeper(t *testing.T, rAddr utils.Address, gIndex uint64) uint64 {
	kindex := testCreateKeeper(t, rAddr)
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = rm.AddKeeperToGroup(rm.GetOwnerAddress(), kindex, gIndex, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	gi, err := rm.GetGroupInfo(rm.GetOwnerAddress(), gIndex)
	if err != nil {
		t.Fatal(err)
	}

	ks := gi.Keepers

	if ks[len(ks)-1] != kindex {
		t.Fatal("add keeper fails")
	}

	return kindex
}

func testAddProvider(t *testing.T, rAddr utils.Address, gIndex uint64) uint64 {
	pindex := testCreateProvider(t, rAddr)
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = rm.AddProviderToGroup(rm.GetOwnerAddress(), pindex, gIndex, nil)
	if err != nil {
		t.Fatal(err)
	}

	gi, err := rm.GetGroupInfo(rm.GetOwnerAddress(), gIndex)
	if err != nil {
		t.Fatal(err)
	}

	ps := gi.Providers

	if ps[len(ps)-1] != pindex {
		t.Fatal("add provider fails")
	}

	return pindex
}

func testCreateUser(t *testing.T, rAddr utils.Address, gIndex uint64) uint64 {
	uindex := testPledge(t, rAddr, big.NewInt(4000))

	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	ui, userAddr, err := rm.GetInfo(rAddr, uindex)
	if err != nil {
		t.Fatal(err)
	}

	ts := rm.GetAllTokens(rm.GetOwnerAddress())

	if len(ts) != 2 {
		t.Log("wrong token")
	}

	pt, err := getErcToken(ts[0])
	if err != nil {
		t.Fatal(err)
	}

	err = rm.RegisterUser(userAddr, ui.Index, 0, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ui, _, err = rm.GetInfo(userAddr, ui.Index)
	if err != nil {
		t.Fatal(err)
	}

	if ui.RoleType != RoleUser {
		t.Fatal("create fs fails")
	}

	gi, err := rm.GetGroupInfo(userAddr, 0)
	if err != nil {
		t.Fatal(err)
	}

	pt.Transfer(pt.GetOwnerAddress(), userAddr, big.NewInt(1000000000000))
	pt.Approve(userAddr, gi.FsAddr, big.NewInt(1000000000000))

	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = fm.Recharge(userAddr, ui.Index, 0, big.NewInt(1000000000000), nil)
	if err != nil {
		t.Fatal(err)
	}

	avail, _, _ := fm.GetBalance(userAddr, ui.Index, 0)
	if avail.Cmp(big.NewInt(1000000000000)) != 0 {
		t.Fatal("recharge fails")
	}

	return ui.Index
}

func testAddOrder(t *testing.T, rAddr utils.Address, kIndex, userIndex, proIndex, start, end, size, nonce uint64) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	_, kAddr, err := rm.GetInfo(rm.GetOwnerAddress(), kIndex)
	if err != nil {
		t.Fatal(err)
	}

	gi, err := rm.GetGroupInfo(kAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		t.Fatal(err)
	}

	bavil, block, bpaid := fm.GetBalance(kAddr, proIndex, 0)
	buavil, bulock, bupaid := fm.GetBalance(kAddr, userIndex, 0)
	bkavil, bklock, bkpaid := fm.GetBalance(kAddr, kIndex, 0)

	t.Log(userIndex, "before:", buavil, bulock, bupaid)
	t.Log(kIndex, "before:", bkavil, bklock, bkpaid)
	t.Log(proIndex, "before:", bavil, block, bpaid)

	err = rm.AddOrder(kAddr, userIndex, proIndex, start, end, size, nonce, 0, big.NewInt(600000), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	avil, lock, paid := fm.GetBalance(kAddr, proIndex, 0)
	kavil, klock, kpaid := fm.GetBalance(kAddr, kIndex, 0)
	uavil, ulock, upaid := fm.GetBalance(kAddr, userIndex, 0)

	t.Log(userIndex, "after:", uavil, ulock, upaid)
	t.Log(kIndex, "after:", kavil, klock, kpaid)
	t.Log(proIndex, "after:", avil, lock, paid)

	pay := new(big.Int).Mul(big.NewInt(600000), new(big.Int).SetUint64(end-start))
	per := new(big.Int).Div(pay, big.NewInt(100))

	tax := new(big.Int).Mul(per, big.NewInt(5))
	payAndTax := new(big.Int).Add(pay, tax)

	uCost := new(big.Int).Sub(buavil, uavil)
	if uCost.Cmp(payAndTax) != 0 {
		t.Fatal("add order to pro fails, user cost not right")
	}

	pErn := new(big.Int).Sub(lock, block)
	if pErn.Cmp(pay) != 0 {
		t.Fatal("add order to pro fails")
	}

}

func testSubOrder(t *testing.T, rAddr utils.Address, kIndex, userIndex, proIndex, start, end, size, nonce uint64) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	_, kAddr, err := rm.GetInfo(rm.GetOwnerAddress(), kIndex)
	if err != nil {
		t.Fatal(err)
	}

	err = rm.SubOrder(kAddr, userIndex, proIndex, start, end, size, nonce, 0, big.NewInt(600000), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func testProWithdraw(t *testing.T, rAddr utils.Address, proIndex uint64, amount, lost *big.Int) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	ts := rm.GetAllTokens(rm.GetOwnerAddress())

	if len(ts) != 2 {
		t.Log("wrong token")
	}

	pt, err := getErcToken(ts[0])
	if err != nil {
		t.Fatal(err)
	}

	_, pAddr, err := rm.GetInfo(rm.GetOwnerAddress(), proIndex)
	if err != nil {
		t.Fatal(err)
	}

	bbal := pt.BalanceOf(pAddr, pAddr)

	gi, err := rm.GetGroupInfo(pAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		t.Fatal(err)
	}

	bavil, block, bpaid := fm.GetBalance(pAddr, proIndex, 0)
	t.Log(proIndex, "before:", bavil, block, bpaid)
	err = fm.ProWithdraw(pAddr, proIndex, 0, amount, lost, nil)
	if err != nil {
		t.Fatal(err)
	}

	bal := pt.BalanceOf(pAddr, pAddr)
	avil, lock, paid := fm.GetBalance(pAddr, proIndex, 0)

	t.Log(proIndex, "after:", avil, lock, paid)

	if paid.Cmp(amount) != 0 {
		t.Fatal("pro withdraw fails")
	}

	bal.Sub(bal, bbal)
	paid.Sub(paid, bpaid)
	if paid.Cmp(bal) != 0 {
		t.Fatal("pro withdraw fails, pro money not right")
	}

	if avil.Cmp(zero) != 0 {
		t.Fatal("pro withdraw fails, pro avail money not right")
	}

	// verify lost
}

func testFsWithdraw(t *testing.T, rAddr utils.Address, kIndex uint64, amount *big.Int) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	ts := rm.GetAllTokens(rm.GetOwnerAddress())

	if len(ts) != 2 {
		t.Log("wrong token")
	}

	pt, err := getErcToken(ts[0])
	if err != nil {
		t.Fatal(err)
	}

	_, kAddr, err := rm.GetInfo(rm.GetOwnerAddress(), kIndex)
	if err != nil {
		t.Fatal(err)
	}

	bbal := pt.BalanceOf(kAddr, kAddr)

	gi, err := rm.GetGroupInfo(kAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	fm, err := GetFsMgr(gi.FsAddr)
	if err != nil {
		t.Fatal(err)
	}

	bavil, block, bpaid := fm.GetBalance(kAddr, kIndex, 0)
	t.Log(kIndex, "before:", bavil, block, bpaid)
	err = fm.Withdraw(kAddr, kIndex, 0, amount, nil)
	if err != nil {
		t.Fatal(err)
	}

	bal := pt.BalanceOf(kAddr, kAddr)
	avil, lock, paid := fm.GetBalance(kAddr, kIndex, 0)

	t.Log(kIndex, "after:", avil, lock, paid)

	bal.Sub(bal, bbal)
	paid.Sub(bavil, avil)
	if paid.Cmp(bal) != 0 {
		t.Fatal("withdraw fails, money not right")
	}

	if amount.Cmp(zero) == 0 && avil.Cmp(zero) != 0 {
		t.Fatal("withdraw fails, avail money not right")
	}
}

func TestRole(t *testing.T) {
	tAddr := testErc(t)
	rAddr := testCreateRoleMgr(t, tAddr)

	tAddr2 := testErc(t)
	testAddToken(t, rAddr, tAddr2)
	uindex1 := testPledge(t, rAddr, big.NewInt(2000))
	testWithdrawPledge(t, rAddr, uindex1, 0, true)
	uindex2 := testCreateKeeper(t, rAddr)
	testWithdrawPledge(t, rAddr, uindex2, 0, true)
	uindex3 := testCreateProvider(t, rAddr)
	testWithdrawPledge(t, rAddr, uindex3, 0, true)

	var keepers []uint64
	for i := 0; i < 7; i++ {
		ind := testCreateKeeper(t, rAddr)
		keepers = append(keepers, ind)
	}

	gIndex := testCreateGroup(t, rAddr, keepers)

	kIndex := testAddKeeper(t, rAddr, gIndex)
	pIndex := testAddProvider(t, rAddr, gIndex)

	testWithdrawPledge(t, rAddr, kIndex, 0, false)
	testWithdrawPledge(t, rAddr, pIndex, 0, false)

	//fAddr := testCreateFsMgr(t, rAddr, gIndex)
	//testSetFsAddr(t, rAddr, fAddr, gIndex)

	uIndex := testCreateUser(t, rAddr, gIndex)

	nt := uint64(time.Now().Unix())

	testAddOrder(t, rAddr, kIndex, uIndex, pIndex, nt-190, nt+10, 300, 0)

	time.Sleep(5 * time.Second)

	testProWithdraw(t, rAddr, pIndex, big.NewInt(1500), big.NewInt(240))
	testProWithdraw(t, rAddr, pIndex, big.NewInt(1800), big.NewInt(450))

	testFsWithdraw(t, rAddr, kIndex, big.NewInt(0))
	testFsWithdraw(t, rAddr, keepers[0], big.NewInt(0))

	testAddOrder(t, rAddr, kIndex, uIndex, pIndex, nt-80, nt+20, 200, 1)

	time.Sleep(6 * time.Second)
	testSubOrder(t, rAddr, kIndex, uIndex, pIndex, nt-190, nt+10, 300, 0)
	time.Sleep(10 * time.Second)
	testSubOrder(t, rAddr, kIndex, uIndex, pIndex, nt-80, nt+20, 200, 1)

	testProWithdraw(t, rAddr, pIndex, big.NewInt(24000), big.NewInt(660))
	testFsWithdraw(t, rAddr, kIndex, big.NewInt(0))
	testFsWithdraw(t, rAddr, keepers[0], big.NewInt(0))
	testFsWithdraw(t, rAddr, uIndex, big.NewInt(1000))
	t.Fatal("end")
}
