package middleware

import (
	_type "boilerplate-go/internal/common/type"
	"boilerplate-go/internal/pkg/helper"
	"boilerplate-go/internal/pkg/jwt"
	"boilerplate-go/internal/pkg/logger"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func AuthMiddleware(auth jwt.IJWTAuth) gin.HandlerFunc {
	return func(c *gin.Context) {
		send := c.MustGet("send").(func(r *_type.Response))
		token := c.GetHeader("Authorization")
		if token == "" {
			send(helper.ParseResponse(&_type.Response{Code: http.StatusBadRequest, Message: "token not found"}))
			return
		}

		logger.Debug.Println("token", token)
		parts := strings.Split(token, " ")
		if len(parts) < 2 {
			send(helper.ParseResponse(&_type.Response{Code: http.StatusBadRequest, Message: "invalid token format", Error: errors.New("invalid token format")}))
			return
		}
		claims, err := auth.ValidateToken(parts[1])
		if err != nil {
			send(helper.ParseResponse(&_type.Response{Code: http.StatusBadRequest, Message: "invalid token", Error: err}))
			return
		}

		c.Set("auth", claims)
		c.Next()
	}
}
