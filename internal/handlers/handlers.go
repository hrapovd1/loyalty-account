package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/hrapovd1/loyalty-account/internal/auth"
	"github.com/hrapovd1/loyalty-account/internal/config"
	"github.com/hrapovd1/loyalty-account/internal/dbstorage"
	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
	"github.com/hrapovd1/loyalty-account/internal/usecase"
)

type AppHandler struct {
	AccrualAddress string
	DB             *sql.DB
	Logger         *log.Logger
}

// NewAppHandler return new app with db connect.
func NewAppHandler(conf config.Config, logger *log.Logger) (*AppHandler, error) {
	app := &AppHandler{
		AccrualAddress: conf.AccrualAddress,
		Logger:         logger,
	}
	dbConnect, err := sql.Open("pgx", conf.DatabaseDSN)
	if err != nil {
		return app, err
	}
	app.DB = dbConnect

	return app, nil
}

// Register POST handler use to register new users.
func (app *AppHandler) Register(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	user := models.User{
		Account: models.Account{Balance: sql.NullFloat64{
			Float64: float64(0),
			Valid:   true,
		}},
	}
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if user.Login == "" || user.Password == "" {
		http.Error(rw, "wrong body format", http.StatusBadRequest)
		return
	}

	if err := auth.CreateUser(ctx, app.DB, user); err != nil {
		if err == dbstorage.ErrUserAlreadyExists {
			http.Error(rw, err.Error(), http.StatusConflict)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := auth.GetToken(ctx, app.DB, user)
	if err != nil {
		if err == dbstorage.ErrInvalidLoginPassword {
			http.Error(rw, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Authorization", token)
	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write([]byte(""))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Login POST handler use to get auth token.
func (app *AppHandler) Login(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var user models.User
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if user.Login == "" || user.Password == "" {
		http.Error(rw, "wrong body format", http.StatusBadRequest)
		return
	}

	token, err := auth.GetToken(ctx, app.DB, user)
	if err != nil {
		if err == dbstorage.ErrInvalidLoginPassword {
			http.Error(rw, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(types.LoginResponse{Authtoken: token})
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Authorization", token)
	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write(resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetOrders handler return list of put orders.
func (app *AppHandler) GetOrders(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	login, ok := r.Header["Login"]
	if !ok {
		http.Error(rw, "login not found", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")

	orders, err := dbstorage.GetOrders(ctx, app.DB, login[0])
	if err != nil {
		if err == dbstorage.ErrNoOrders {
			http.Error(rw, err.Error(), http.StatusNoContent)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(usecase.OrdersTimeFormat(*orders))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write(resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

}

// PostOrders handler put new order.
func (app *AppHandler) PostOrders(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	bodyStr := string(body)

	login := r.Header.Get("Login")

	contentType := r.Header.Get("Content-type")
	if contentType != "text/plain" || bodyStr == "" {
		http.Error(rw, "Invalid request", http.StatusBadRequest)
		return
	}

	if !usecase.IsOrderNumValid(bodyStr) {
		http.Error(rw, "Order number is not valid", http.StatusUnprocessableEntity)
		return
	}

	if err := usecase.SaveOrder(ctx, app.DB, login, bodyStr); err != nil {
		if err == dbstorage.ErrOrderExists {
			http.Error(rw, "Order exists", http.StatusOK)
			return
		}
		if err == dbstorage.ErrOrderExistsAnother {
			http.Error(rw, "Order exists", http.StatusConflict)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusAccepted)
	_, err = rw.Write([]byte(""))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetBalance handler return user accrual balance.
func (app *AppHandler) GetBalance(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	login, ok := r.Header["Login"]
	if !ok {
		http.Error(rw, "login not found", http.StatusInternalServerError)
		return
	}

	result, err := dbstorage.GetBalance(ctx, app.DB, login[0])
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write(resultJSON)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Withdraw POST handler withdraw accrual to pay order.
func (app *AppHandler) Withdraw(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	login := r.Header.Get("Login")

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var orderLog models.OrderLog
	if err := json.Unmarshal(body, &orderLog); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if !usecase.IsOrderNumValid(orderLog.OrderNumber) {
		http.Error(rw, "Order number is not valid", http.StatusUnprocessableEntity)
		return
	}

	if err = dbstorage.WithdrawOrder(ctx, app.DB, login, orderLog); err != nil {
		if err == dbstorage.ErrNotEnoughFunds {
			http.Error(rw, err.Error(), http.StatusPaymentRequired)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write([]byte(""))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

}

// Withdrawals GET handler return list of payment with accrual.
func (app *AppHandler) Withdrawals(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	login, ok := r.Header["Login"]
	if !ok {
		http.Error(rw, "login not found", http.StatusInternalServerError)
		return
	}

	orderLogs, err := dbstorage.GetOrderLogs(ctx, app.DB, login[0])
	if err != nil {
		if err == dbstorage.ErrNoOrders {
			http.Error(rw, err.Error(), http.StatusNoContent)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(usecase.OrderLogsTimeFormat(*orderLogs))
	if err != nil {
		// TODO: analyze error to different response
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO different codes for empty.
	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write(resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}
