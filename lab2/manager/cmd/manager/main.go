package main

import (
	"sync"

	"common/logger"
	"manager/internal/connection"
	"manager/internal/rabbit"
	"manager/internal/server"

	"log"
)

func main() {
	// Подключение к MongoDB.
	mongoClient, db, err := connection.ConnectMongoDB()
	if err != nil {
		logger.Log("Manager", "Не удалось подключиться к MongoDB: "+err.Error())
		log.Fatal(err)
	}
	defer mongoClient.Disconnect(nil)
	taskColl := db.Collection("hash_tasks")

	// Подключение к RabbitMQ.
	rabbitConn, rabbitCh, rabbitURI, err := connection.ConnectRabbitMQ()
	if err != nil {
		logger.Log("Manager", "Не удалось подключиться к RabbitMQ: "+err.Error())
		log.Fatal(err)
	}
	defer rabbitConn.Close()

	logger.Log("Manager", "Соединения установлены")

	// Запуск consumer-а для очереди "results" и publisher-а подзадач.
	go rabbit.StartResultConsumer(rabbitCh, taskColl, &rabbitConn, rabbitURI)
	go rabbit.StartPublisher(taskColl, &rabbitConn, rabbitURI, rabbitCh)
	// Запуск HTTP-сервера.
	go server.StartHTTPServer(taskColl)

	logger.Log("Manager", "Сервис запущен")
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
