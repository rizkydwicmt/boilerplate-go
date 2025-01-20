package auth

import (
	"boilerplate-go/internal/pkg/helper"
	"boilerplate-go/internal/pkg/jwt"
	"boilerplate-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	auth jwt.IJWTAuth
}

type IHandler interface {
	NewRoutes(e *gin.RouterGroup, auth jwt.IJWTAuth)
	Login(c *gin.Context)
	LoginEncrypt(c *gin.Context)
	SampleDataLoginEncrypt(c *gin.Context)
	GetMessage(c *gin.Context)
}

func NewHandler(auth jwt.IJWTAuth) IHandler {
	return &Handler{auth: auth}
}

func (h *Handler) Login(c *gin.Context) {
	sample := map[string]interface{}{
		"username": "username",
		"password": "password",
	}

	sample["token"], sample["expired"] = h.auth.GenerateToken(sample)

	token, err := h.auth.ValidateToken(sample["token"].(string))
	if err != nil {
		c.JSON(400, gin.H{
			"message": "error",
			"error":   err,
		})
	}

	sample["claims"] = token

	c.JSON(200, sample)
}

func (h *Handler) LoginEncrypt(c *gin.Context) {
	sample := c.MustGet("body").(map[string]interface{})
	sample["token"], sample["expired"] = h.auth.GenerateToken(sample)

	token, err := h.auth.ValidateToken(sample["token"].(string))
	if err != nil {
		c.JSON(400, gin.H{
			"message": "error",
			"error":   err,
		})
	}

	sample["claims"] = token

	c.JSON(200, sample)
}

func (h *Handler) SampleDataLoginEncrypt(c *gin.Context) {
	logger.Debug.Println("SampleDataLoginEncrypt")
	data := map[string]interface{}{
		"username": "username",
		"password": "password",
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

	c.JSON(200, gin.H{
		"message": "success",
		"data":    resp,
	})
}

func (h *Handler) GetMessage(c *gin.Context) {
	data := c.MustGet("auth").(map[string]interface{})

	c.JSON(200, gin.H{
		"message": "success",
		"data":    data,
	})
}
