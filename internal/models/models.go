package models

import "database/sql"

type User struct {
	ID        uint       `gorm:"primaryKey" json:"-"`
	Login     string     `gorm:"uniqueIndex:idx_logins" json:"login"`
	Password  string     `json:"password,omitempty"`
	Account   Account    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	Orders    []Order    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	OrderLogs []OrderLog `json:"-"`
}

type Account struct {
	ID      uint `gorm:"primaryKey"`
	UserID  uint
	Balance sql.NullFloat64
}

type Order struct {
	ID         uint    `gorm:"primaryKey" json:"-"`
	UserID     uint    `json:"-"`
	Number     string  `gorm:"uniqueIndex:idx_numbers,sort:desc" json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt int64   `gorm:"autoCreateTime" json:"uploaded_at"`
}

type OrderLog struct {
	ID          uint    `gorm:"primaryKey" json:"-"`
	UserID      uint    `json:"-"`
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt int64   `gorm:"autoCreateTime" json:"processed_at"`
}
