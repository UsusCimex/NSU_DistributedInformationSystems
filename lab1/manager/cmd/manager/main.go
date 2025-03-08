package main

import (
	"log"
	"manager/balancer"
	"manager/dispatcher"
	"manager/queue"
	"manager/server"
	"manager/store"
)

func main() {
	// Инициализация хранилища
	store.Init()

	// Инициализация балансировщика
	balancer.Init()

	// Создание очереди задач
	taskQueue := queue.NewTaskQueue()

	// Создание диспетчера задач
	taskDispatcher := dispatcher.NewTaskDispatcher(taskQueue)
	taskDispatcher.Start()

	// Запуск HTTP-сервера
	log.Println("Starting manager on :8080")
	server.Start(taskQueue, taskDispatcher)
}
