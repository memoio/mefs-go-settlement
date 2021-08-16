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

	rm := NewRoleMgr(adminAddr, et.GetContractAddress(), big.NewInt(123450), big.NewInt(12345))

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

	if len(ts) < 2 {
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
	err = pt.Transfer(pt.GetOwnerAddress(), userAddr, big.NewInt(5000000))
	if err != nil {
		t.Fatal(err)
	}

	addrs := rm.GetAllAddrs(rm.GetOwnerAddress())

	err = rm.Register(userAddr, userAddr, nil)
	if err != nil {
		t.Fatal(err)
	}

	ui, err := rm.GetInfo(userAddr, userAddr)
	if err != nil {
		t.Fatal(err)
	}

	if ui.index != uint64(len(addrs)) {
		t.Fatal("register fails")
	}

	pt.Approve(userAddr, rm.GetContractAddress(), amount)

	err = rm.Pledge(userAddr, ui.index, amount, nil)
	if err != nil {
		t.Fatal(err)
	}

	ui, _, err = rm.GetInfoByIndex(userAddr, ui.index)
	if err != nil {
		t.Fatal(err)
	}
	if ui.rewards[0].Cmp(amount) != 0 {
		t.Fatal("pledge fail")
	}

	return ui.index
}

func testWithdraw(t *testing.T, rAddr utils.Address, index uint64, tIndex uint32, send bool) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	ts := rm.GetAllTokens(rm.GetOwnerAddress())

	et, err := getErcToken(ts[tIndex])
	if err != nil {
		t.Fatal(err)
	}

	ui, userAddr, err := rm.GetInfoByIndex(rm.GetOwnerAddress(), index)
	if err != nil {
		t.Fatal(err)
	}

	if send {
		err = et.Transfer(et.GetOwnerAddress(), rm.GetContractAddress(), big.NewInt(100000000))
		if err != nil {
			t.Fatal(err)
		}
	}

	bal, err := rm.GetBalance(userAddr, ui.index)
	if err != nil {
		t.Fatal(err)
	}

	if len(bal) < 2 {
		t.Fatal("get balance fails")
	}

	t.Log(index, bal)

	before := et.BalanceOf(userAddr, userAddr)
	t.Log(index, before)

	err = rm.Withdraw(userAddr, ui.index, tIndex, big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}

	after := et.BalanceOf(userAddr, userAddr)
	after.Sub(after, before)

	kp, pp, res := rm.GetPledge(userAddr)
	t.Log(res)
	if tIndex == 0 {
		if ui.roleType == roleKeeper {
			after.Add(after, kp)
		} else if ui.roleType == roleProvider {
			after.Add(after, pp)
		}
	}

	if after.Cmp(bal[tIndex]) != 0 {
		t.Fatal("withdraw fails")
	}
}

func testCreateKeeper(t *testing.T, rAddr utils.Address) uint64 {
	index := testPledge(t, rAddr, big.NewInt(123450))
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = rm.RegisterKeeper(rm.GetOwnerAddress(), index, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	return index
}

func testCreateProvider(t *testing.T, rAddr utils.Address) uint64 {
	index := testPledge(t, rAddr, big.NewInt(123450))
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = rm.RegisterProvider(rm.GetOwnerAddress(), index, nil)
	if err != nil {
		t.Fatal(err)
	}

	return index
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
	if len(gs) == 0 || gs[len(gs)-1].level != 7 || !gs[len(gs)-1].isActive {
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

	ks, err := rm.GetKeepersByIndex(rm.GetOwnerAddress(), gIndex)
	if err != nil {
		t.Fatal(err)
	}

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

	ks, err := rm.GetProvidersByIndex(rm.GetOwnerAddress(), gIndex)
	if err != nil {
		t.Fatal(err)
	}

	if ks[len(ks)-1] != pindex {
		t.Fatal("add provider fails")
	}

	return pindex
}

func testCreateFsMgr(t *testing.T, rAddr utils.Address, gIndex uint64) utils.Address {
	adminKey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	adminAddr := utils.ToAddress(adminKey.PubKey)

	fm, err := NewFsMgr(adminAddr, rAddr, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	return fm.GetContractAddress()
}

func testSetFsAddr(t *testing.T, rAddr, fsAddr utils.Address, gIndex uint64) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = rm.SetFsAddrForGroup(rm.GetOwnerAddress(), fsAddr, nil)
	if err == nil {
		t.Fatal("should fail for set fs")
	}

	gi, err := rm.GetGroupInfoByIndex(rAddr, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	if gi.fsAddr == fsAddr {
		t.Fatal("set fs addr fails")
	}
}

func testCreateUser(t *testing.T, rAddr utils.Address, gIndex uint64) uint64 {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	ts := rm.GetAllTokens(rm.GetOwnerAddress())

	if len(ts) < 2 {
		t.Log("wrong token")
	}

	userkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	userAddr := utils.ToAddress(userkey.PubKey)

	addrs := rm.GetAllAddrs(rm.GetOwnerAddress())

	err = rm.Register(userAddr, userAddr, nil)
	if err != nil {
		t.Fatal(err)
	}

	ui, err := rm.GetInfo(userAddr, userAddr)
	if err != nil {
		t.Fatal(err)
	}

	if ui.index != uint64(len(addrs)) {
		t.Fatal("register fails")
	}

	pt, err := getErcToken(ts[0])
	if err != nil {
		t.Fatal(err)
	}

	err = pt.Transfer(pt.GetOwnerAddress(), userAddr, big.NewInt(5000000))
	if err != nil {
		t.Fatal(err)
	}

	et, err := getErcToken(ts[1])
	if err != nil {
		t.Fatal(err)
	}

	err = et.Transfer(et.GetOwnerAddress(), userAddr, big.NewInt(30000000000))
	if err != nil {
		t.Fatal(err)
	}

	gi, err := rm.GetGroupInfoByIndex(userAddr, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	fm, err := getFsMgr(gi.fsAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = fm.CreateFs(userAddr, ui.index, 1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	et.Approve(userAddr, fm.GetContractAddress(), big.NewInt(20000000000))
	err = fm.Recharge(userAddr, ui.index, 1, big.NewInt(20000000000), nil)
	if err != nil {
		t.Fatal(err)
	}

	bal, _, _ := fm.GetBalance(userAddr, ui.index, 1)
	if bal.Cmp(big.NewInt(20000000000)) != 0 {
		t.Fatal("user recharge fails")
	}

	ui, err = rm.GetInfo(userAddr, userAddr)
	if err != nil {
		t.Fatal(err)
	}

	if ui.roleType != roleUser {
		t.Fatal("create fs fails")
	}

	return ui.index
}

func testAddOrder(t *testing.T, rAddr, fsAddr utils.Address, kIndex, userIndex, proIndex, start, end, size, nonce uint64) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	kAddr, _ := rm.GetAddressByIndex(rm.GetOwnerAddress(), kIndex)

	fm, err := getFsMgr(fsAddr)
	if err != nil {
		t.Fatal(err)
	}

	bal, _, _ := fm.GetBalance(kAddr, userIndex, 1)
	_, pbal, _ := fm.GetBalance(kAddr, proIndex, 1)
	t.Log("before:", bal, pbal)
	err = fm.AddOrder(kAddr, userIndex, proIndex, start, end, size, nonce, 1, big.NewInt(600000), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	bal, _, _ = fm.GetBalance(kAddr, userIndex, 1)
	_, pbal, _ = fm.GetBalance(kAddr, proIndex, 1)
	t.Log("after:", bal, pbal)
}

func testSubOrder(t *testing.T, rAddr, fsAddr utils.Address, kIndex, userIndex, proIndex, start, end, size, nonce uint64) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	kAddr, _ := rm.GetAddressByIndex(rm.GetOwnerAddress(), kIndex)

	fm, err := getFsMgr(fsAddr)
	if err != nil {
		t.Fatal(err)
	}

	bal, _, _ := fm.GetBalance(kAddr, userIndex, 1)
	_, pbal, _ := fm.GetBalance(kAddr, proIndex, 1)
	avail, _, _ := fm.GetBalance(kAddr, kIndex, 1)
	t.Log("before:", bal, pbal, avail)
	err = fm.SubOrder(kAddr, userIndex, proIndex, start, end, size, nonce, 1, big.NewInt(600000), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	bal, _, _ = fm.GetBalance(kAddr, userIndex, 1)
	_, pbal, _ = fm.GetBalance(kAddr, proIndex, 1)
	avail, _, _ = fm.GetBalance(kAddr, kIndex, 1)
	t.Log("after:", bal, pbal, avail)

}

func testProWithdraw(t *testing.T, rAddr, fsAddr utils.Address, proIndex uint64, amount *big.Int) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	proAddr, err := rm.GetAddressByIndex(rAddr, proIndex)
	if err != nil {
		t.Fatal(err)
	}

	gIndex, err := rm.GetGroupByIndex(proAddr, proIndex)
	if err != nil {
		t.Fatal(err)
	}

	ks, err := rm.GetKeepersByIndex(proAddr, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	fm, err := getFsMgr(fsAddr)
	if err != nil {
		t.Fatal(err)
	}

	_, lock, paid := fm.GetBalance(proAddr, proIndex, 1)
	t.Log("before wd:", lock, paid)

	for _, keeper := range ks {
		avail, lock, _ := fm.GetBalance(proAddr, keeper, 1)
		t.Log(keeper, avail, lock)
	}
	err = fm.ProWithdraw(proAddr, proIndex, 1, amount, big.NewInt(0), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, lock, paid = fm.GetBalance(proAddr, proIndex, 1)
	t.Log("after wd:", lock, paid)
	for _, keeper := range ks {
		avail, lock, _ := fm.GetBalance(proAddr, keeper, 1)
		t.Log(keeper, avail, lock)
	}
}

func testKeeperWithdraw(t *testing.T, rAddr, fsAddr utils.Address, kIndex uint64, amount *big.Int) {
	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	kAddr, err := rm.GetAddressByIndex(rAddr, kIndex)
	if err != nil {
		t.Fatal(err)
	}

	tAddr, err := rm.GetTokenByIndex(rAddr, 1)
	if err != nil {
		t.Fatal(err)
	}

	fm, err := getFsMgr(fsAddr)
	if err != nil {
		t.Fatal(err)
	}

	avail, _, _ := fm.GetBalance(kAddr, kIndex, 1)
	t.Log("before wd:", avail, getBalance(tAddr, kAddr))

	err = fm.KeeperWithdraw(kAddr, kIndex, 1, amount, nil)
	if err != nil {
		t.Fatal(err)
	}

	avail, _, _ = fm.GetBalance(kAddr, kIndex, 1)
	t.Log("after wd:", avail, getBalance(tAddr, kAddr))
}

func TestRole(t *testing.T) {
	tAddr := testErc(t)
	rAddr := testCreateRoleMgr(t, tAddr)

	tAddr2 := testErc(t)
	testAddToken(t, rAddr, tAddr2)
	uindex1 := testPledge(t, rAddr, big.NewInt(3000))
	testWithdraw(t, rAddr, uindex1, 1, true)
	uindex2 := testPledge(t, rAddr, big.NewInt(1000))
	testWithdraw(t, rAddr, uindex2, 1, true)
	testWithdraw(t, rAddr, uindex1, 0, false)
	testWithdraw(t, rAddr, uindex1, 1, true)
	testWithdraw(t, rAddr, uindex2, 1, true)
	testWithdraw(t, rAddr, uindex2, 1, true)

	var keepers []uint64
	for i := 0; i < 7; i++ {
		ind := testCreateKeeper(t, rAddr)
		keepers = append(keepers, ind)
	}

	gIndex := testCreateGroup(t, rAddr, keepers)

	kIndex := testAddKeeper(t, rAddr, gIndex)
	pIndex := testAddProvider(t, rAddr, gIndex)

	testWithdraw(t, rAddr, kIndex, 0, false)
	testWithdraw(t, rAddr, pIndex, 0, false)

	//fAddr := testCreateFsMgr(t, rAddr, gIndex)
	//testSetFsAddr(t, rAddr, fAddr, gIndex)

	uIndex := testCreateUser(t, rAddr, gIndex)

	rm, err := getRoleMgr(rAddr)
	if err != nil {
		t.Fatal(err)
	}

	gi, err := rm.GetGroupInfoByIndex(rAddr, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	fAddr := gi.fsAddr

	nt := uint64(time.Now().Unix())

	testAddOrder(t, rAddr, fAddr, kIndex, uIndex, pIndex, nt-190, nt+10, 300, 0)
	testAddOrder(t, rAddr, fAddr, kIndex, uIndex, pIndex, nt-80, nt+20, 200, 1)

	testProWithdraw(t, rAddr, fAddr, pIndex, big.NewInt(2000))
	testProWithdraw(t, rAddr, fAddr, pIndex, big.NewInt(2000))
	testProWithdraw(t, rAddr, fAddr, pIndex, big.NewInt(9000))

	time.Sleep(11 * time.Second)
	testSubOrder(t, rAddr, fAddr, kIndex, uIndex, pIndex, nt-190, nt+10, 300, 0)
	time.Sleep(10 * time.Second)
	testSubOrder(t, rAddr, fAddr, kIndex, uIndex, pIndex, nt-80, nt+20, 200, 1)

	testKeeperWithdraw(t, rAddr, fAddr, kIndex, big.NewInt(1000))
	t.Fatal("end")
}
