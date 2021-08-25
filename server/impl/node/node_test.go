package node

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/contract"
	"github.com/memoio/go-settlement/utils"
	"github.com/minio/blake2b-simd"
)

var addrMap = make(map[utils.Address]*utils.Key)

func sign(t *testing.T, addr utils.Address, uid uuid.UUID) []byte {
	key, ok := addrMap[addr]
	if !ok {
		t.Fatal("no secretkey")
	}
	msg := blake2b.Sum256(uid[:])
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
	uid := uuid.New()
	sig := sign(t, admin, uid)

	taddr, err := n.CreateErcToken(uid, sig, admin)
	if err != nil {
		t.Fatal(err)
	}

	return taddr
}

func testCreateRoleMgr(t *testing.T, n *Node, admin, taddr, founder utils.Address) utils.Address {
	uid := uuid.New()
	sig := sign(t, admin, uid)

	raddr, err := n.CreateRoleMgr(uid, sig, admin, founder, taddr)
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

	uid := uuid.New()
	sig := sign(t, admin, uid)

	err := n.Transfer(uid, sig, ts[0], admin, uAddr, amount)
	if err != nil {
		t.Fatal(err)
	}

	uid = uuid.New()
	sig = sign(t, uAddr, uid)

	err = n.Register(uid, sig, uAddr, uAddr, nil)
	if err != nil {
		t.Fatal(err)
	}

	plAddr := n.GetPledgeAddress(uAddr)
	uid = uuid.New()
	sig = sign(t, uAddr, uid)
	err = n.Approve(uid, sig, ts[0], uAddr, plAddr, amount)
	if err != nil {
		t.Fatal(err)
	}

	uindex, err := n.GetIndex(uAddr, uAddr)
	if err != nil {
		t.Fatal(err)
	}
	uid = uuid.New()
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
	_, uAddr, err := n.GetInfo(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	ts := n.GetAllTokens(uAddr)
	if len(ts) < int(tIndex)+1 {
		t.Fatal("no token")
	}

	if send {
		plAddr := n.GetPledgeAddress(uAddr)
		uid := uuid.New()
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
	bkp, bpp, bres := n.GetPledge(uAddr)

	uid := uuid.New()
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

	kp, pp, res := n.GetPledge(uAddr)

	if kp.Cmp(bkp) != 0 {
		t.Fatal("keeper posit change")
	}

	if pp.Cmp(bpp) != 0 {
		t.Fatal("keeper posit change")
	}

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
	kPledge, _, _ := n.GetPledge(admin)
	index := testPledge(t, n, admin, new(big.Int).Mul(kPledge, big.NewInt(3)))

	ui, uAddr, err := n.GetInfo(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	uid := uuid.New()
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
	_, pPledge, _ := n.GetPledge(admin)
	index := testPledge(t, n, admin, new(big.Int).Mul(pPledge, big.NewInt(3)))

	ui, uAddr, err := n.GetInfo(admin, index)
	if err != nil {
		t.Fatal(err)
	}

	uid := uuid.New()
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

	uid := uuid.New()
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
	kindex := testCreateKeeper(t, n, admin)

	uid := uuid.New()
	sig := sign(t, admin, uid)
	err := n.AddKeeperToGroup(uid, sig, admin, kindex, gIndex, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	return kindex
}

func testAddProvider(t *testing.T, n *Node, admin utils.Address, gIndex uint64) uint64 {
	pindex := testCreateProvider(t, n, admin)

	_, pAddr, err := n.GetInfo(admin, pindex)
	if err != nil {
		t.Fatal(err)
	}

	uid := uuid.New()
	sig := sign(t, pAddr, uid)

	err = n.AddProviderToGroup(uid, sig, pAddr, pindex, gIndex, nil)
	if err != nil {
		t.Fatal(err)
	}

	return pindex
}

func testCreateUser(t *testing.T, n *Node, admin utils.Address, gIndex uint64) uint64 {
	uindex := testPledge(t, n, admin, big.NewInt(4000))

	_, uAddr, err := n.GetInfo(admin, uindex)
	if err != nil {
		t.Fatal(err)
	}

	uid := uuid.New()
	sig := sign(t, uAddr, uid)
	err = n.RegisterUser(uid, sig, uAddr, uindex, gIndex, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ui, _, err := n.GetInfo(admin, uindex)
	if err != nil {
		t.Fatal(err)
	}

	if ui.RoleType != contract.RoleUser {
		t.Fatal("create fs fails")
	}

	ts := n.GetAllTokens(uAddr)

	uid = uuid.New()
	sig = sign(t, admin, uid)
	err = n.Transfer(uid, sig, ts[0], admin, uAddr, big.NewInt(400000000))
	if err != nil {
		t.Fatal(err)
	}

	gi, err := n.GetGroupInfo(uAddr, gIndex)
	if err != nil {
		t.Fatal(err)
	}

	uid = uuid.New()
	sig = sign(t, uAddr, uid)
	n.Approve(uid, sig, ts[0], uAddr, gi.FsAddr, big.NewInt(400000000))

	return uindex
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
	testAddKeeper(t, n, admin, gindex)
	testAddProvider(t, n, admin, gindex)

	testCreateUser(t, n, admin, gindex)

	t.Fatal("end")
}
