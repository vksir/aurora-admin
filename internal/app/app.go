package app

import (
	"aurora-admin/ent"
	"aurora-admin/internal/api/authapi"
	"aurora-admin/internal/api/dstapi"
	"aurora-admin/internal/api/svcapi"
	"aurora-admin/internal/plugin/dontstarve"
	"aurora-admin/pkg/database"
	"context"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/service"
	"os"
	"os/signal"
	"syscall"
)

type application struct {
	ctx    context.Context
	db     *ent.Client
	logger *log.Logger

	dontstarvePlugin *dontstarve.Plugin

	dstApi    *dstapi.Api
	authApi   *authapi.Api
	pluginApi *svcapi.Api
}

func Run() {
	log.Warn("aurora admin start...")
	ctx, cancel := context.WithCancel(context.Background())

	app := &application{
		ctx:    ctx,
		db:     database.G,
		logger: log.DefaultLogger(),
	}

	app.dontstarvePlugin = dontstarve.NewPlugin(app.db, app.logger)

	app.authApi = authapi.NewApi(app.db, app.logger)
	app.dstApi = dstapi.NewApi(app.dontstarvePlugin, app.db)
	app.pluginApi = svcapi.NewApi(app.db, app.logger)

	service.Register(dontstarve.PluginName, app.dontstarvePlugin.Service())

	go func() {
		serveHttp(app)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Info("begin exit")
	cancel()
}
