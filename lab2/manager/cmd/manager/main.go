package main

import (
	"log"
	"sync"

	"common/logger"
	"manager/internal/connection"
	"manager/internal/rabbit"
	"manager/internal/server"
)

func main() {
	// Connect to MongoDB
	mongoClient, db, err := connection.ConnectMongoDB()
	if err != nil {
		logger.Log("Manager", "Не удалось подключиться к MongoDB: "+err.Error())
		log.Fatal(err)
	}
	defer func() {
		_ = mongoClient.Disconnect(nil)
	}()
	taskColl := db.Collection("hash_tasks")

	// Connect to RabbitMQ
	rabbitConn, rabbitCh, rabbitURI, err := connection.ConnectRabbitMQ()
	if err != nil {
		logger.Log("Manager", "Не удалось подключиться к RabbitMQ: "+err.Error())
		log.Fatal(err)
	}
	defer rabbitConn.Close()

	logger.Log("Manager", "Соединения с MongoDB и RabbitMQ установлены")

	// Запускаем фоновые горутины:
	// 1. Потребитель очереди "results" для обработки результатов завершенных подзадач.
	go rabbit.StartResultConsumer(rabbitCh, taskColl, &rabbitConn, rabbitURI)
	// 2. Публикатор для отправки новых подзадач в очередь "tasks".
	go rabbit.StartPublisher(taskColl, &rabbitConn, rabbitURI, rabbitCh)
	// 3. HTTP-сервер для обработки входящих API-запросов.
	go server.StartHTTPServer(taskColl)

	logger.Log("Manager", "Все компоненты запущены")

	// Предотвращаем завершение главной горутины (сервисы работают в отдельных горутинах).
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
