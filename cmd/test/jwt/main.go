package main

import (
	"boilerplate-go/internal/handler/auth"
	"boilerplate-go/internal/pkg/jwt"
	"boilerplate-go/internal/pkg/logger"
	"boilerplate-go/internal/pkg/middleware"
	"github.com/gin-gonic/gin"
	"time"
)

func main() {
	logger.Setup()
	jwtOpts := jwt.DefaultOptions("bismillah")
	jwtOpts.TokenExpiredTime = 60 * time.Second
	jwtAuth := jwt.New(jwtOpts)
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.RequestInit())
	r.Use(middleware.ResponseInit())

	handler := auth.NewHandler(jwtAuth)
	handler.NewRoutes(r.Group("/api"), jwtAuth)

	err := r.Run(":8001")
	if err != nil {
		return
	}
}
