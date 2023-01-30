package usecase

import (
	"context"
	"strconv"
	"time"

	"github.com/hrapovd1/loyalty-account/internal/dbstorage"
	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
)

func SaveOrder(ctx context.Context, storage *dbstorage.DBStorage, login string, number string) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		order := models.Order{
			Number: number,
			Status: "NEW",
		}
		if err := storage.CreateOrder(ctx, login, order); err != nil {
			return err
		}
		return nil
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

// Func check number according Luhn algorithm
// https://ru.wikipedia.org/wiki/Алгоритм_Луна
func IsOrderNumValid(number string) bool {
	num, err := strconv.Atoi(number)
	if err != nil {
		return false
	}
	return (num%10+checksum(num/10))%10 == 0
}

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
