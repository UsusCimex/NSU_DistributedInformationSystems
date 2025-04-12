package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"manager/internal/api"
	"manager/internal/publisher"
	"manager/internal/receiver"
	"manager/internal/requeue"
)

var (
	hashTaskCollection *mongo.Collection
	rabbitMQChannel    *amqp.Channel
)

func main() {
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI("mongodb://mongo:27017"))
	if err != nil {
		log.Fatalf("Ошибка создания клиента MongoDB: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Fatalf("Ошибка подключения к MongoDB: %v", err)
	}

	db := mongoClient.Database("hash_cracker")
	hashTaskCollection = db.Collection("hash_tasks")
	log.Println("Подключение к MongoDB успешно")

	// Создание индекса по полю "hash"
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "hash", Value: 1}},
		Options: options.Index().SetBackground(true),
	}
	_, err = hashTaskCollection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Ошибка создания индекса по полю hash: %v", err)
	} else {
		log.Println("Индекс по полю hash успешно создан")
	}

	// Создание индекса по полю "requestId"
	indexModel2 := mongo.IndexModel{
		Keys:    bson.D{{Key: "requestId", Value: 1}},
		Options: options.Index().SetBackground(true),
	}
	_, err = hashTaskCollection.Indexes().CreateOne(ctx, indexModel2)
	if err != nil {
		log.Printf("Ошибка создания индекса по полю requestId: %v", err)
	} else {
		log.Println("Индекс по полю requestId успешно создан")
	}

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	defer conn.Close()

	rabbitMQChannel, err = conn.Channel()
	if err != nil {
		log.Fatalf("Ошибка открытия канала RabbitMQ: %v", err)
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
		log.Fatalf("Ошибка объявления очереди tasks: %v", err)
	}

	log.Println("Подключение к RabbitMQ успешно")

	// Инициализируем пакеты, передавая им зависимости
	publisher.Init(hashTaskCollection, rabbitMQChannel)
	requeue.Init(hashTaskCollection)
	receiver.Init(hashTaskCollection, rabbitMQChannel)
	api.Init(hashTaskCollection)

	// Запускаем фоновые процессы
	go publisher.StartPublisherLoop()
	go requeue.StartRequeueChecker()
	go receiver.StartResultConsumer()

	// Регистрируем HTTP-обработчики
	http.HandleFunc("/api/hash/crack", api.HandleCrack)
	http.HandleFunc("/api/hash/status", api.HandleStatus)

	log.Println("Сервис Manager запущен на порту 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Ошибка HTTP-сервера: %v", err)
	}
}
