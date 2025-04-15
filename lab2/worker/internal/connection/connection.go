package connection

import (
	"log"
	"time"

	"github.com/streadway/amqp"
)

// ConnectRabbitMQ устанавливает подключение к RabbitMQ с повторными попытками.
func ConnectRabbitMQ() (*amqp.Connection, string, error) {
	rabbitURI := "amqp://guest:guest@rabbitmq:5672/"
	const maxRetries = 10
	var conn *amqp.Connection
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(rabbitURI)
		if err == nil {
			log.Println("[Worker Connection] Connected to RabbitMQ")
			return conn, rabbitURI, nil
		}
		log.Printf("[Worker Connection] Connection attempt failed: %v", err)
		time.Sleep(5 * time.Second)
	}
	return nil, rabbitURI, err
}
