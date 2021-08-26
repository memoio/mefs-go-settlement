package api

import (
	"context"
	"math/big"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/google/uuid"
	"github.com/memoio/go-settlement/server/contract"
	"github.com/memoio/go-settlement/utils"
	"golang.org/x/xerrors"
)

type FullNode interface {
	Common

	CreateErcToken(uuid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error)
	TotalSupply(tAddr, caller utils.Address) *big.Int
	BalanceOf(tAddr, caller, tokenOwner utils.Address) *big.Int
	Allowance(tAddr, caller, tokenOwner, spender utils.Address) *big.Int
	Approve(uid uuid.UUID, sig []byte, tAddr, caller, spender utils.Address, value *big.Int) error
	Transfer(uid uuid.UUID, sig []byte, tAddr, caller, to utils.Address, value *big.Int) error
	TransferFrom(uid uuid.UUID, sig []byte, tAddr, caller, from, to utils.Address, value *big.Int) error
	MintToken(uid uuid.UUID, sig []byte, tAddr, caller, target utils.Address, mintedAmount *big.Int) error
	Burn(uid uuid.UUID, sig []byte, tAddr, caller utils.Address, burnAmount *big.Int) error
	AirDrop(uid uuid.UUID, sig []byte, tAddr, caller utils.Address, addrs []utils.Address, money *big.Int) error

	CreateRoleMgr(uid uuid.UUID, sig []byte, caller, founder, token utils.Address) (utils.Address, error)
	Register(uid uuid.UUID, sig []byte, caller, addr utils.Address, sign []byte) error
	RegisterToken(uid uuid.UUID, sig []byte, caller, taddr utils.Address, asign []byte) error
	RegisterKeeper(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, blsKey, signature []byte) error
	RegisterProvider(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, signature []byte) error
	RegisterUser(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, token uint32, blsKey, usign []byte) error
	Pledge(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, money *big.Int, signature []byte) error
	Withdraw(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, money *big.Int, signature []byte) error
	CreateGroup(uid uuid.UUID, sig []byte, caller utils.Address, inds []uint64, level uint16, asign []byte) error
	AddKeeperToGroup(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, ksign, asign []byte) error
	AddProviderToGroup(uid uuid.UUID, sig []byte, caller utils.Address, index, gIndex uint64, psign []byte) error
	Recharge(uid uuid.UUID, sig []byte, caller utils.Address, user uint64, tokenIndex uint32, money *big.Int, sign []byte) error
	ProWithdraw(uid uuid.UUID, sig []byte, caller utils.Address, proIndex uint64, tokenIndex uint32, pay, lost *big.Int, ksigns [][]byte) error
	WithdrawFromFs(uid uuid.UUID, sig []byte, caller utils.Address, index uint64, tokenIndex uint32, amount *big.Int, sign []byte) error
	AddOrder(uid uuid.UUID, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error
	SubOrder(uid uuid.UUID, sig []byte, caller utils.Address, user, proIndex, start, end, size, nonce uint64, tokenIndex uint32, sprice *big.Int, usign, psign []byte, ksigns [][]byte) error

	GetIndex(caller, addr utils.Address) (uint64, error)
	GetInfo(caller utils.Address, index uint64) (*contract.BaseInfo, utils.Address, error)
	GetTokenIndex(caller, taddr utils.Address) (uint32, error)
	GetTokenAddress(caller utils.Address, index uint32) (utils.Address, error)
	GetGroupInfo(caller utils.Address, gindex uint64) (*contract.GroupInfo, error)
	GetBalance(caller utils.Address, index uint64) ([]*big.Int, error)
	GetBalanceInFs(caller utils.Address, index uint64, tIndex uint32) (*big.Int, *big.Int, *big.Int, error)
	GetPledgeAddress(caller utils.Address) utils.Address
	GetPledge(caller utils.Address) (*big.Int, *big.Int, []*big.Int)
	GetAllTokens(caller utils.Address) []utils.Address
	GetAllAddrs(caller utils.Address) []utils.Address
	GetAllGroups(caller utils.Address) []*contract.GroupInfo
	GetFoundation(caller utils.Address) utils.Address
}

type APIAlg jwt.HMACSHA
type APIToken []byte

type CommonAPI struct {
	APISecret *APIAlg
}

type jwtPayload struct {
	Allow []auth.Permission
}

func (a *CommonAPI) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	var payload jwtPayload
	if _, err := jwt.Verify([]byte(token), (*jwt.HMACSHA)(a.APISecret), &payload); err != nil {
		return nil, xerrors.Errorf("JWT Verification failed: %w", err)
	}

	return payload.Allow, nil
}

func (a *CommonAPI) AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error) {
	p := jwtPayload{
		Allow: perms, // TODO: consider checking validity
	}

	return jwt.Sign(&p, (*jwt.HMACSHA)(a.APISecret))
}
