package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"common/amqputil"
	"common/logger"
	"common/models"
	"manager/internal/processor"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func StartPublisher(coll *mongo.Collection, conn **amqp.Connection, rabbitURI string, ch *amqp.Channel) {
	publishLimit := 100

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cursor, err := coll.Find(ctx, models.BsonFilterReceived())
		if err != nil {
			logger.Log("Publisher", fmt.Sprintf("Ошибка получения задач: %v", err))
			cancel()
			time.Sleep(5 * time.Second)
			continue
		}

		publishedCount := 0
		for cursor.Next(ctx) {
			var task models.HashTask
			if err := cursor.Decode(&task); err != nil {
				logger.Log("Publisher", fmt.Sprintf("Ошибка декодирования задачи: %v", err))
				continue
			}

			taskUpdated := false
			for i := range task.SubTasks {
				if task.SubTasks[i].Status != "RECEIVED" {
					continue
				}
				if publishedCount >= publishLimit {
					break
				}

				msg := models.TaskMessage{
					Hash:          task.SubTasks[i].Hash,
					MaxLength:     task.MaxLength,
					SubTaskNumber: task.SubTasks[i].SubTaskNumber,
					SubTaskCount:  task.SubTaskCount,
				}
				data, err := json.Marshal(msg)
				if err != nil {
					logger.LogTask("Publisher", task.SubTasks[i].Hash, task.SubTasks[i].SubTaskNumber, task.SubTaskCount, fmt.Sprintf("Ошибка маршалинга: %v", err))
					continue
				}

				err = ch.Publish(
					"",
					"tasks",
					false,
					false,
					amqp.Publishing{
						ContentType:  "application/json",
						Body:         data,
						DeliveryMode: amqp.Persistent,
					},
				)
				if err != nil {
					logger.LogTask("Publisher", task.SubTasks[i].Hash, task.SubTasks[i].SubTaskNumber, task.SubTaskCount, fmt.Sprintf("Ошибка публикации: %v", err))
					newCh, recErr := amqputil.Reconnect(conn, rabbitURI, "tasks", 0, amqputil.DefaultMaxRetries)
					if recErr != nil {
						logger.Log("Publisher", fmt.Sprintf("Реконнект не удался для задачи %s: %v", task.RequestId, recErr))
					} else {
						ch = newCh
					}
					break
				}

				task.SubTasks[i].Status = "PUBLISHED"
				task.SubTasks[i].UpdatedAt = time.Now()
				publishedCount++
				taskUpdated = true
			}

			if taskUpdated {
				_, err = coll.UpdateOne(ctx, bson.M{"requestId": task.RequestId}, bson.M{"$set": bson.M{"subTasks": task.SubTasks}})
				if err != nil {
					logger.Log("Publisher", fmt.Sprintf("Ошибка обновления задачи %s: %v", task.RequestId, err))
				}
			}
			if publishedCount >= publishLimit {
				break
			}
		}

		cursor.Close(ctx)
		cancel()

		if publishedCount > 0 {
			logger.Log("Publisher", fmt.Sprintf("Опубликовано сообщений для %d подзадач", publishedCount))
		}
		time.Sleep(2 * time.Second)
	}
}

func StartResultConsumer(ch *amqp.Channel, coll *mongo.Collection, conn **amqp.Connection, dialURL string) {
	for {
		_, err := ch.QueueDeclare("results", true, false, false, false, nil)
		if err != nil {
			logger.Log("Consumer", fmt.Sprintf("Ошибка объявления очереди results: %v", err))
			newCh, recErr := amqputil.Reconnect(conn, dialURL, "results", 0, 5)
			if recErr != nil {
				logger.Log("Consumer", fmt.Sprintf("Реконнект не удался: %v", recErr))
				time.Sleep(5 * time.Second)
				continue
			}
			ch = newCh
			continue
		}

		msgs, err := ch.Consume("results", "", false, false, false, false, nil)
		if err != nil {
			logger.Log("Consumer", fmt.Sprintf("Ошибка регистрации consumer: %v", err))
			newCh, recErr := amqputil.Reconnect(conn, dialURL, "results", 0, 5)
			if recErr != nil {
				logger.Log("Consumer", fmt.Sprintf("Реконнект не удался: %v", recErr))
				time.Sleep(5 * time.Second)
				continue
			}
			ch = newCh
			continue
		}
		logger.Log("Consumer", "Consumer для results зарегистрирован")
		processResults(msgs, coll, conn, dialURL)
		logger.Log("Consumer", "Consumer завершился. Переподключаемся...")
	}
}

func processResults(msgs <-chan amqp.Delivery, coll *mongo.Collection, conn **amqp.Connection, dialURL string) {
	for m := range msgs {
		var res models.ResultMessage
		if err := json.Unmarshal(m.Body, &res); err != nil {
			logger.Log("Consumer", fmt.Sprintf("Ошибка декодирования: %v", err))
			m.Ack(false)
			continue
		}
		var task models.HashTask
		if err := coll.FindOne(context.Background(), bson.M{"hash": res.Hash}).Decode(&task); err != nil {
			logger.LogHash("Consumer", res.Hash, "Задача не найдена")
			m.Ack(false)
			continue
		}
		logger.LogTask("Consumer", res.Hash, res.SubTaskNumber, task.SubTaskCount, fmt.Sprintf("Получен результат: %s", res.Result))
		err := processor.ProcessResult(res, task, coll)
		if err != nil {
			logger.LogTask("Consumer", res.Hash, res.SubTaskNumber, task.SubTaskCount, fmt.Sprintf("Ошибка обработки результата: %v", err))
		}
		m.Ack(false)
	}
	logger.Log("Consumer", "Канал сообщений закрыт")
}
