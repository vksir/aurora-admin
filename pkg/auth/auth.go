package auth

import (
	"net/http"
	"strings"
	"time"

	"aurora-admin/pkg/cfg"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const tokenExpiry = 24 * time.Hour

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func Login(username, password string, config *cfg.Config) (string, error) {
	if username != config.Username || password != config.Password {
		return "", jwt.ErrSignatureInvalid
	}
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Password))
}

func Middleware(config *cfg.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "missing authorization header"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			return []byte(config.Password), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
			c.Abort()
			return
		}
		c.Set("username", claims.Username)
		c.Next()
	}
}
