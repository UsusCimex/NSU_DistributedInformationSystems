package mongodb

import (
	"context"
	"fmt"
	"time"

	"common/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectMongo устанавливает подключение к MongoDB и возвращает клиента и базу данных.
func ConnectMongo(uri, dbName string) (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = client.Connect(ctx); err != nil {
		return nil, nil, err
	}
	db := client.Database(dbName)
	createIndex(ctx, db, "hash_tasks", "hash")
	createIndex(ctx, db, "hash_tasks", "requestId")
	return client, db, nil
}

func createIndex(ctx context.Context, db *mongo.Database, collName, field string) {
	index := mongo.IndexModel{
		Keys:    bson.D{{Key: field, Value: 1}},
		Options: options.Index().SetBackground(true),
	}
	if _, err := db.Collection(collName).Indexes().CreateOne(ctx, index); err != nil {
		logger.Log("MongoDB", fmt.Sprintf("Ошибка создания индекса на поле %s: %v", field, err))
	}
}
