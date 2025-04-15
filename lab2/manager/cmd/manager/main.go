package main

import (
	"log"
	"sync"

	"manager/internal/connection"
	"manager/internal/rabbit"
	"manager/internal/server"
)

func main() {
	// Подключаемся к MongoDB.
	mongoClient, db, err := connection.ConnectMongoDB()
	if err != nil {
		log.Fatalf("[Main] Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(nil)
	taskColl := db.Collection("hash_tasks")

	// Подключаемся к RabbitMQ.
	rabbitConn, rabbitCh, rabbitURI, err := connection.ConnectRabbitMQ()
	if err != nil {
		log.Fatalf("[Main] Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	log.Println("[Main] All connections established")

	// Запуск consumer-а для очереди "results" с реконнектом.
	go rabbit.StartResultConsumer(rabbitCh, taskColl, &rabbitConn, rabbitURI)
	// Запуск publisher-а, публикующего подзадачи с реконнектом.
	go rabbit.StartPublisher(taskColl, &rabbitConn, rabbitURI, rabbitCh)
	// Запуск HTTP-сервера.
	go server.StartHTTPServer(taskColl)

	log.Println("[Main] Manager service is running...")
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
