package jwt

import (
	"time"
)

// Define constants
const (
	DefaultTokenExpiredTime = 0
	DefaultSigningMethod    = "HS256"
)

type Options struct {
	TokenExpiredTime time.Duration
	TokenSecretKey   string
	SigningMethod    string
}

func DefaultOptions(secretKey string) *Options {
	return &Options{
		TokenExpiredTime: DefaultTokenExpiredTime,
		TokenSecretKey:   secretKey,
		SigningMethod:    DefaultSigningMethod,
	}
}
