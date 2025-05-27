package app

import (
	"aurora-admin/assets"
	"aurora-admin/docs"
	_ "aurora-admin/docs"
	"aurora-admin/pkg/cfg"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/middleware"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
)

func loadStaticRouters(e *gin.Engine) {
	log.InfoF("static page: http://%s", cfg.G.ApiDocHost)
	distFS, err := fs.Sub(assets.StaticFS, "web/dist")
	errutil.Check(err)

	distHttpFS := http.FS(distFS)
	e.StaticFS("/assets", distHttpFS)

	e.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		file, err := distHttpFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			defer log.Close(file)
			http.FileServer(distHttpFS).ServeHTTP(c.Writer, c.Request)
			return
		}

		if !strings.HasPrefix(path, "/api") {
			indexFile, err := distHttpFS.Open("index.html")
			if err != nil {
				c.String(http.StatusNotFound, "Index Not Found")
				return
			}
			defer log.Close(indexFile)
			stat, _ := indexFile.Stat()
			http.ServeContent(c.Writer, c.Request, "index.html", stat.ModTime(), indexFile)
		}
	})
}

func loadSwaggerRouters(g *gin.RouterGroup) {
	apiDocHost := cfg.G.ApiDocHost
	if apiDocHost != "" {
		docs.SwaggerInfo.Host = apiDocHost
	}
	log.InfoF("api doc: http://%s/api/docs", apiDocHost)
	g.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/api/docs/index.html")
	})
	g.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// Run
// Swagger spec:
// @title           Aurora Admin
// @description 	Aurora Admin API
// @version     	1.0
// @host        	0.0.0.0:5800
// @BasePath    	/
func serveHttp(a *application) {
	e := gin.New()
	e.Use(gin.Recovery())
	e.Use(middleware.Logger(middleware.LoggerConfig{}, a.logger))
	loadStaticRouters(e)

	g := e.Group("/api")
	loadSwaggerRouters(g)
	a.authApi.Load(g)
	a.dstApi.Load(g)
	a.pluginApi.Load(g)

	listen := cfg.G.Listen
	log.InfoF("listen: %s", listen)
	err := e.Run(listen)
	errutil.Check(err)
}
