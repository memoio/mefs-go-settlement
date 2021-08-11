package contract

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/memoio/go-settlement/utils"
)

func TestErc(t *testing.T) {
	adminkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	adminAddr := utils.ToAddress(adminkey.PubKey)

	et, err := NewErcToken(adminAddr)
	if err != nil {
		t.Fatal(err)
	}

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

	t.Log(getBalance(et.GetContractAddress(), userAddr), getBalance(et.GetContractAddress(), userAddr2))

	//t.Fatal("end")
}

func TestRole(t *testing.T) {
	adminkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	adminAddr := utils.ToAddress(adminkey.PubKey)
	et, err := NewErcToken(adminAddr)
	if err != nil {
		t.Fatal(err)
	}

	rm := NewRoleMgr(adminAddr, et.GetContractAddress(), big.NewInt(123450), big.NewInt(12345))

	userkey, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	userAddr := utils.ToAddress(userkey.PubKey)
	err = sendBalance(et.GetContractAddress(), adminAddr, userAddr, big.NewInt(12345))
	if err != nil {
		t.Fatal(err)
	}

	err = rm.Register(userAddr, userAddr, nil)
	if err != nil {
		t.Fatal(err)
	}

	rInfo, err := rm.GetInfo(userAddr, userAddr)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("info:", rInfo)

	approve(et.GetContractAddress(), userAddr, rm.GetContractAddress(), big.NewInt(12345))

	err = rm.Pledge(userAddr, 0, big.NewInt(12345), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = rm.RegisterKeeper(userAddr, 0, nil, nil)
	if err == nil {
		t.Fatal("should fail")
	}

	err = rm.RegisterProvider(userAddr, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	rInfo2, err := rm.GetInfo(userAddr, userAddr)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("info2:", rInfo2)

	for i := 0; i < 4; i++ {
		userkey, err := utils.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}

		userAddr := utils.ToAddress(userkey.PubKey)
		err = sendBalance(et.GetContractAddress(), adminAddr, userAddr, big.NewInt(123450))
		if err != nil {
			t.Fatal(err)
		}

		err = rm.Register(userAddr, userAddr, nil)
		if err != nil {
			t.Fatal(err)
		}

		approve(et.GetContractAddress(), userAddr, rm.GetContractAddress(), big.NewInt(123450))

		rInfo, err := rm.GetInfo(userAddr, userAddr)
		if err != nil {
			t.Fatal(err)
		}

		err = rm.Pledge(userAddr, rInfo.index, big.NewInt(123450), nil)
		if err != nil {
			t.Fatal(err)
		}
		err = rm.RegisterKeeper(userAddr, rInfo.index, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		rInfo, err = rm.GetInfo(userAddr, userAddr)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("info:", rInfo)
	}

	err = rm.CreateGroup(adminAddr, []uint64{1, 2}, 3, nil)
	if err != nil {
		t.Fatal(err)
	}

	gi, err := rm.GetGroupInfoByIndex(adminAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ginfo:", gi)

	err = rm.AddKeeperToGroup(adminAddr, 3, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	gi, err = rm.GetGroupInfoByIndex(adminAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ginfo:", gi)

	err = rm.AddProviderToGroup(userAddr, 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	ks, err := rm.GetKeepersByIndex(adminAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ks)

	ps, err := rm.GetProvidersByIndex(adminAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ps)

	t.Fatal("end")
}
