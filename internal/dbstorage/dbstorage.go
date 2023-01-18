package dbstorage

import (
	"context"
	"database/sql"
	"fmt"
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

func CreateUser(ctx context.Context, pdb *sql.DB, user models.User) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return err
		}
		log.Println(user)

		err = db.Create(&user).Error
		if err != nil {
			return err
		}
		// Bonus for new user :)
		return IncreaceBalance(ctx, pdb, user.Login, float64(500))
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

func CreateOrder(ctx context.Context, pdb *sql.DB, login string, order models.Order) error {
	select {
	case <-ctx.Done():
		return nil
	default:
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

func WithdrawOrder(ctx context.Context, pdb *sql.DB, login string, orderLog models.OrderLog) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		if orderLog.Sum <= 0 {
			return fmt.Errorf("sum must be >0")
		}
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return err
		}
		// transaction start
		return db.Transaction(
			func(tx *gorm.DB) error {
				var user models.User
				if err = db.First(&user, "login = ?", login).Error; err != nil {
					return err
				}
				// compare balance to sum
				balance := models.Account{UserID: user.ID}
				if err = tx.Select("balance").Find(&balance).Error; err != nil {
					return err
				}
				if balance.Balance.Float64 < orderLog.Sum {
					return fmt.Errorf("not enough funds")
				}
				// balance - sum
				if err = tx.Model(&models.Account{}).Where(
					"user_id = (?)", user.ID,
				).UpdateColumn("balance", gorm.Expr("balance - ?", orderLog.Sum)).Error; err != nil {
					return err
				}
				// write orderLog entry
				orderLog.UserID = user.ID
				return tx.Create(&orderLog).Error
			},
		)
		// transaction end
	}
}

func IncreaceBalance(ctx context.Context, pdb *sql.DB, login string, sum float64) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		if sum <= 0 {
			return fmt.Errorf("sum must be >0")
		}
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return err
		}
		var user models.User
		if err = db.First(&user, "login = ?", login).Error; err != nil {
			return err
		}
		return db.Model(&models.Account{}).Where(
			"user_id = (?)", user.ID,
		).UpdateColumn("balance", gorm.Expr("balance + ?", sum)).Error
	}
}

func DispatchGetOrders(ctx context.Context, pdb *sql.DB, status string) (*[]string, error) {
	numList := make([]string, 0)
	select {
	case <-ctx.Done():
		return &numList, nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return &numList, err
		}

		rows, err := db.Model(&models.Order{}).Select("number").Where("status = ?", status).Rows()
		defer rows.Close()
		if err != nil {
			return &numList, err
		}
		for rows.Next() {
			var number string
			if err = rows.Scan(&number); err != nil {
				return &numList, err
			}
			numList = append(numList, number)
		}
		if err = rows.Err(); err != nil {
			return &numList, err
		}
		return &numList, nil
	}
}

func DispatchUpdateOrder(ctx context.Context, pdb *sql.DB, order models.Order) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pdb}), &gorm.Config{})
		if err != nil {
			return err
		}
		return db.Model(&models.Order{}).Where(
			"number = ?", order.Number,
		).Updates(
			models.Order{Status: order.Status, Accrual: order.Accrual},
		).Error
	}
}
