package main

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"worker/internal/processor"
	"worker/internal/rabbitmq"

	"github.com/streadway/amqp"
)

func main() {
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = "worker-" + strconv.Itoa(os.Getpid())
	}
	log.Printf("%s: запускается", workerID)

	conn, err := connectRabbitMQ(workerID)
	if err != nil {
		log.Fatalf("%s: ошибка подключения к RabbitMQ: %v", workerID, err)
	}
	defer conn.Close()

	proc := processor.NewProcessor(workerID, nil)

	for {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 3) // ограничение одновременной обработки до 3 задач
		log.Printf("%s: запускается consumer", workerID)
		rabbitmq.ConsumeTasks(conn, proc, sem, &wg)
		wg.Wait()
		log.Printf("%s: consumer завершился, перезапуск через 5 секунд...", workerID)
		time.Sleep(5 * time.Second)
	}
}

func connectRabbitMQ(workerID string) (*amqp.Connection, error) {
	const maxRetries = 10
	var conn *amqp.Connection
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err == nil {
			log.Printf("%s: подключение к RabbitMQ установлено", workerID)
			return conn, nil
		}
		log.Printf("%s: попытка %d: ошибка подключения к RabbitMQ: %v", workerID, i+1, err)
		time.Sleep(5 * time.Second)
	}
	return nil, err
}
