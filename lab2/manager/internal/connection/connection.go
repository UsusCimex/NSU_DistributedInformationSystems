package connection

import (
	"os"

	"common/amqputil"
	"common/logger"
	"common/mongodb"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo"
)

// Параметры соединений по умолчанию
const (
	defaultMongoURI  = "mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0"
	defaultRabbitURI = "amqp://guest:guest@rabbitmq:5672/"
	defaultDBName    = "hash_cracker"
)

// ConnectMongoDB устанавливает соединение с MongoDB и возвращает клиент и базу данных.
func ConnectMongoDB() (*mongo.Client, *mongo.Database, error) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = defaultMongoURI
	}
	client, db, err := mongodb.ConnectMongo(mongoURI, defaultDBName)
	if err != nil {
		return nil, nil, err
	}
	logger.Log("Connector", "Соединение с MongoDB установлено")
	return client, db, nil
}

// ConnectRabbitMQ устанавливает соединение с RabbitMQ и открывает канал для очереди "tasks".
// Возвращает соединение, канал и URI, который был использован.
func ConnectRabbitMQ() (*amqp.Connection, *amqp.Channel, string, error) {
	rabbitURI := os.Getenv("RABBITMQ_URI")
	if rabbitURI == "" {
		rabbitURI = defaultRabbitURI
	}
	conn, err := amqputil.ConnectRabbitMQ(rabbitURI)
	if err != nil {
		return nil, nil, rabbitURI, err
	}
	ch, err := amqputil.CreateChannel(conn, "tasks", 0)
	if err != nil {
		conn.Close()
		return nil, nil, rabbitURI, err
	}
	logger.Log("Connector", "Соединение с RabbitMQ установлено")
	return conn, ch, rabbitURI, nil
}
