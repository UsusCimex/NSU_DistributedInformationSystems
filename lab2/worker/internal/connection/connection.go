package connection

import (
	"time"

	"common/logger"

	"github.com/streadway/amqp"
)

func ConnectRabbitMQ() (*amqp.Connection, string, error) {
	rabbitURI := "amqp://guest:guest@rabbitmq:5672/"
	const maxRetries = 10
	var conn *amqp.Connection
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(rabbitURI)
		if err == nil {
			logger.Log("Worker Connection", "Подключение к RabbitMQ установлено")
			return conn, rabbitURI, nil
		}
		logger.Log("Worker Connection", "Ошибка подключения: "+err.Error())
		time.Sleep(5 * time.Second)
	}
	return nil, rabbitURI, err
}
