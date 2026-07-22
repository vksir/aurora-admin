package authapi

import (
	"aurora-admin/internal/entity/comety"
	"aurora-admin/pkg/auth"
	"aurora-admin/pkg/cfg"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Api struct {
	config *cfg.Config
}

func NewApi(config *cfg.Config) *Api {
	return &Api{config: config}
}

func (a *Api) Load(g *gin.RouterGroup) {
	g.POST("/login", a.login)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *Api) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	token, err := auth.Login(req.Username, req.Password, a.config)
	if err != nil {
		c.JSON(http.StatusUnauthorized, comety.Response{Message: "invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: gin.H{"accessToken": token}})
}
