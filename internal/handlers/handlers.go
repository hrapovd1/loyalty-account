package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hrapovd1/loyalty-account/internal/auth"
	"github.com/hrapovd1/loyalty-account/internal/config"
	"github.com/hrapovd1/loyalty-account/internal/dbstorage"
	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
	"github.com/hrapovd1/loyalty-account/internal/usecase"
)

type AppHandler struct {
	AccrualAddress string
	Storage        *dbstorage.DBStorage
	Logger         *log.Logger
}

// NewAppHandler return new app with db connect.
func NewAppHandler(conf config.Config, logger *log.Logger) (*AppHandler, error) {
	app := &AppHandler{
		AccrualAddress: conf.AccrualAddress,
		Logger:         logger,
	}
	storage, err := dbstorage.NewDB(conf.DatabaseDSN)
	if err != nil {
		return app, err
	}
	app.Storage = &storage

	return app, nil
}

// NewRouter return ready chi router with configured API urls.
func NewRouter(app *AppHandler) *chi.Mux {
	// Публикация API
	router := chi.NewRouter()
	router.Use(GzipMiddle)

	// Публично доступные маршруты.
	router.Group(
		func(r chi.Router) {
			r.Post("/api/user/register", app.Register)
			r.Post("/api/user/login", app.Login)
		})

	// Маршруты для аутентифицированных пользователей.
	router.Group(func(r chi.Router) {
		r.Use(Authenticator)
		r.Get("/api/user/orders", app.GetOrders)
		r.Post("/api/user/orders", app.PostOrders)
		r.Get("/api/user/balance", app.GetBalance)
		r.Post("/api/user/balance/withdraw", app.Withdraw)
		r.Get("/api/user/withdrawals", app.Withdrawals)
	})

	return router
}

// Register POST handler is used to register new users.
func (app *AppHandler) Register(rw http.ResponseWriter, r *http.Request) {
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

	if err := auth.CreateUser(r.Context(), app.Storage, user); err != nil {
		if errors.Is(err, dbstorage.ErrUserAlreadyExists) {
			http.Error(rw, err.Error(), http.StatusConflict)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := auth.GetToken(r.Context(), app.Storage, user)
	if err != nil {
		if errors.Is(err, dbstorage.ErrInvalidLoginPassword) {
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

// Login POST handler is used to get auth token.
func (app *AppHandler) Login(rw http.ResponseWriter, r *http.Request) {
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

	token, err := auth.GetToken(r.Context(), app.Storage, user)
	if err != nil {
		if errors.Is(err, dbstorage.ErrInvalidLoginPassword) {
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
	login := r.Header.Get("Login")

	rw.Header().Set("Content-Type", "application/json")

	orders, err := app.Storage.GetOrders(r.Context(), login)
	if err != nil {
		if errors.Is(err, dbstorage.ErrNoOrders) {
			http.Error(rw, err.Error(), http.StatusNoContent)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(usecase.OrdersTimeFormat(orders))
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

	if err := usecase.SaveOrder(r.Context(), app.Storage, login, bodyStr); err != nil {
		if errors.Is(err, dbstorage.ErrOrderExists) {
			http.Error(rw, "Order exists", http.StatusOK)
			return
		}
		if errors.Is(err, dbstorage.ErrOrderExistsAnother) {
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
	login := r.Header.Get("Login")

	result, err := app.Storage.GetBalance(r.Context(), login)
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

	if err = app.Storage.WithdrawOrder(r.Context(), login, orderLog); err != nil {
		if errors.Is(err, dbstorage.ErrNotEnoughFunds) {
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
	login := r.Header.Get("Login")

	orderLogs, err := app.Storage.GetOrderLogs(r.Context(), login)
	if err != nil {
		if errors.Is(err, dbstorage.ErrNoOrders) {
			http.Error(rw, err.Error(), http.StatusNoContent)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(usecase.OrderLogsTimeFormat(orderLogs))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write(resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}
