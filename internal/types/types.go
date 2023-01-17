package types

import "github.com/golang-jwt/jwt/v4"

type Config struct {
	AppAddress     string
	DatabaseDSN    string
	AccrualAddress string
}

type DBModeler interface {
	Read() uint
}

type Claims struct {
	Login string `json:"login"`
	jwt.RegisteredClaims
}

type LoginResponse struct {
	Auth_token string `json:"auth_token"`
}

type Balance struct {
	Balance float64 `json:"current"`
	Summ    float64 `json:"withdrawn"`
}
