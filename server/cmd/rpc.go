package main

import (
	"net/http"
	_ "net/http/pprof"

	jsonrpc "github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gorilla/mux"

	"github.com/memoio/go-settlement/server/api"
	"github.com/memoio/go-settlement/server/impl"
	"golang.org/x/xerrors"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

func ServeRPC(h http.Handler, id string, addr ma.Multiaddr) (impl.StopFunc, error) {
	// Start listening to the addr; if invalid or occupied, we will fail early.
	lst, err := manet.Listen(addr)
	if err != nil {
		return nil, xerrors.Errorf("could not listen: %w", err)
	}

	// Instantiate the server and start listening.
	srv := &http.Server{
		Handler: h,
	}

	go func() {
		err = srv.Serve(manet.NetListener(lst))
		if err != http.ErrServerClosed {
			log.Warnf("rpc server failed: %s", err)
		}
	}()

	return srv.Shutdown, err
}

func FullNodeHandler(a api.FullNode, permissioned bool) (http.Handler, error) {
	m := mux.NewRouter()

	if permissioned {
		a = api.PermissionedFullAPI(a)
	}

	rpcServer := jsonrpc.NewServer()
	rpcServer.Register("Memoriae", a)

	m.Handle("/rpc/v0", rpcServer)

	if !permissioned {
		return m, nil
	}

	ah := &auth.Handler{
		Verify: a.AuthVerify,
		Next:   m.ServeHTTP,
	}
	return ah, nil
}
