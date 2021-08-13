package common

import (
	"context"

	"github.com/gbrlsnchs/jwt/v3"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc/auth"
)

type APIToken []byte
type APIAlg jwt.HMACSHA

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
