package main

import (
	"sync"
	"time"

	"common/logger"
	"log"
	"worker/internal/connection"
	"worker/internal/processor"
	"worker/internal/rabbitmq"
)

func main() {
	logger.Log("Worker", "Запуск...")
	conn, rabbitURI, err := connection.ConnectRabbitMQ()
	if err != nil {
		logger.Log("Worker", "Не удалось подключиться к RabbitMQ: "+err.Error())
		log.Fatal(err)
	}
	defer conn.Close()

	proc := processor.NewProcessor(nil)

	for {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 3)
		logger.Log("Worker", "Запуск потребителя...")
		rabbitmq.ConsumeTasks(&conn, rabbitURI, proc, sem, &wg)
		wg.Wait()
		logger.Log("Worker", "Consumer завершил работу, перезапуск через 5 секунд")
		time.Sleep(5 * time.Second)
	}
}
