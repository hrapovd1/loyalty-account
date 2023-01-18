package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hrapovd1/loyalty-account/internal/dbstorage"
	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
)

func SaveOrder(ctx context.Context, pdb *sql.DB, login string, number string) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		order := models.Order{
			Number: number,
			Status: "NEW",
		}
		if err := dbstorage.CreateOrder(ctx, pdb, login, order); err != nil {
			return err
		}
		return nil
	}
}

func Dispatcher(ctx context.Context, pdb *sql.DB, logger *log.Logger, accrualAddress string) {
	select {
	case <-ctx.Done():
		return
	default:
		answer := types.AccrualAnswer{}
		checkNew := true
		client := resty.New()
		for {
			var status string
			if checkNew {
				status = "NEW"
			} else {
				status = "PROCESSING"
			}
			orderNumbers, err := dbstorage.DispatchGetOrders(ctx, pdb, status)
			if err != nil {
				logger.Print(err)
			}
			if len(*orderNumbers) == 0 {
				checkNew = !checkNew
				time.Sleep(time.Second * 5)
				continue
			}
			logger.Printf("Dispatcher, orderNums = %+v, type = %T", orderNumbers, orderNumbers)
			for _, order := range *orderNumbers {
				resp, err := client.R().Get(fmt.Sprintf("http://%v/api/orders/%v", accrualAddress, order))
				if err != nil {
					logger.Print(err)
					continue
				}
				if err = json.Unmarshal(resp.Body(), &answer); err != nil {
					logger.Print(err)
				}
				if resp.StatusCode() != http.StatusOK {
					logger.Printf("For order number = %v, accrual system returned status: %v", order, resp.StatusCode())
					logger.Printf("Answer = '%+v'", resp.String())
					continue
				}
				switch answer.Status {
				case "INVALID", "PROCESSED":
					if err = dbstorage.DispatchUpdateOrder(
						ctx, pdb,
						models.Order{
							Number:  answer.OrderNumber,
							Status:  answer.Status,
							Accrual: answer.Accrual,
						},
					); err != nil {
						logger.Print(err)
						continue
					}
				case "REGISTERED", "PROCESSING":
					continue
				}
			}
			checkNew = !checkNew
		}
	}
}
