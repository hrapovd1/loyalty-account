package types

import "github.com/golang-jwt/jwt/v4"

type Config struct {
	AppAddress     string
	DatabaseDSN    string
	AccrualAddress string
}

type Claims struct {
	Login string `json:"login"`
	jwt.RegisteredClaims
}

type LoginResponse struct {
	Authtoken string `json:"auth_token"`
}

type OrderResponse struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

type OrderLogResponse struct {
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type Balance struct {
	Balance float64 `json:"current"`
	Summ    float64 `json:"withdrawn"`
}

type AccrualAnswer struct {
	OrderNumber string  `json:"order"`
	Status      string  `json:"status"`
	Accrual     float64 `json:"accrual"`
}
