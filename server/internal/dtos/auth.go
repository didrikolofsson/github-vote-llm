package dtos

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	UserID int64  `json:"uid"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}
