package auth

import (
	"boilerplate-go/internal/pkg/jwt"
	"boilerplate-go/internal/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) NewRoutes(e *gin.RouterGroup, auth jwt.IJWTAuth) {
	group := e.Group("/auth")

	group.
		POST("/login", h.Login).
		POST("/login-encrypt", h.LoginEncrypt).
		GET("/sample-data-login-encrypt", h.SampleDataLoginEncrypt).
		GET("/data", middleware.AuthMiddleware(auth), h.GetMessage)
}
