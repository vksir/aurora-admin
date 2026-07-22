package authapi

import (
	"aurora-admin/internal/entity/comety"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Api struct {
}

func NewApi() *Api {
	return &Api{}
}

func (a *Api) Load(g *gin.RouterGroup) {
	g.POST("/auth/login", a.login)
	g.GET("/auth/codes", a.getAccessCode)
	g.GET("/user/info", a.getUserInfo)
}

func (a *Api) login(c *gin.Context) {
	// Simple JSON response
	c.JSON(http.StatusOK, comety.Response{Data: gin.H{
		"accessToken": "mock token",
	}})
}

func (a *Api) getAccessCode(c *gin.Context) {
	c.JSON(http.StatusOK, comety.Response{Data: []string{}})
}

func (a *Api) getUserInfo(c *gin.Context) {
	c.JSON(http.StatusOK, comety.Response{Data: gin.H{
		"roles":    "super",
		"realName": "Vkiss",
	}})
}
