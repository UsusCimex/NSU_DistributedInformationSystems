package main

import (
	"log"
	"os"
	"time"

	"common/amqputil"
	"common/constants"
	"common/logger"
	"worker/internal/consumer"
)

func main() {
	logger.Log("Worker", "Запуск Worker...")

	// Определяем URI RabbitMQ из переменных окружения или используем стандартное значение
	rabbitURI := os.Getenv("RABBITMQ_URI")
	if rabbitURI == "" {
		rabbitURI = constants.DefaultRabbitURI
	}

	// Подключаемся к RabbitMQ
	conn, err := amqputil.ConnectRabbitMQ(rabbitURI)
	if err != nil {
		log.Fatalf("Не удалось подключиться к RabbitMQ: %v", err)
	}
	defer conn.Close()

	for {
		err = consumer.Consume(&conn, rabbitURI)
		if err != nil {
			logger.Log("Worker", "Ошибка в Consumer: "+err.Error())

			if conn.IsClosed() {
				newConn, connErr := amqputil.ConnectRabbitMQ(rabbitURI)
				if connErr != nil {
					logger.Log("Worker", "Не удалось восстановить соединение: "+connErr.Error())
				} else {
					conn = newConn
				}
			}
		}

		logger.Log("Worker", "Перезапуск consumer через 5 секунд...")
		time.Sleep(5 * time.Second)
	}
}
