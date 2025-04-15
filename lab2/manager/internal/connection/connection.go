package connection

import (
	"os"

	"common/amqputil"
	"common/logger"
	"common/mongodb"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo"
)

func ConnectMongoDB() (*mongo.Client, *mongo.Database, error) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0"
	}
	client, db, err := mongodb.ConnectMongo(mongoURI, "hash_cracker")
	if err != nil {
		return nil, nil, err
	}
	logger.Log("Connector", "Подключение к MongoDB установлено")
	return client, db, nil
}

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
	logger.Log("Connector", "Подключение к RabbitMQ установлено")
	return conn, ch, rabbitURI, nil
}
