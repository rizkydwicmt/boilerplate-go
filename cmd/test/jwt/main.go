package main

import (
	"boilerplate-go/internal/handler/auth"
	"boilerplate-go/internal/pkg/jwt"
	"boilerplate-go/internal/pkg/logger"
	"boilerplate-go/internal/pkg/middleware"
	"boilerplate-go/internal/pkg/redis"
	"context"
	"github.com/gin-gonic/gin"
	"time"
)

func main() {
	logger.Setup()
	ctx := context.Background()
	jwtOpts := jwt.DefaultOptions("bismillah")
	jwtOpts.TokenExpiredTime = 60 * time.Second
	jwtOpts.SaveMethod = jwt.REDIS

	//jwtAuth := jwt.New(nil, jwtOpts)
	rds, err := redis.Setup(ctx, &redis.Config{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		PoolSize: 10,
	})
	if err != nil {
		logger.Error.Println("Error connecting to redis")
		panic(err)
	}
	jwtAuth := jwt.New(rds, jwtOpts)
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.RequestInit())
	r.Use(middleware.ResponseInit())

	handler := auth.NewHandler(jwtAuth)
	handler.NewRoutes(r.Group("/api"), jwtAuth)

	err = r.Run(":8001")
	if err != nil {
		return
	}
}
