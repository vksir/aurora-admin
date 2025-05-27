package http

import (
	"aurora-admin/docs"
	_ "aurora-admin/docs"
	"aurora-admin/internal/controller/api/svcapi"
	"aurora-admin/internal/controller/http/authhttp"
	"aurora-admin/internal/controller/http/svchttp"
	"aurora-admin/pkg/cfg"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/vksir/vkiss-lib/pkg/log"
	"net/http"
)

func loadSwaggerRouters(g *gin.RouterGroup) {
	if cfg.C.ApiDocHost == "" {
		return
	}
	log.InfoF("api doc host: %s", cfg.C.ApiDocHost)
	docs.SwaggerInfo.Host = cfg.C.ApiDocHost
	g.GET("/docs", redirectSwagger)
	g.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func redirectSwagger(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/api/docs/index.html")
}

// LoadRouter
// Swagger spec:
// @title           Aurora Admin
// @description 	Aurora Admin API
// @version     	1.0
// @host        	0.0.0.0:5800
// @BasePath    	/
func LoadRouter(handler *gin.Engine, services []svcapi.Interface) {
	g := handler.Group("/api")

	loadSwaggerRouters(g)

	for _, svc := range services {
		r := svchttp.NewSvcRouter(svc)
		r.Load(g)
	}

	r := authhttp.AuthRouter{}
	r.Load(g)
}
