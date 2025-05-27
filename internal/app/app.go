package app

import (
	"aurora-admin/internal/controller/api/svcapi"
	"aurora-admin/internal/controller/http"
	"aurora-admin/internal/plugin/donstarve"
	"aurora-admin/pkg/database"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/vksir/vkiss-lib/pkg/log"
)

func Run(ctx context.Context, listen string, logger *log.Logger) {
	dst := donstarve.NewPlugin(database.DB, logger)

	e := gin.Default()
	services := []svcapi.Interface{dst}
	http.LoadRouter(e, services)

	log.Info("serving", "listen", listen)
	err := e.Run(listen)
	if err != nil {
		log.Error("gin run failed", "err", err)
	}
}
