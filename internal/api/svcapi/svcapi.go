package svcapi

import (
	"aurora-admin/ent"
	"aurora-admin/internal/entity/comety"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/service"
)

type Api struct {
}

func NewApi(db *ent.Client, logger *log.Logger) *Api {
	return &Api{}
}

func (a *Api) Load(g *gin.RouterGroup) {
	g.GET("/:service/status", a.GetStatus)
	g.POST("/:service/:action", a.CreateControl)
}

// GetStatus
// @Summary			获取状态
// @Tags			service_status
// @Accept			json
// @Produce			json
// @Param			service path string true "service" Enums(dontstarve)
// @Success			200 {object} comety.Status{status=string} "status: Inactive, Active, Starting, Stopping, WaitingActive, Abnormal"
// @Failure			500 {object} any
// @Router			/api/{service}/status [get]
func (a *Api) GetStatus(c *gin.Context) {
	name := c.Param("service")
	svc, ok := service.Lookup(name)
	if !ok {
		msg := fmt.Sprintf("svc %s not found", name)
		c.JSON(http.StatusInternalServerError, comety.Response{Message: msg})
		return
	}

	status := comety.Status{Status: svc.Status().String()}
	c.JSON(http.StatusOK, comety.Response{Data: status})
}

// CreateControl
// @Summary			服务控制
// @Tags			service_control
// @Accept			json
// @Produce			json
// @Param			service path string true "service" Enums(dontstarve)
// @Param			action path string true "action" Enums(start, stop, restart, install, uninstall, update)
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/{service}/{action} [post]
func (a *Api) CreateControl(c *gin.Context) {
	name := c.Param("service")
	action := c.Param("action")
	svc, ok := service.Lookup(name)
	if !ok {
		msg := fmt.Sprintf("plugin %s not found", name)
		c.JSON(http.StatusInternalServerError, comety.Response{Message: msg})
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
	default:
		err = fmt.Errorf("invalid action: %s", action)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{})
}
