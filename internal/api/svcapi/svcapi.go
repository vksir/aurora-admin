package svcapi

import (
	"aurora-admin/internal/entity/comety"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"aurora-admin/pkg/service"
)

type Api struct{}

func NewApi() *Api {
	return &Api{}
}

func (a *Api) Load(g *gin.RouterGroup) {
	dst := g.Group("/dst/service")
	{
		dst.GET("/status", a.getStatus("dontstarve"))
		dst.POST("/start", a.control("dontstarve", "start"))
		dst.POST("/stop", a.control("dontstarve", "stop"))
		dst.POST("/restart", a.control("dontstarve", "restart"))
		dst.POST("/install", a.control("dontstarve", "install"))
		dst.POST("/uninstall", a.control("dontstarve", "uninstall"))
		dst.POST("/update", a.control("dontstarve", "update"))
	}
}

func (a *Api) getStatus(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := service.Lookup(name)
		if !ok {
			c.JSON(http.StatusInternalServerError, comety.Response{Message: fmt.Sprintf("svc %s not found", name)})
			return
		}
		c.JSON(http.StatusOK, comety.Response{Data: comety.Status{Status: svc.Status().String()}})
	}
}

func (a *Api) control(name, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := service.Lookup(name)
		if !ok {
			c.JSON(http.StatusInternalServerError, comety.Response{Message: fmt.Sprintf("svc %s not found", name)})
			return
		}
		var err error
		switch action {
		case "start":
			err = svc.Start(c.Request.Context())
		case "stop":
			err = svc.Stop(c.Request.Context())
		case "restart":
			err = svc.Restart(c.Request.Context())
		case "install":
			err = svc.Install(c.Request.Context())
		case "uninstall":
			err = svc.Uninstall(c.Request.Context())
		case "update":
			err = svc.Update(c.Request.Context())
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, comety.Response{})
	}
}
