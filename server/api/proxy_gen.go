package api

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/google/uuid"
	"github.com/memoio/go-settlement/utils"
)

// common API permissions constraints
type CommonStruct struct {
	Internal struct {
		AuthVerify func(ctx context.Context, token string) ([]auth.Permission, error) `perm:"read"`
		AuthNew    func(ctx context.Context, erms []auth.Permission) ([]byte, error)  `perm:"admin"`
	}
}

type FullNodeStruct struct {
	CommonStruct

	Internal struct {
		CreateErcToken func(uuid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error) `perm:"admin"`
	}
}

func (s *CommonStruct) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	return s.Internal.AuthVerify(ctx, token)
}

func (s *CommonStruct) AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error) {
	return s.Internal.AuthNew(ctx, perms)
}

func (s *FullNodeStruct) CreateErcToken(uuid uuid.UUID, sig []byte, caller utils.Address) (utils.Address, error) {
	return s.Internal.CreateErcToken(uuid, sig, caller)
}
