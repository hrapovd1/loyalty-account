package main

import (
	"log"
	"os"

	"github.com/go-chi/chi/v5"
)

func main() {
	logger = log.New(os.Stdout, "GOPHERMART:\t", log.Ldate|log.Ltime)
	// Чтение флагов и установка конфигурации

	// Объявление подключения к БД.

	// Публикация API
	router := chi.NewRouter()
	router.Post("/api/user/register")

}
