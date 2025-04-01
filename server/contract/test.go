package contract

import (
	"crypto/rand"
	"github.com/memoio/go-settlement/utils"
)

func createAndRegisterKeeper() {
	_, err := utils.GenerateKey(rand.Reader)
	if err != nil {
		return
	}
}
