package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/hrapovd1/loyalty-account/internal/config"
	"github.com/hrapovd1/loyalty-account/internal/dispatcher"
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
		logger.Fatalln(err)
	}
	defer app.Storage.Close()

	if err := app.Storage.InitDB(); err != nil {
		logger.Fatalln(err)
	}

	// Запуск диспетчера системы расчета баллов
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dsptchr := dispatcher.Dispatcher{
		Storage:        app.Storage,
		Logger:         logger,
		AccrualAddress: appConf.AccrualAddress,
	}

	go dsptchr.Run(ctx)

	// Открытие порта и обслуживание API запросов
	router := handlers.NewRouter(app)
	logger.Println("App is waiting connections on: ", appConf.AppAddress)
	logger.Fatal(http.ListenAndServe(appConf.AppAddress, router))
}
