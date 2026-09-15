package util

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Claims struct {
	UserID uint   `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func SignJWT(secret string, id uint, role string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: id, Role: role, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour))}}).SignedString([]byte(secret))
}
func ParseJWT(secret, token string) (*Claims, error) {
	c := &Claims{}
	_, e := jwt.ParseWithClaims(token, c, func(t *jwt.Token) (any, error) { return []byte(secret), nil })
	return c, e
}
