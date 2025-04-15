package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectMongo подключается к экземпляру MongoDB по указанному URI и имени базы данных.
// Возвращает клиент и объект базы данных в случае успеха.
func ConnectMongo(uri string, dbName string) (*mongo.Client, *mongo.Database, error) {
	// Устанавливаем таймаут подключения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Подготавливаем параметры клиента и подключаемся
	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, err
	}
	// Проверяем соединение
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, err
	}
	db := client.Database(dbName)
	return client, db, nil
}
