package main

import (
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	workerId           string
	mongoClient        *mongo.Client
	db                 *mongo.Database
	hashTaskCollection *mongo.Collection
	rabbitMQChannel    *amqp.Channel
)
