package api

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/google/uuid"
	"github.com/memoio/go-settlement/utils"
	"golang.org/x/xerrors"
)

type FullNode interface {
	Common

	CreateErcToken(uuid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error)
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
