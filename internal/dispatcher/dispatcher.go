package dispatcher

import (
	"context"
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

const (
	statNew        = "NEW"
	statProcessing = "PROCESSING"
	checkPause     = 5 // seconds
	empty          = 0
)

func Dispatcher(ctx context.Context, storage *dbstorage.DBStorage, logger *log.Logger, accrualAddress string) {
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
				status = statNew
			} else {
				status = statProcessing
			}
			orderNumbers, err := storage.DispatchGetOrders(ctx, status)
			if err != nil {
				logger.Print(err)
			}
			if len(*orderNumbers) == empty {
				checkNew = !checkNew
				time.Sleep(time.Second * checkPause)
				continue
			}
			logger.Printf("Dispatcher, orderNums = %+v, type = %T", orderNumbers, orderNumbers)
			for _, order := range *orderNumbers {
				resp, err := client.R().Get(fmt.Sprintf("%v/api/orders/%v", accrualAddress, order))
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
					if err = storage.DispatchUpdateOrder(
						ctx,
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
