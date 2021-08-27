package node

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/memoio/go-settlement/server/contract"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
)

var addrMap = make(map[utils.Address]*utils.Key)

func sign(t *testing.T, addr utils.Address, uid uint64) []byte {
	key, ok := addrMap[addr]
	if !ok {
		t.Fatal("no secretkey")
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uid)
	msg := blake2b.Sum256(buf)
	sig, err := utils.Sign(key.SecretKey, msg[:])
	if err != nil {
		t.Fatal(err)
	}

	return sig
}

func signMsg(t *testing.T, addr utils.Address, msg []byte) []byte {
	key, ok := addrMap[addr]
	if !ok {
		t.Fatal("no secretkey")
	}
	sig, err := utils.Sign(key.SecretKey, msg)
	if err != nil {
		t.Fatal(err)
	}

	return sig
}

func testNewKey(t *testing.T) utils.Address {
	adminkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	adminAddr := utils.ToAddress(adminkey.PubKey)
	addrMap[adminAddr] = adminkey
	return adminAddr
}

func testNewNode(t *testing.T) *Node {
	return NewNode()
}

func testErc(t *testing.T, n *Node, admin utils.Address) utils.Address {
	uid := n.GetNonce(admin, admin)
	sig := sign(t, admin, uid)

	taddr, err := n.CreateErcToken(uid, sig, admin)
	if err != nil {
		t.Fatal(err)
	}

	return taddr
}

func testCreateRoleMgr(t *testing.T, n *Node, admin, taddr, founder utils.Address) utils.Address {
	uid := n.GetNonce(admin, admin)
	sig := sign(t, admin, uid)

	raddr, err := n.CreateRoleMgr(uid, sig, admin, founder, taddr)
	if err != nil {
		t.Fatal(err)
	}

	uid = n.GetNonce(admin, admin)
	sig = sign(t, admin, uid)

	err = n.Transfer(uid, sig, taddr, admin, raddr, big.NewInt(1000000000000000))
	if err != nil {
		t.Fatal(err)
	}

	return raddr
}

func testPledge(t *testing.T, n *Node, admin utils.Address, amount *big.Int) uint64 {
	uAddr := testNewKey(t)
	ts := n.GetAllTokens(uAddr)
	if len(ts) < 1 {
		t.Fatal("no token")
	}

	uid := n.GetNonce(admin, admin)
	sig := sign(t, admin, uid)

	err := n.Transfer(uid, sig, ts[0], admin, uAddr, amount)
	if err != nil {
		t.Fatal(err)
	}

	uid = n.GetNonce(admin, uAddr)
	sig = sign(t, uAddr, uid)

	err = n.Register(uid, sig, uAddr, uAddr, nil)
	if err != nil {
		t.Fatal(err)
	}

	plAddr := n.GetPledgeAddress(uAddr)
	uid = n.GetNonce(admin, uAddr)
	sig = sign(t, uAddr, uid)
	err = n.Approve(uid, sig, ts[0], uAddr, plAddr, amount)
	if err != nil {
		t.Fatal(err)
	}

	uindex, err := n.GetIndex(uAddr, uAddr)
	if err != nil {
		t.Fatal(err)
	}
	uid = n.GetNonce(admin, uAddr)
	sig = sign(t, uAddr, uid)
	err = n.Pledge(uid, sig, uAddr, uindex, amount, nil)
	if err != nil {
		t.Fatal(err)
	}

	bal, err := n.GetBalance(uAddr, uindex)
	if err != nil {
		t.Fatal(err)
	}

	if bal[0].Cmp(amount) != 0 {
		t.Fatal("pledge fail:", bal[0], amount)
	}

	return uindex
}

func testWithdrawPledge(t *testing.T, n *Node, admin utils.Address, index uint64, tIndex uint32, amount *big.Int, send bool) {
	uAddr, err := n.GetAddr(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	ts := n.GetAllTokens(uAddr)
	if len(ts) < int(tIndex)+1 {
		t.Fatal("no token")
	}

	if send {
		plAddr := n.GetPledgeAddress(uAddr)
		uid := n.GetNonce(admin, admin)
		sig := sign(t, admin, uid)
		val := new(big.Int).Mul(new(big.Int).SetUint64(1), new(big.Int).SetUint64(contract.Token))
		err := n.Transfer(uid, sig, ts[tIndex], admin, plAddr, val)
		if err != nil {
			t.Fatal(err)
		}
	}

	bbal, err := n.GetBalance(uAddr, index)
	if err != nil {
		t.Fatal(err)
	}

	before := n.BalanceOf(ts[tIndex], uAddr, uAddr)
	bres := n.GetPledgeBalance(uAddr)

	uid := n.GetNonce(admin, uAddr)
	sig := sign(t, uAddr, uid)
	err = n.Withdraw(uid, sig, uAddr, index, tIndex, amount, nil)
	if err != nil {
		t.Fatal(err)
	}

	bal, err := n.GetBalance(uAddr, index)
	if err != nil {
		t.Fatal(err)
	}

	after := n.BalanceOf(ts[tIndex], uAddr, uAddr)
	getM := new(big.Int).Sub(after, before)

	res := n.GetPledgeBalance(uAddr)

	val := new(big.Int).Sub(bres[tIndex], res[tIndex])
	// 合约中少的金额和user账户多的金额匹配
	if getM.Cmp(val) != 0 {
		t.Log(index, before, after)
		t.Fatal(index, "withdraw fails: ", getM, bres[tIndex], res[tIndex])
	}

	//  Withdraw前后用户的余额=取出的余额+最小质押额
	ba := new(big.Int).Sub(bbal[tIndex], bal[tIndex])
	if getM.Cmp(ba) != 0 {
		t.Fatal("withdraw fails: ", getM, bbal[tIndex], bal[tIndex])
	}

	t.Log(index, "withdraw:", getM)
}

func testCreateKeeper(t *testing.T, n *Node, admin utils.Address) uint64 {
	kPledge := n.GetKeeperPledge(admin)
	index := testPledge(t, n, admin, new(big.Int).Mul(kPledge, big.NewInt(3)))

	ui, err := n.GetInfo(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	uAddr, err := n.GetAddr(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	uid := n.GetNonce(admin, uAddr)
	sig := sign(t, uAddr, uid)

	err = n.RegisterKeeper(uid, sig, uAddr, index, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if ui.RoleType != contract.RoleKeeper {
		t.Fatal("RegisterKeeper fails")
	}

	return index
}

func testCreateProvider(t *testing.T, n *Node, admin utils.Address) uint64 {
	pPledge := n.GetProviderPledge(admin)
	index := testPledge(t, n, admin, new(big.Int).Mul(pPledge, big.NewInt(3)))

	ui, err := n.GetInfo(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	uAddr, err := n.GetAddr(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	uid := n.GetNonce(admin, uAddr)
	sig := sign(t, uAddr, uid)

	err = n.RegisterProvider(uid, sig, uAddr, index, nil)
	if err != nil {
		t.Fatal(err)
	}

	if ui.RoleType != contract.RoleProvider {
		t.Fatal("RegisterProvider fails")
	}

	return index
}

func testCreateGroup(t *testing.T, n *Node, admin utils.Address, inds []uint64) uint64 {
	gs := n.GetAllGroups(admin)

	uid := n.GetNonce(admin, admin)
	sig := sign(t, admin, uid)
	err := n.CreateGroup(uid, sig, admin, inds, 7, nil)
	if err != nil {
		t.Fatal(err)
	}

	ags := n.GetAllGroups(admin)
	if len(gs)+1 != len(ags) {
		t.Fatal("create group fails")
	}

	return uint64(len(gs))
}

func testAddKeeper(t *testing.T, n *Node, admin utils.Address, gIndex uint64) uint64 {
	gi, err := n.GetGroupInfo(admin, gIndex)
	if err != nil {
		t.Fatal(err)
	}
	knum := len(gi.Keepers)

	kindex := testCreateKeeper(t, n, admin)

	uid := n.GetNonce(admin, admin)
	sig := sign(t, admin, uid)
	err = n.AddKeeperToGroup(uid, sig, admin, kindex, gIndex, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	gi, err = n.GetGroupInfo(admin, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	if knum+1 != len(gi.Keepers) {
		t.Fatal("add keeper fails")
	}

	return kindex
}

func testAddProvider(t *testing.T, n *Node, admin utils.Address, gIndex uint64) uint64 {
	gi, err := n.GetGroupInfo(admin, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	pnum := len(gi.Providers)

	pindex := testCreateProvider(t, n, admin)

	pAddr, err := n.GetAddr(admin, pindex)
	if err != nil {
		t.Fatal(err)
	}

	uid := n.GetNonce(admin, pAddr)
	sig := sign(t, pAddr, uid)

	err = n.AddProviderToGroup(uid, sig, pAddr, pindex, gIndex, nil)
	if err != nil {
		t.Fatal(err)
	}

	gi, err = n.GetGroupInfo(admin, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	if pnum+1 != len(gi.Providers) {
		t.Fatal("add provider fails")
	}

	return pindex
}

func testCreateUser(t *testing.T, n *Node, admin utils.Address, gIndex uint64) uint64 {
	uindex := testPledge(t, n, admin, big.NewInt(4000))

	uAddr, err := n.GetAddr(admin, uindex)
	if err != nil {
		t.Fatal(err)
	}

	uid := n.GetNonce(admin, uAddr)
	sig := sign(t, uAddr, uid)
	err = n.RegisterUser(uid, sig, uAddr, uindex, gIndex, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ui, err := n.GetInfo(admin, uindex)
	if err != nil {
		t.Fatal(err)
	}

	if ui.RoleType != contract.RoleUser {
		t.Fatal("create fs fails")
	}

	// recharge
	ts := n.GetAllTokens(uAddr)
	amount := big.NewInt(4000000000000)

	uid = n.GetNonce(admin, admin)
	sig = sign(t, admin, uid)
	err = n.Transfer(uid, sig, ts[0], admin, uAddr, amount)
	if err != nil {
		t.Fatal(err)
	}

	gi, err := n.GetGroupInfo(uAddr, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	uid = n.GetNonce(admin, uAddr)
	sig = sign(t, uAddr, uid)
	n.Approve(uid, sig, ts[0], uAddr, gi.FsAddr, amount)

	uid = n.GetNonce(admin, uAddr)
	sig = sign(t, uAddr, uid)
	err = n.Recharge(uid, sig, uAddr, uindex, 0, amount, nil)
	if err != nil {
		t.Fatal(err)
	}

	avail, err := n.GetBalanceInFs(uAddr, uindex, 0)
	if err != nil {
		t.Fatal(err)
	}

	if avail[0].Cmp(amount) != 0 {
		t.Fatal("recharge fail")
	}

	return uindex
}

func testAddOrder(t *testing.T, n *Node, admin utils.Address, kIndex, userIndex, proIndex, start, end, size, nonce uint64) {
	kAddr, err := n.GetAddr(admin, kIndex)
	if err != nil {
		t.Fatal(err)
	}

	bp, _ := n.GetBalanceInFs(kAddr, proIndex, 0)
	bu, _ := n.GetBalanceInFs(kAddr, userIndex, 0)
	bk, _ := n.GetBalanceInFs(kAddr, kIndex, 0)

	t.Log(userIndex, "before:", bu[0], bu[1])
	t.Log(kIndex, "before:", bk[0], bk[1])
	t.Log(proIndex, "before:", bp[0], bp[1])

	uid := n.GetNonce(admin, kAddr)
	sig := sign(t, kAddr, uid)

	err = n.AddOrder(uid, sig, kAddr, userIndex, proIndex, start, end, size, nonce, 0, big.NewInt(600000), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	p, _ := n.GetBalanceInFs(kAddr, proIndex, 0)
	u, _ := n.GetBalanceInFs(kAddr, userIndex, 0)
	k, _ := n.GetBalanceInFs(kAddr, kIndex, 0)

	t.Log(userIndex, "after:", u[0], u[1])
	t.Log(kIndex, "after:", k[0], k[1])
	t.Log(proIndex, "after:", p[0], p[1])

	pay := new(big.Int).Mul(big.NewInt(600000), new(big.Int).SetUint64(end-start))
	per := new(big.Int).Div(pay, big.NewInt(100))

	tax := new(big.Int).Mul(per, big.NewInt(5))
	payAndTax := new(big.Int).Add(pay, tax)

	uCost := new(big.Int).Sub(bu[0], u[0])
	if uCost.Cmp(payAndTax) != 0 {
		t.Fatal("add order to pro fails, user cost not right")
	}

	pErn := new(big.Int).Sub(p[1], bp[1])
	if pErn.Cmp(pay) != 0 {
		t.Fatal("add order to pro fails")
	}

}

func testSubOrder(t *testing.T, n *Node, admin utils.Address, kIndex, userIndex, proIndex, start, end, size, nonce uint64) {
	kAddr, err := n.GetAddr(admin, kIndex)
	if err != nil {
		t.Fatal(err)
	}

	bp, _ := n.GetBalanceInFs(kAddr, proIndex, 0)
	bu, _ := n.GetBalanceInFs(kAddr, userIndex, 0)
	bk, _ := n.GetBalanceInFs(kAddr, kIndex, 0)

	t.Log(userIndex, "before:", bu[0], bu[1])
	t.Log(kIndex, "before:", bk[0], bk[1])
	t.Log(proIndex, "before:", bp[0], bp[1])

	uid := n.GetNonce(admin, kAddr)
	sig := sign(t, kAddr, uid)

	err = n.SubOrder(uid, sig, kAddr, userIndex, proIndex, start, end, size, nonce, 0, big.NewInt(600000), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	p, _ := n.GetBalanceInFs(kAddr, proIndex, 0)
	u, _ := n.GetBalanceInFs(kAddr, userIndex, 0)
	k, _ := n.GetBalanceInFs(kAddr, kIndex, 0)

	t.Log(userIndex, "after:", u[0], u[1])
	t.Log(kIndex, "after:", k[0], k[1])
	t.Log(proIndex, "after:", p[0], p[1])
}

func testProWithdraw(t *testing.T, n *Node, admin utils.Address, proIndex uint64, amount, lost *big.Int) {

	pAddr, err := n.GetAddr(admin, proIndex)
	if err != nil {
		t.Fatal(err)
	}

	ts := n.GetAllTokens(admin)

	bbal := n.BalanceOf(ts[0], pAddr, pAddr)

	bp, _ := n.GetBalanceInFs(pAddr, proIndex, 0)
	t.Log(proIndex, "before:", bp[0], bp[1])

	se, _ := n.GetSettleInfo(pAddr, proIndex, 0)
	paid := new(big.Int).Set(se.HasPaid)

	uid := n.GetNonce(admin, pAddr)
	sig := sign(t, pAddr, uid)
	err = n.ProWithdraw(uid, sig, pAddr, proIndex, 0, amount, lost, nil)
	if err != nil {
		t.Fatal(err)
	}

	bal := n.BalanceOf(ts[0], pAddr, pAddr)
	p, _ := n.GetBalanceInFs(pAddr, proIndex, 0)
	t.Log(proIndex, "after:", p[0], p[1])

	bal.Sub(bal, bbal)
	amount.Sub(amount, paid)
	if amount.Cmp(bal) != 0 {
		t.Fatal("pro withdraw fails, pro money not right")
	}
	// verify lost
}

func testFsWithdraw(t *testing.T, n *Node, admin utils.Address, index uint64, amount *big.Int) {
	pAddr, err := n.GetAddr(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	ts := n.GetAllTokens(admin)

	bbal := n.BalanceOf(ts[0], pAddr, pAddr)

	bp, _ := n.GetBalanceInFs(pAddr, index, 0)
	t.Log(index, "before:", bp[0], bp[1])

	uid := n.GetNonce(admin, pAddr)
	sig := sign(t, pAddr, uid)

	err = n.WithdrawFromFs(uid, sig, pAddr, index, 0, amount, nil)
	if err != nil {
		t.Fatal(err)
	}

	bal := n.BalanceOf(ts[0], pAddr, pAddr)
	p, _ := n.GetBalanceInFs(pAddr, index, 0)
	t.Log(index, "after:", p[0], p[1])

	bal.Sub(bal, bbal)
	paid := new(big.Int).Sub(bp[0], p[0])
	if paid.Cmp(bal) != 0 {
		t.Fatal("withdraw fails, money not right")
	}

	if amount.Cmp(big.NewInt(0)) == 0 && p[0].Cmp(big.NewInt(0)) != 0 {
		t.Fatal("withdraw fails, avail money not right")
	}
}

func TestNode(t *testing.T) {
	n := testNewNode(t)
	admin := testNewKey(t)
	founder := testNewKey(t)
	taddr := testErc(t, n, admin)
	testCreateRoleMgr(t, n, admin, taddr, founder)
	index := testPledge(t, n, admin, big.NewInt(1000))
	testWithdrawPledge(t, n, admin, index, 0, big.NewInt(1000), true)
	kindex := testCreateKeeper(t, n, admin)
	testWithdrawPledge(t, n, admin, kindex, 0, big.NewInt(0), true)

	pindex := testCreateProvider(t, n, admin)
	testWithdrawPledge(t, n, admin, pindex, 0, big.NewInt(20000), true)

	var keepers []uint64
	for i := 0; i < 7; i++ {
		ind := testCreateKeeper(t, n, admin)
		keepers = append(keepers, ind)
	}

	gindex := testCreateGroup(t, n, admin, keepers)
	kIndex := testAddKeeper(t, n, admin, gindex)
	pIndex := testAddProvider(t, n, admin, gindex)

	uIndex := testCreateUser(t, n, admin, gindex)

	aindex := testCreateKeeper(t, n, admin)

	nt := uint64(time.Now().Unix())

	testAddOrder(t, n, admin, kIndex, uIndex, pIndex, nt-190, nt+10, 300, 0)

	time.Sleep(5 * time.Second)

	testProWithdraw(t, n, admin, pIndex, big.NewInt(1500), big.NewInt(240))
	testProWithdraw(t, n, admin, pIndex, big.NewInt(1800), big.NewInt(450))

	testFsWithdraw(t, n, admin, kIndex, big.NewInt(0))
	testFsWithdraw(t, n, admin, keepers[0], big.NewInt(0))

	testAddOrder(t, n, admin, kIndex, uIndex, pIndex, nt-80, nt+20, 200, 1)

	time.Sleep(6 * time.Second)
	testSubOrder(t, n, admin, kIndex, uIndex, pIndex, nt-190, nt+10, 300, 0)
	time.Sleep(10 * time.Second)
	testSubOrder(t, n, admin, kIndex, uIndex, pIndex, nt-80, nt+20, 200, 1)

	testProWithdraw(t, n, admin, pIndex, big.NewInt(24000), big.NewInt(660))
	testFsWithdraw(t, n, admin, kIndex, big.NewInt(0))
	testFsWithdraw(t, n, admin, keepers[0], big.NewInt(0))
	testFsWithdraw(t, n, admin, uIndex, big.NewInt(1000))

	testWithdrawPledge(t, n, admin, aindex, 0, big.NewInt(0), false)

	t.Fatal("end")
}
