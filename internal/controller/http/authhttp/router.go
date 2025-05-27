package authhttp

import (
	"aurora-admin/pkg/util/httputil"
	"github.com/gin-gonic/gin"
)

type AuthRouter struct {
}

func (r *AuthRouter) Load(g *gin.RouterGroup) {
	g.POST("/auth/login", r.login)
	g.GET("/auth/codes", r.getAccessCode)
	g.GET("/user/info", r.getUserInfo)
}

func (r *AuthRouter) login(c *gin.Context) {
	httputil.Ok(c, gin.H{
		"accessToken": "mock token",
	})
}

func (r *AuthRouter) getAccessCode(c *gin.Context) {
	httputil.Ok(c, []string{})
}

func (r *AuthRouter) getUserInfo(c *gin.Context) {
	httputil.Ok(c, gin.H{
		"roles":    "super",
		"realName": "Vkiss",
	})
}
