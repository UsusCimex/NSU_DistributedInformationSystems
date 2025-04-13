package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/streadway/amqp"

	"manager/internal/api"
	"manager/internal/mongodb"
	"manager/internal/publisher"
	"manager/internal/receiver"

	"go.mongodb.org/mongo-driver/mongo"
)

// ManagerApp инкапсулирует все основные зависимости сервиса.
type ManagerApp struct {
	MongoClient *mongo.Client
	DB          *mongo.Database
	RMQConn     *amqp.Connection
	RMQChannel  *amqp.Channel
	Publisher   *publisher.Publisher
}

func main() {
	app, err := initializeManagerApp()
	if err != nil {
		log.Fatalf("Ошибка инициализации: %v", err)
	}
	defer app.MongoClient.Disconnect(context.Background())
	defer app.RMQConn.Close()

	// Запуск consumer-а для результатов.
	receiver.StartResultConsumer(app.RMQChannel, app.DB.Collection("hash_tasks"))

	// Запуск публикации подзадач.
	go app.Publisher.Start()

	// Запуск HTTP сервера.
	startHTTPServer(app.DB.Collection("hash_tasks"))

	// Можно использовать синхронизацию/обработку сигналов для graceful shutdown.
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

func initializeManagerApp() (*ManagerApp, error) {
	// Получаем строку подключения из переменной окружения.
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	// Подключение к MongoDB
	client, db, err := mongodb.ConnectMongo(uri, "hash_cracker")
	if err != nil {
		return nil, err
	}
	log.Println("Подключение к MongoDB успешно")

	// Подключение к RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	// Объявление очереди "tasks".
	_, err = channel.QueueDeclare("tasks", true, false, false, false, nil)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, err
	}
	log.Println("Подключение к RabbitMQ успешно")

	pub := publisher.NewPublisher(db.Collection("hash_tasks"), channel)

	return &ManagerApp{
		MongoClient: client,
		DB:          db,
		RMQConn:     conn,
		RMQChannel:  channel,
		Publisher:   pub,
	}, nil
}

func startHTTPServer(coll *mongo.Collection) {
	mux := http.NewServeMux()
	api.RegisterHandlers(mux, coll)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf("Сервис Manager запущен на порту %s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Ошибка HTTP-сервера: %v", err)
	}
}
