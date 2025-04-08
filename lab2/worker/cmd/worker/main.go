package main

import (
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/streadway/amqp"

	"worker/internal/mongodb"
	"worker/internal/processor"
	"worker/internal/rabbitmq"
)

func main() {
	workerID := uuid.New().String()

	// Подключение к MongoDB
	mongoClient, db, err := mongodb.ConnectMongo("mongodb://mongo:27017", "hash_cracker")
	if err != nil {
		log.Fatalf("Worker %s: ошибка подключения к MongoDB: %v", workerID[:5], err)
	}
	defer mongoClient.Disconnect(nil)
	hashTaskCollection := db.Collection("hash_tasks")
	log.Printf("Worker %s: подключение к MongoDB установлено", workerID[:5])

	// Подключение к RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Worker %s: ошибка подключения к RabbitMQ: %v", workerID[:5], err)
	}
	defer conn.Close()
	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Worker %s: ошибка открытия канала RabbitMQ: %v", workerID[:5], err)
	}
	_, err = channel.QueueDeclare(
		"tasks",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Worker %s: ошибка объявления очереди tasks: %v", workerID[:5], err)
	}
	// Устанавливаем QoS: не более 5 сообщений одновременно
	if err = channel.Qos(5, 0, false); err != nil {
		log.Fatalf("Worker %s: ошибка установки QoS: %v", workerID[:5], err)
	}

	// Создание экземпляра процессора
	proc := processor.NewProcessor(workerID, hashTaskCollection, channel)

	// Запуск обновления heartbeat в отдельной горутине
	hbUpdater := processor.NewHeartbeatUpdater(workerID, hashTaskCollection)
	go hbUpdater.Start()

	// Семафор для ограничения одновременной обработки (не более 5 задач)
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	// Запуск потребителя RabbitMQ
	rabbitmq.ConsumeTasks(channel, proc, sem, &wg)
	wg.Wait()
}
