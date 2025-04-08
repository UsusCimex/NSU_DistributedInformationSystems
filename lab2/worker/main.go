package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
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
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Worker %s: ошибка регистрации consumer: %v", workerId, err)
	}

	var wg sync.WaitGroup
	for d := range msgs {
		var taskMsg TaskMessage
		if err := json.Unmarshal(d.Body, &taskMsg); err != nil {
			log.Printf("Worker %s: ошибка декодирования сообщения: %v", workerId, err)
			d.Nack(false, false)
			continue
		}
		wg.Add(1)
		go processTask(d, taskMsg, &wg)
	}
	wg.Wait()
}
