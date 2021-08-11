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

	t.Fatal("end")
}
