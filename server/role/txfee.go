package role

import (
	"math/big"

	"github.com/memoio/go-settlement/utils"
)

type txFee struct {
	tAddr  utils.Address
	amount *big.Int
}
