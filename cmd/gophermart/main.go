package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/hrapovd1/loyalty-account/internal/config"
	"github.com/hrapovd1/loyalty-account/internal/dbstorage"
	"github.com/hrapovd1/loyalty-account/internal/handlers"
)

func main() {
	logger := log.New(os.Stdout, "GOPHERMART:\t", log.Ldate|log.Ltime)

	// Чтение флагов и установка конфигурации
	appConf, err := config.NewAppConf(config.GetAppFlags())
	if err != nil {
		logger.Fatalln(err)
	}

	// Настройка подключения к БД и создание приложения.
	app, err := handlers.NewAppHandler(*appConf, logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer app.DB.Close()
	if err := dbstorage.InitDB(app.DB); err != nil {
		logger.Fatalln(err)
	}

	// Публикация API
	router := chi.NewRouter()
	router.Use(handlers.GzipMiddle)

	// Публично доступные маршруты.
	router.Group(
		func(r chi.Router) {
			r.Post("/api/user/register", app.Register)
			r.Post("/api/user/login", app.Login)
		})

	// Маршруты для аутентифицированных пользователей.
	router.Group(func(r chi.Router) {
		r.Use(handlers.Authenticator)
		r.Get("/api/user/orders", app.GetOrders)
		r.Post("/api/user/orders", app.PostOrders)
		r.Get("/api/user/balance", app.GetBalance)
		r.Post("/api/user/balance/withdraw", app.Withdraw)
		r.Get("/api/user/withdrawals", app.Withdrawals)
	})

	logger.Println("App is waiting connections on: ", appConf.AppAddress)
	logger.Fatal(http.ListenAndServe(appConf.AppAddress, router))
}
