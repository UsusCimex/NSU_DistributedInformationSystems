package main

import (
	"log"
	"worker/config"
	"worker/pool"
	"worker/registration"
	"worker/server"
)

func main() {
	cfg := config.Load()

	// Регистрируем worker у менеджера
	if err := registration.RegisterWithManager(cfg); err != nil {
		log.Fatal("Не удалось зарегистрироваться с менеджером:", err)
	}

	// Создаём пул воркеров
	workerPool := pool.New(cfg)

	// Запускаем HTTP-сервер
	server.Start(cfg, workerPool)
}
