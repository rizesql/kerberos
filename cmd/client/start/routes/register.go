package routes

import (
	"github.com/rizesql/kerberos/cmd/client/start/platform"
	call_api "github.com/rizesql/kerberos/cmd/client/start/routes/call_api"
	"github.com/rizesql/kerberos/cmd/client/start/routes/login"
	"github.com/rizesql/kerberos/cmd/client/start/routes/ticket"
	"github.com/rizesql/kerberos/internal/server"
)

func Register(srv *server.Server, platform *platform.Platform) {
	srv.Register(login.NewHandler(platform))
	srv.Register(ticket.NewHandler(platform))
	srv.Register(call_api.NewHandler(platform))
}
