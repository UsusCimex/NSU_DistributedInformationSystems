package main

import (
	"log"
	"sync"
	"time"

	"worker/internal/processor"
	"worker/internal/rabbitmq"

	"github.com/streadway/amqp"
)

func main() {
	log.Printf("[Worker]: запускается")

	conn, err := connectRabbitMQ()
	if err != nil {
		log.Fatalf("[Worker]: ошибка подключения к RabbitMQ: %v", err)
	}
	defer conn.Close()

	proc := processor.NewProcessor(nil)

	for {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 3) // ограничение одновременной обработки до 3 задач
		log.Printf("[Worker]: запускается consumer")
		rabbitmq.ConsumeTasks(conn, proc, sem, &wg)
		wg.Wait()
		log.Printf("[Worker]: consumer завершился, перезапуск через 5 секунд...")
		time.Sleep(5 * time.Second)
	}
}

func connectRabbitMQ() (*amqp.Connection, error) {
	const maxRetries = 10
	var conn *amqp.Connection
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err == nil {
			log.Printf("[Worker]: подключение к RabbitMQ установлено")
			return conn, nil
		}
		log.Printf("[Worker]: попытка подключения к RabbitMQ: %v", err)
		time.Sleep(5 * time.Second)
	}
	return nil, err
}
