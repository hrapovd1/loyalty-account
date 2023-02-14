package dispatcher

import (
	"context"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hrapovd1/loyalty-account/internal/dbstorage"
	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
)

const (
	statNew        = "NEW"
	statProcessing = "PROCESSING"
	checkPause     = 5 * time.Second
)

type Dispatcher struct {
	Storage        *dbstorage.DBStorage
	Logger         *log.Logger
	AccrualAddress string
}

func (disp Dispatcher) Run(ctx context.Context) {
	answer := types.AccrualAnswer{}
	checkNew := true
	client := resty.New()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		clientCTX, cltCancel := context.WithTimeout(ctx, 3*time.Second)
		defer cltCancel()
		dbCTX, dbCancel := context.WithTimeout(ctx, time.Second)
		defer dbCancel()

		var status string
		if checkNew {
			status = statNew
		} else {
			status = statProcessing
		}
		orderNumbers, err := disp.Storage.DispatchGetOrders(dbCTX, status)
		if err != nil {
			disp.Logger.Print(err)
		}

		if len(orderNumbers) == 0 {
			checkNew = !checkNew
			time.Sleep(checkPause)
			continue
		}
		disp.Logger.Printf("Dispatcher, orderNums = %+v, type = %T", orderNumbers, orderNumbers)
		for _, order := range orderNumbers {
			resp, err := client.R().
				SetContext(clientCTX).
				SetResult(&answer).
				Get(path.Join(disp.AccrualAddress, "api/orders", order))
			if err != nil {
				disp.Logger.Print(err)
				continue
			}
			if resp.StatusCode() != http.StatusOK {
				disp.Logger.Printf("For order number = %v, accrual system returned status: %v", order, resp.StatusCode())
				disp.Logger.Printf("Answer = '%+v'", resp.String())
				continue
			}
			switch answer.Status {
			case "INVALID", "PROCESSED":
				if err = disp.Storage.DispatchUpdateOrder(
					dbCTX,
					models.Order{
						Number:  answer.OrderNumber,
						Status:  answer.Status,
						Accrual: answer.Accrual,
					},
				); err != nil {
					disp.Logger.Print(err)
					continue
				}
			case "REGISTERED", "PROCESSING":
				continue
			}
		}
		checkNew = !checkNew
	}
}
