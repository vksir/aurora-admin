package app

import (
	"aurora-admin/internal/api/authapi"
	"aurora-admin/internal/api/dstapi"
	"aurora-admin/internal/api/svcapi"
	"aurora-admin/internal/plugin/dontstarve"
	"aurora-admin/internal/store"
	"aurora-admin/pkg/cache"
	"aurora-admin/pkg/cfg"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/service"
)

type application struct {
	config *cfg.Config
	state  *cache.State
	db     *store.DB
	logger *log.Logger

	dontstarvePlugin *dontstarve.Plugin

	dstApi    *dstapi.Api
	authApi   *authapi.Api
	pluginApi *svcapi.Api
}

func Run(config *cfg.Config, state *cache.State, db *store.DB) {
	log.Warn("aurora admin start...")

	app := &application{
		config: config,
		state:  state,
		db:     db,
		logger: log.DefaultLogger(),
	}

	app.dontstarvePlugin = dontstarve.NewPlugin(app.db, app.config, app.state, app.logger)

	app.authApi = authapi.NewApi(app.config)
	app.dstApi = dstapi.NewApi(app.dontstarvePlugin, app.db)
	app.pluginApi = svcapi.NewApi()

	service.Register(dontstarve.PluginName, app.dontstarvePlugin.Service())

	srv := serveHttp(app)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Info("begin exit")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "err", err)
	}
	log.Info("server stopped")
}
