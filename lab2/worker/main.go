package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	workerId = os.Getenv("WORKER_ID")
	if workerId == "" {
		workerId = "worker-" + strconv.Itoa(os.Getpid())
	}

	var err error
	mongoClient, err = mongo.NewClient(options.Client().ApplyURI("mongodb://mongo:27017"))
	if err != nil {
		log.Fatalf("Worker %s: ошибка создания MongoDB клиента: %v", workerId, err)
	}
	// Используем context.Background для подключения
	ctx := context.Background()
	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Fatalf("Worker %s: ошибка подключения к MongoDB: %v", workerId, err)
	}
	db = mongoClient.Database("hash_cracker")
	hashTaskCollection = db.Collection("hash_tasks")
	log.Printf("Worker %s: подключение к MongoDB установлено", workerId)

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Worker %s: ошибка подключения к RabbitMQ: %v", workerId, err)
	}
	defer conn.Close()
	rabbitMQChannel, err = conn.Channel()
	if err != nil {
		log.Fatalf("Worker %s: ошибка открытия канала RabbitMQ: %v", workerId, err)
	}
	_, err = rabbitMQChannel.QueueDeclare(
		"tasks",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Worker %s: ошибка объявления очереди tasks: %v", workerId, err)
	}

	go heartbeatUpdater()

	msgs, err := rabbitMQChannel.Consume(
		"tasks",
		"",
		false, // auto-ack отключён
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Worker %s: ошибка регистрации consumer: %v", workerId, err)
	}

	// Семафор для ограничения количества одновременно обрабатываемых задач (не более 5)
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for d := range msgs {
		sem <- struct{}{} // Блокируется, если уже 5 задач обрабатываются
		wg.Add(1)
		go func(d amqp.Delivery) {
			defer wg.Done()
			// После завершения обработки освобождаем слот
			defer func() { <-sem }()
			var taskMsg TaskMessage
			if err := json.Unmarshal(d.Body, &taskMsg); err != nil {
				log.Printf("Worker %s: ошибка декодирования сообщения: %v", workerId, err)
				d.Nack(false, false)
				return
			}
			// processTask теперь использует context.Background для долгих вычислений
			processTask(d, taskMsg, &wg)
		}(d)
	}
	wg.Wait()
}
