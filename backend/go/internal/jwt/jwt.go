package jwtutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Maker struct {
	secret []byte
	ttl    time.Duration
}

func New(secret string, ttl time.Duration) *Maker {
	return &Maker{secret: []byte(secret), ttl: ttl}
}

func (m *Maker) Create(sub string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": sub,
		"iat": now.Unix(),
		"exp": now.Add(m.ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Maker) Parse(tokenStr string) (jwt.MapClaims, error) {
	tk, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if c, ok := tk.Claims.(jwt.MapClaims); ok && tk.Valid {
		return c, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
