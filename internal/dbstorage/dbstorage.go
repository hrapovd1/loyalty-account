package dbstorage

import (
	"context"
	"database/sql"
	"log"

	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(pdb *sql.DB) error {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
	if err != nil {
		return err
	}
	return db.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.Order{},
		&models.OrderLog{},
	)
}

func Save(ctx context.Context, pdb *sql.DB, dbmodel types.DBModeler) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return err
		}
		log.Println(dbmodel)
		user := dbmodel.(models.User)

		err = db.Create(&user).Error
		return err
	}
}

func GetUser(ctx context.Context, pdb *sql.DB, login string) (*models.User, error) {
	var user models.User
	select {
	case <-ctx.Done():
		return &user, nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return &user, err
		}
		err = db.First(&user, "login = ?", login).Error
		return &user, err
	}
}

func GetOrders(ctx context.Context, pdb *sql.DB, login string) (*[]models.Order, error) {
	orders := make([]models.Order, 0)
	select {
	case <-ctx.Done():
		return &orders, nil
	default:
		user, err := GetUser(ctx, pdb, login)
		if err != nil {
			return &orders, err
		}
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return &orders, err
		}

		db.Where("user_id = ?", user.ID).Find(&orders)
		return &orders, err
	}
}

func CreateOrder(ctx context.Context, pdb *sql.DB, login string, number uint64) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		order := models.Order{
			Number: number,
		}
		user, err := GetUser(ctx, pdb, login)
		if err != nil {
			return err
		}
		order.UserID = user.ID
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return err
		}

		result := db.Create(&order)
		return result.Error
	}
}

func GetBalance(ctx context.Context, pdb *sql.DB, login string) (*types.Balance, error) {
	var result types.Balance

	select {
	case <-ctx.Done():
		return &result, nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return &result, err
		}

		err = db.Model(&models.User{}).Select(
			"sum(order_logs.sum) as summ, accounts.balance as balance",
		).Joins(
			"left join order_logs on order_logs.user_id = users.id",
		).Joins(
			"left join accounts on accounts.user_id = users.id",
		).Group("accounts.balance").Where("users.login = ?", login).Scan(&result).Error
		return &result, err
	}
}

func GetOrderLogs(ctx context.Context, pdb *sql.DB, login string) (*[]models.OrderLog, error) {
	orders := make([]models.OrderLog, 0)
	select {
	case <-ctx.Done():
		return &orders, nil
	default:
		user, err := GetUser(ctx, pdb, login)
		if err != nil {
			return &orders, err
		}
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return &orders, err
		}

		err = db.Where("user_id = ?", user.ID).Find(&orders).Error
		return &orders, err
	}
}

func WithdrawnOrder(ctx context.Context, pdb *sql.DB, login string, orderLog models.OrderLog) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	return nil
}
