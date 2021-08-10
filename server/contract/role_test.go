package contract

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/memoio/go-settlement/utils"
)

func TestPledge(t *testing.T) {
	adminkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	adminAddr := utils.ToAddress(adminkey.PubKey)
	t.Log("adminAddr: ", adminAddr.String())

	et, err := NewErcToken(adminAddr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("et token adminAddr: ", et.GetOwnerAddress(), "local: ", et.GetContractAddress(), "value: ", getBalance(et.GetContractAddress(), adminAddr))

	userkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	userAddr := utils.ToAddress(userkey.PubKey)
	t.Log("userAddr: ", userAddr.String())
	sendBalance(et.GetContractAddress(), adminAddr, userAddr, big.NewInt(100000000))

	t.Log(getBalance(et.GetContractAddress(), adminAddr).String())
	t.Log(getBalance(et.GetContractAddress(), userAddr).String())

	uet, err := NewErcToken(userAddr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("uet token adminAddr: ", uet.GetOwnerAddress(), "local: ", uet.GetContractAddress().String(), "value: ", getBalance(uet.GetContractAddress(), uet.GetOwnerAddress()))

	err = sendBalance(uet.GetContractAddress(), userAddr, userAddr, big.NewInt(100000000))
	if err != nil {
		t.Fatal(err)
	}

	pToken := et.GetContractAddress()

	pm := NewPledgeMgr(adminAddr, pToken)
	t.Log("pm local addr: ", pm.GetContractAddress(), pToken)

	pm.AddToken(uet.GetContractAddress())

	userkey1, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	userAddr1 := utils.ToAddress(userkey1.PubKey)

	err = sendBalance(et.GetContractAddress(), et.GetOwnerAddress(), userAddr1, big.NewInt(101000000))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(getBalance(et.GetContractAddress(), userAddr1))

	err = pm.Stake(userAddr1, big.NewInt(100000000))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("after stack: user and pm.local has primary:", getBalance(pToken, userAddr1), getBalance(pToken, pm.local))

	sendBalance(uet.GetContractAddress(), uet.GetOwnerAddress(), pm.local, big.NewInt(432000000))
	t.Log("send profit: user and pm.local has primary:", getBalance(uet.GetContractAddress(), userAddr1), getBalance(uet.GetContractAddress(), pm.local))

	err = pm.Stake(userAddr1, big.NewInt(1000))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("after stack2: user and pm.local has primary:", getBalance(pToken, userAddr1), getBalance(pToken, pm.local))
	t.Log("after stack2: user and pm.local has:", getBalance(uet.GetContractAddress(), userAddr1), getBalance(uet.GetContractAddress(), pm.local))

	err = pm.Withdraw(userAddr1, true)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("after withdraw: user and pm.local has primary:", getBalance(pToken, userAddr1), getBalance(pToken, pm.local))
	t.Log("after withdraw: user and pm.local has:", getBalance(uet.GetContractAddress(), userAddr1), getBalance(uet.GetContractAddress(), pm.local))

	t.Fatal("end")
}
