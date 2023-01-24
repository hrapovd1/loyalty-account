package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

func OrdersTimeFormat(orders []models.Order) *[]types.OrderResponse {
	orderResp := make([]types.OrderResponse, 0)
	for _, order := range orders {
		resp := types.OrderResponse{
			Number:     order.Number,
			Status:     order.Status,
			UploadedAt: time.Unix(order.UploadedAt, 0).Format(time.RFC3339),
		}
		if order.Accrual > 0 {
			resp.Accrual = order.Accrual
		}
		orderResp = append(orderResp, resp)
	}

	return &orderResp
}

func OrderLogsTimeFormat(orders []models.OrderLog) *[]types.OrderLogResponse {
	orderLogResp := make([]types.OrderLogResponse, 0)
	for _, order := range orders {
		orderLogResp = append(orderLogResp, types.OrderLogResponse{
			OrderNumber: order.OrderNumber,
			Sum:         order.Sum,
			ProcessedAt: time.Unix(order.ProcessedAt, 0).Format(time.RFC3339),
		})
	}

	return &orderLogResp
}

func IsOrderNumValid(number string) bool {
	num, err := strconv.Atoi(number)
	if err != nil {
		return false
	}
	return (num%10+checksum(num/10))%10 == 0
}

// Func check number according Luhn algorithm
// https://ru.wikipedia.org/wiki/Алгоритм_Луна
func checksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
