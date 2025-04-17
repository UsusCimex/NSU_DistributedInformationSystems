package connection

import (
	"os"

	"common/amqputil"
	"common/constants"
	"common/logger"
	"common/mongodb"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo"
)

// ConnectMongoDB устанавливает соединение с MongoDB и возвращает клиент и базу данных.
func ConnectMongoDB() (*mongo.Client, *mongo.Database, error) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = constants.DefaultMongoURI
	}
	client, db, err := mongodb.ConnectMongo(mongoURI, constants.DefaultDBName)
	if err != nil {
		return nil, nil, err
	}
	logger.Log("Connector", "Соединение с MongoDB установлено")
	return client, db, nil
}

// ConnectRabbitMQ устанавливает соединение с RabbitMQ и открывает канал для очереди "tasks".
func ConnectRabbitMQ() (*amqp.Connection, *amqp.Channel, string, error) {
	rabbitURI := os.Getenv("RABBITMQ_URI")
	if rabbitURI == "" {
		rabbitURI = constants.DefaultRabbitURI
	}
	conn, err := amqputil.ConnectRabbitMQ(rabbitURI)
	if err != nil {
		return nil, nil, rabbitURI, err
	}
	ch, err := amqputil.CreateChannel(conn, constants.TasksQueue, 0)
	if err != nil {
		conn.Close()
		return nil, nil, rabbitURI, err
	}
	logger.Log("Connector", "Соединение с RabbitMQ установлено")
	return conn, ch, rabbitURI, nil
}
