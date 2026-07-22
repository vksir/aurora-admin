package app

import (
	"aurora-admin/assets"
	"aurora-admin/docs"
	_ "aurora-admin/docs"
	"aurora-admin/pkg/auth"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"aurora-admin/pkg/log"
	"aurora-admin/pkg/middleware"
	"aurora-admin/pkg/util/errutil"
)

func loadStaticRouters(e *gin.Engine, apiDocHost string) {
	log.InfoF("static page: http://%s", apiDocHost)
	distFS, err := fs.Sub(assets.StaticFS, "web/dist")
	errutil.Fatal(err)

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

func loadSwaggerRouters(g *gin.RouterGroup, apiDocHost string) {
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
// @title           Aurora Admin
// @description 	Aurora Admin API
// @version     	1.0
// @host        	0.0.0.0:5800
// @BasePath    	/api/v1
func serveHttp(a *application) *http.Server {
	e := gin.New()
	e.Use(gin.Recovery())
	e.Use(middleware.Logger(middleware.LoggerConfig{}, a.logger))
	loadStaticRouters(e, a.config.ApiDocHost)

	// Public routes (no auth)
	g := e.Group("/api")
	loadSwaggerRouters(g, a.config.ApiDocHost)

	v1 := g.Group("/v1")
	{
		a.authApi.Load(v1)
	}

	// Protected routes (JWT required)
	protected := v1.Group("")
	protected.Use(auth.Middleware(a.config))
	{
		a.dstApi.Load(protected)
		a.pluginApi.Load(protected)
	}

	listen := a.config.Listen
	log.InfoF("listen: %s", listen)

	srv := &http.Server{
		Addr:    listen,
		Handler: e,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server listen failed", "err", err)
		}
	}()
	return srv
}
