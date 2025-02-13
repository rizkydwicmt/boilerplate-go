package jwt

import (
	"time"
)

const (
	DefaultTokenExpiredTime = 0
	DefaultSigningMethod    = "HS256"
)

type SaveMethodJWTEnum string

const (
	REDIS SaveMethodJWTEnum = "REDIS"
	JWT   SaveMethodJWTEnum = "JWT"
)

type Options struct {
	TokenExpiredTime time.Duration
	TokenSecretKey   string
	SigningMethod    string
	SaveMethod       SaveMethodJWTEnum
}

func DefaultOptions(secretKey string) *Options {
	return &Options{
		TokenExpiredTime: DefaultTokenExpiredTime,
		TokenSecretKey:   secretKey,
		SigningMethod:    DefaultSigningMethod,
		SaveMethod:       JWT,
	}
}
