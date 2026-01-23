package http

import (
	"github.com/rizesql/kerberos/internal/kdc"
	"github.com/rizesql/kerberos/internal/kdc/as"
	"github.com/rizesql/kerberos/internal/server"
)

func Register(srv *server.Server, platform *kdc.Platform, cfg as.Config) {
	srv.Register(as.NewHandler(platform, cfg),
		server.WithLogging(platform.Logger),
	)
}
