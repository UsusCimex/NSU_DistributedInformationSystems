package connection

import (
	"log"
	"os"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo"

	"common/amqputil"
	"common/mongodb"
)

// ConnectMongoDB устанавливает подключение к MongoDB и возвращает клиента и базу данных.
func ConnectMongoDB() (*mongo.Client, *mongo.Database, error) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		// Если не задана переменная, используем URI для replica set с 3 узлами.
		mongoURI = "mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0"
	}
	client, db, err := mongodb.ConnectMongo(mongoURI, "hash_cracker")
	if err != nil {
		return nil, nil, err
	}
	log.Println("[Connection] Connected to MongoDB")
	return client, db, nil
}

// ConnectRabbitMQ устанавливает подключение к RabbitMQ и возвращает соединение, канал и URI.
func ConnectRabbitMQ() (*amqp.Connection, *amqp.Channel, string, error) {
	rabbitURI := os.Getenv("RABBITMQ_URI")
	if rabbitURI == "" {
		rabbitURI = "amqp://guest:guest@rabbitmq:5672/"
	}
	conn, err := amqp.Dial(rabbitURI)
	if err != nil {
		return nil, nil, "", err
	}
	ch, err := amqputil.CreateChannel(conn, "tasks", 0)
	if err != nil {
		conn.Close()
		return nil, nil, "", err
	}
	log.Println("[Connection] Connected to RabbitMQ")
	return conn, ch, rabbitURI, nil
}
