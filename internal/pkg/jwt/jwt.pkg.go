package jwt

import (
	"boilerplate-go/internal/pkg/helper"
	"boilerplate-go/internal/pkg/redis"
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	UserDataKey = "user_data"
)

// Auth struct
type Auth struct {
	TokenExpiredTime time.Duration
	TokenSecretKey   string
	SigningMethod    string
	SaveMethod       SaveMethodJWTEnum
	Redis            redis.IRedis
}

type IJWTAuth interface {
	GenerateToken(data map[string]interface{}) (string, *time.Time)
	ValidateToken(jwtToken string) (map[string]interface{}, error)
}

// New Auth object
func New(rds redis.IRedis, opt *Options) IJWTAuth {
	return &Auth{
		TokenExpiredTime: opt.TokenExpiredTime,
		TokenSecretKey:   opt.TokenSecretKey,
		SigningMethod:    opt.SigningMethod,
		SaveMethod:       opt.SaveMethod,
		Redis:            rds,
	}
}

// GenerateToken generate jwt token
func (a *Auth) GenerateToken(data map[string]interface{}) (string, *time.Time) {
	exp := time.Now().Add(a.TokenExpiredTime)
	sessionID, err := helper.GenerateID()
	if err != nil {
		return "", nil
	}

	tokenContent := jwt.MapClaims{}
	for key, value := range data {
		if key == "aud" || key == "exp" || key == "iat" || key == "iss" || key == "nbf" {
			continue
		}
		tokenContent[key] = value
	}

	if a.TokenExpiredTime > 0 && a.SaveMethod == JWT {
		tokenContent["exp"] = exp.Unix()
	}

	tokenContent["iat"] = time.Now().Unix()
	tokenContent["session_id"] = sessionID

	jwtToken := jwt.NewWithClaims(
		jwt.GetSigningMethod(a.SigningMethod),
		tokenContent,
	)
	token, err := jwtToken.SignedString([]byte(a.TokenSecretKey))
	if err != nil {
		return "", nil
	}

	if a.SaveMethod == REDIS {
		if id, ok := data["id"]; ok {
			strID := fmt.Sprintf("%v", id)
			if strID != "" {
				err = a.Redis.Set(os.Getenv("APP_TENANT")+":"+strID, sessionID, a.TokenExpiredTime)
				if err != nil {
					return "", nil
				}
			} else {
				return "", nil
			}
		} else {
			return "", nil
		}
	}

	return token, &exp
}

// ValidateToken validate jwt token
func (a *Auth) ValidateToken(jwtToken string) (map[string]interface{}, error) {
	tokenData := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(jwtToken, tokenData, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.TokenSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrInvalidKey
	}

	if a.SaveMethod == REDIS {
		if id, ok := tokenData["id"]; ok {
			strID := fmt.Sprintf("%v", id)
			if strID != "" {
				sessionID, er := a.Redis.Get(os.Getenv("APP_TENANT") + ":" + strID)
				if er != nil {
					return nil, jwt.ErrInvalidKey
				}
				if sessionID == "" {
					return nil, jwt.ErrInvalidKey
				}
				if sessionID != fmt.Sprintf("\"%s\"", tokenData["session_id"]) {
					return nil, jwt.ErrInvalidKey
				}
			} else {
				return nil, jwt.ErrInvalidKey
			}
		} else {
			return nil, fmt.Errorf("id is required")
		}
	}

	return tokenData, nil
}
