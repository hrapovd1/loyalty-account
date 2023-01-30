package dbstorage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBStorage struct {
	DB *sql.DB
}

func NewDB(dsn string) (DBStorage, error) {
	dbConnect, err := sql.Open("pgx", dsn)
	return DBStorage{DB: dbConnect}, err
}

func (ds *DBStorage) Close() {
	ds.DB.Close()
}

func (ds *DBStorage) InitDB() error {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
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

func (ds *DBStorage) CreateUser(ctx context.Context, user models.User) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
		if err != nil {
			return err
		}
		var exists bool
		db.Model(&models.User{}).
			Select("count(*) > 0").
			Where("login = ?", user.Login).
			Find(&exists)
		if exists {
			return ErrUserAlreadyExists
		}

		return db.Create(&user).Error
	}
}

func (ds *DBStorage) GetUser(ctx context.Context, login string) (*models.User, error) {
	var user models.User
	select {
	case <-ctx.Done():
		return &user, nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
		if err != nil {
			return &user, err
		}
		err = db.First(&user, "login = ?", login).Error
		return &user, err
	}
}

func (ds *DBStorage) GetOrders(ctx context.Context, login string) (*[]models.Order, error) {
	orders := make([]models.Order, 0)
	select {
	case <-ctx.Done():
		return &orders, nil
	default:
		user, err := ds.GetUser(ctx, login)
		if err != nil {
			return &orders, err
		}
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
		if err != nil {
			return &orders, err
		}

		db.Where("user_id = ?", user.ID).Find(&orders)
		if len(orders) == 0 {
			return &orders, ErrNoOrders
		}
		return &orders, err
	}
}

func (ds *DBStorage) CreateOrder(ctx context.Context, login string, order models.Order) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		user, err := ds.GetUser(ctx, login)
		if err != nil {
			return err
		}
		order.UserID = user.ID
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
		if err != nil {
			return err
		}

		var dbOrder models.Order
		db.Where("number = ?", order.Number).Find(&dbOrder)
		if dbOrder.Number != "" {
			if dbOrder.UserID == user.ID {
				return ErrOrderExists
			} else {
				return ErrOrderExistsAnother
			}
		}
		return db.Create(&order).Error

	}
}

func (ds *DBStorage) GetBalance(ctx context.Context, login string) (*types.Balance, error) {
	var result types.Balance

	select {
	case <-ctx.Done():
		return &result, nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
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

func (ds *DBStorage) GetOrderLogs(ctx context.Context, login string) (*[]models.OrderLog, error) {
	orders := make([]models.OrderLog, 0)
	select {
	case <-ctx.Done():
		return &orders, nil
	default:
		user, err := ds.GetUser(ctx, login)
		if err != nil {
			return &orders, err
		}
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
		if err != nil {
			return &orders, err
		}

		err = db.Where("user_id = ?", user.ID).Find(&orders).Error
		if len(orders) == 0 {
			return &orders, ErrNoOrders
		}
		return &orders, err
	}
}

func (ds *DBStorage) WithdrawOrder(ctx context.Context, login string, orderLog models.OrderLog) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		if orderLog.Sum <= 0 {
			return fmt.Errorf("sum must be >0")
		}
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
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
					return ErrNotEnoughFunds
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

func (ds *DBStorage) IncreaceBalance(ctx context.Context, login string, sum float64) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		if sum <= 0 {
			return fmt.Errorf("sum must be >0")
		}
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
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

func (ds *DBStorage) DispatchGetOrders(ctx context.Context, status string) (*[]string, error) {
	numList := make([]string, 0)
	select {
	case <-ctx.Done():
		return &numList, nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
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

func (ds *DBStorage) DispatchUpdateOrder(ctx context.Context, order models.Order) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: ds.DB}), &gorm.Config{})
		if err != nil {
			return err
		}
		// transaction start
		return db.Transaction(
			func(tx *gorm.DB) error {
				if err := tx.Model(&models.Order{}).Where(
					"number = ?", order.Number,
				).Updates(
					models.Order{Status: order.Status, Accrual: order.Accrual},
				).Error; err != nil {
					return err
				}
				if order.Accrual > 0 {
					if err = tx.Model(&models.Account{}).Where(
						"user_id = (?)",
						tx.Model(&models.Order{}).Select("id").Where(
							"number = ?", order.Number,
						),
					).UpdateColumn(
						"balance",
						gorm.Expr("balance + ?", order.Accrual),
					).Error; err != nil {
						return err
					}
				}
				return nil
			},
		)
		// transaction end
	}
}
