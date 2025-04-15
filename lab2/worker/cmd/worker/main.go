package main

import (
	"log"
	"os"
	"time"

	"common/amqputil"
	"common/logger"
	"worker/internal/consumer"
)

func main() {
	logger.Log("Worker", "Запуск компонента Worker...")

	rabbitURI := os.Getenv("RABBITMQ_URI")
	if rabbitURI == "" {
		rabbitURI = "amqp://guest:guest@rabbitmq:5672/"
	}
	conn, err := amqputil.ConnectRabbitMQ(rabbitURI)
	if err != nil {
		log.Fatalf("Не удалось подключиться к RabbitMQ: %v", err)
	}

	for {
		// Передаём указатель на переменную соединения, чтобы его можно было обновить при реконнекте
		err = consumer.Consume(&conn, rabbitURI)

		if err != nil {
			logger.Log("Worker", "Ошибка работы consumer: "+err.Error())
			// Если текущее соединение закрыто, пробуем восстановить его
			if conn.IsClosed() {
				newConn, err := amqputil.ConnectRabbitMQ(rabbitURI)
				if err != nil {
					logger.Log("Worker", "Не удалось восстановить соединение: "+err.Error())
				} else {
					conn = newConn
				}
			}
		}
		logger.Log("Worker", "Перезапуск consumer через 5 секунд...")
		time.Sleep(5 * time.Second)
	}
}
