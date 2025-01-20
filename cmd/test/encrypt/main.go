package main

import (
	"boilerplate-go/internal/handler/auth"
	"boilerplate-go/internal/pkg/helper"
	"boilerplate-go/internal/pkg/jwt"
	"boilerplate-go/internal/pkg/logger"
	"boilerplate-go/internal/pkg/middleware"
	"boilerplate-go/internal/pkg/redis"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Check struct {
	Level   string `json:"level"`
	GroupID string `json:"group_id"`
	UserID  int    `json:"userid"`
	Tenant  string `json:"tenant_id"`
}

type Check2 struct {
	Level   string `json:"level"`
	GroupID string `json:"group_id"`
	UserID  int    `json:"userid"`
	Tenant  string `json:"tenant_id"`
	Asd     string `json:"asd"`
}

func main() {
	logger.Setup()
	err := godotenv.Load()
	if err != nil {
		panic("Error reading .env file")
	}
	ctx := context.Background()

	rds, err := setupRedis(ctx)
	if err != nil {
		logger.Error.Println("Error connecting to redis")
		panic(err)
	}
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.RequestInit())
	r.Use(middleware.ResponseInit())

	jwtOpts := jwt.DefaultOptions("bismillah")
	jwtOpts.TokenExpiredTime = 60 * time.Second
	jwtAuth := jwt.New(jwtOpts)

	r.POST("/encrypt", encryptHandler)

	r.Use(middleware.EncryptMiddleware(rds))

	r.POST("/post", postHandler)

	handler := auth.NewHandler(jwtAuth)
	handler.NewRoutes(r.Group("/api"), jwtAuth)
	err = r.Run(":8003")
	if err != nil {
		return
	}
}

func setupRedis(ctx context.Context) (redis.IRedis, error) {
	return redis.Setup(ctx, &redis.Config{
		Host:     "localhost",
		Port:     6379,
		PoolSize: 10,
	})
}

func postHandler(c *gin.Context) {
	body1 := c.MustGet("body")
	var payload2 Check2
	err := helper.JSONToStruct[Check2](body1, &payload2)
	if err != nil {
		c.JSON(500, gin.H{
			"message": "Error converting body to struct",
		})
		return
	}
	var payload Check
	if err := c.ShouldBind(&payload); err != nil {
		c.JSON(400, gin.H{
			"message": "Error binding payload",
		})
		return
	}
	c.JSON(200, gin.H{
		"message":  "Hello World",
		"body":     body1,
		"payload":  payload,
		"payload2": payload2,
	})
}

func encryptHandler(c *gin.Context) {
	// data := Check{
	//	GroupID: "123a",
	//	Level:   "admin",
	//	Tenant:  "tenant",
	//	UserID:  1,
	//}

	data := gin.H{
		"userid":    1,
		"group_id":  "123a",
		"level":     "admin",
		"tenant_id": "tenant",
	}

	strData, err := helper.JSONToString(data)
	logger.Debug.Println(strData)
	if err != nil {
		c.JSON(500, gin.H{
			"message": "Error converting data to string",
		})
		return
	}

	resp, err := helper.EncryptAESCBC(strData)
	if err != nil {
		c.JSON(500, gin.H{
			"message": "Error encrypting data",
		})
		return
	}

	resp2, err := helper.DecryptAESCBC(resp)
	if err != nil {
		c.JSON(500, gin.H{
			"message": "Error decrypting data",
			"resp":    resp,
		})
		return
	}

	c.JSON(200, gin.H{
		"message":   "Data encrypted",
		"data":      resp,
		"decrypted": resp2,
	})
	return
}
