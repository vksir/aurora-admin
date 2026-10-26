package app

import (
	"aurora-admin/assets"
	"aurora-admin/docs"
	_ "aurora-admin/docs"
	"aurora-admin/pkg/auth"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/middleware"
	"aurora-admin/pkg/util/errutil"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func loadStaticRouters(e *gin.Engine, apiDocHost string) {
	log.InfoF("static page: http://%s", apiDocHost)
	distFS, err := fs.Sub(assets.StaticFS, "web/dist")
	errutil.Fatal(err)

	distHttpFS := http.FS(distFS)
	e.StaticFS("/assets", distHttpFS)

	e.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		file, err := distHttpFS.Open(strings.TrimPrefix(reqPath, "/"))
		if err == nil {
			defer log.Close(file)
			http.FileServer(distHttpFS).ServeHTTP(c.Writer, c.Request)
			return
		}

		if strings.HasPrefix(reqPath, "/api") {
			return
		}

		// 静态资源缺失时返回真正的 404，避免 SPA fallback 把 index.html
		// (text/html) 当作 js/css 返回，触发浏览器的严格 MIME 校验报错
		if path.Ext(reqPath) != "" {
			c.String(http.StatusNotFound, "Not Found")
			return
		}

		indexFile, err := distHttpFS.Open("index.html")
		if err != nil {
			c.String(http.StatusNotFound, "Index Not Found")
			return
		}
		defer log.Close(indexFile)
		stat, _ := indexFile.Stat()
		// index.html 引用的是带 hash 的构建产物，一旦被浏览器缓存，
		// 重新发布后旧 index 会指向已删除的 chunk，必须强制重新验证
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeContent(c.Writer, c.Request, "index.html", stat.ModTime(), indexFile)
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
		Addr:              listen,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second, // 防慢速头部攻击占用连接
		IdleTimeout:       120 * time.Second,
		// 注意：不设 ReadTimeout/WriteTimeout，存档上传/下载是长耗时流式传输
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server listen failed", "err", err)
		}
	}()
	return srv
}
