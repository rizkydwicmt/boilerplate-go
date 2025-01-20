package jwt

import (
	"boilerplate-go/internal/pkg/helper"
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
}

type IJWTAuth interface {
	GenerateToken(data map[string]interface{}) (string, *time.Time)
	ValidateToken(jwtToken string) (map[string]interface{}, error)
}

// New Auth object
func New(opt *Options) IJWTAuth {
	return &Auth{
		TokenExpiredTime: opt.TokenExpiredTime,
		TokenSecretKey:   opt.TokenSecretKey,
		SigningMethod:    opt.SigningMethod,
	}
}

// GenerateToken generate jwt token
func (a *Auth) GenerateToken(data map[string]interface{}) (string, *time.Time) {
	exp := time.Now().Add(a.TokenExpiredTime)

	tokenContent := jwt.MapClaims{}
	for key, value := range data {
		if key == "aud" || key == "exp" || key == "iat" || key == "iss" || key == "nbf" {
			continue
		}
		tokenContent[key] = value
	}

	if a.TokenExpiredTime > 0 {
		tokenContent["exp"] = exp.Unix()
	}

	tokenContent["iat"] = time.Now().Unix()

	jwtToken := jwt.NewWithClaims(
		jwt.GetSigningMethod(a.SigningMethod),
		tokenContent,
	)
	token, err := jwtToken.SignedString([]byte(a.TokenSecretKey))
	if err != nil {
		return "", nil
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

	var payload map[string]interface{}
	if err := helper.JSONToStruct[map[string]interface{}](tokenData, &payload); err != nil {
		return nil, err
	}

	return payload, nil
}
