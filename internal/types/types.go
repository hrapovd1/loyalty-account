package types

type UserModel struct {
	ID        uint `gorm:"primaryKey"`
	Login     string
	Password  string
	Account   AccountModel `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Orders    []OrderModel `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	OrderLogs []OrderLogModel
}

type AccountModel struct {
	ID          uint64 `gorm:"primaryKey"`
	UserModelID uint
	Balance     float64
}

type OrderModel struct {
	ID          uint64 `gorm:"primaryKey"`
	UserModelID uint
	Number      uint64
	Status      string
	Accrual     float64
	UploadedAt  int64 `gorm:"autoCreateTime"`
}

type OrderLogModel struct {
	ID          uint64 `gorm:"primaryKey"`
	UserModelID uint
	OrderNumber uint64
	Sum         float64
	ProcessedAt int64 `gorm:"autoCreateTime"`
}
