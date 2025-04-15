package rabbit

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"common/amqputil"
	"common/models"
	"manager/internal/processor"
)

func StartPublisher(coll *mongo.Collection, conn **amqp.Connection, rabbitURI string, ch *amqp.Channel) {
	publishLimit := 100

	for {
		// Создаём контекст с 10-секундным таймаутом для выборки задач.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cursor, err := coll.Find(ctx, models.BsonFilterReceived())
		if err != nil {
			log.Printf("[Rabbit Publisher] Error fetching tasks: %v", err)
			cancel()
			time.Sleep(5 * time.Second)
			continue
		}

		publishedCount := 0

		// Обходим все найденные задачи.
		for cursor.Next(ctx) {
			var task models.HashTask
			if err := cursor.Decode(&task); err != nil {
				log.Printf("[Rabbit Publisher] Error decoding task: %v", err)
				continue
			}

			taskUpdated := false
			// Проходим по подзадачам по индексам, чтобы изменения сохранялись в самом срезе.
			for i := range task.SubTasks {
				if task.SubTasks[i].Status != "RECEIVED" {
					continue
				}

				// Если достигнут лимит публикаций, выходим из цикла.
				if publishedCount >= publishLimit {
					break
				}

				// Формируем сообщение для очереди.
				msg := models.TaskMessage{
					Hash:          task.SubTasks[i].Hash,
					MaxLength:     task.MaxLength,
					SubTaskNumber: task.SubTasks[i].SubTaskNumber,
					SubTaskCount:  task.SubTaskCount,
				}
				data, err := json.Marshal(msg)
				if err != nil {
					log.Printf("[Rabbit Publisher] Marshal error for task %s, subtask %d: %v", task.RequestId, task.SubTasks[i].SubTaskNumber, err)
					continue
				}

				// Публикуем сообщение в очередь RabbitMQ.
				err = ch.Publish(
					"",      // exchange (пустая строка означает стандартный обменник)
					"tasks", // имя очереди
					false,
					false,
					amqp.Publishing{
						ContentType:  "application/json",
						Body:         data,
						DeliveryMode: amqp.Persistent,
					},
				)
				if err != nil {
					log.Printf("[Rabbit Publisher] Publish error for task %s, subtask %d: %v", task.RequestId, task.SubTasks[i].SubTaskNumber, err)
					// При ошибке публикации пытаемся восстановить соединение и канал.
					newCh, recErr := amqputil.Reconnect(conn, rabbitURI, "tasks", 0, amqputil.DefaultMaxRetries)
					if recErr != nil {
						log.Printf("[Rabbit Publisher] Reconnect failed for task %s: %v", task.RequestId, recErr)
					} else {
						ch = newCh
					}
					// Прерываем цикл подзадач для данной задачи при ошибке.
					break
				}

				// Обновляем статус подзадачи в структуре.
				task.SubTasks[i].Status = "PUBLISHED"
				task.SubTasks[i].UpdatedAt = time.Now()
				publishedCount++
				taskUpdated = true
			}

			// Если были изменения в подзадачах, обновляем документ в MongoDB.
			if taskUpdated {
				_, err = coll.UpdateOne(ctx, bson.M{"requestId": task.RequestId}, bson.M{"$set": bson.M{"subTasks": task.SubTasks}})
				if err != nil {
					log.Printf("[Rabbit Publisher] Error updating task %s: %v", task.RequestId, err)
				}
			}
			if publishedCount >= publishLimit {
				break
			}
		}

		cursor.Close(ctx)
		cancel()

		if publishedCount > 0 {
			log.Printf("[Rabbit Publisher] Published messages for %d subtasks", publishedCount)
		}

		// Задержка перед следующим циклом.
		time.Sleep(2 * time.Second)
	}
}

// StartResultConsumer регистрирует consumer для очереди "results" и обрабатывает входящие сообщения.
// В случае ошибок объявление очереди или регистрации consumer-а вызывает реконнект.
func StartResultConsumer(ch *amqp.Channel, coll *mongo.Collection, conn **amqp.Connection, dialURL string) {
	for {
		_, err := ch.QueueDeclare("results", true, false, false, false, nil)
		if err != nil {
			log.Printf("[Rabbit Consumer] Error declaring results queue: %v", err)
			newCh, recErr := amqputil.Reconnect(conn, dialURL, "results", 0, 5)
			if recErr != nil {
				log.Printf("[Rabbit Consumer] Reconnect failed: %v", recErr)
				time.Sleep(5 * time.Second)
				continue
			}
			ch = newCh
			continue
		}

		msgs, err := ch.Consume("results", "", false, false, false, false, nil)
		if err != nil {
			log.Printf("[Rabbit Consumer] Error registering consumer: %v", err)
			newCh, recErr := amqputil.Reconnect(conn, dialURL, "results", 0, 5)
			if recErr != nil {
				log.Printf("[Rabbit Consumer] Reconnect failed: %v", recErr)
				time.Sleep(5 * time.Second)
				continue
			}
			ch = newCh
			continue
		}
		log.Println("[Rabbit Consumer] Consumer for results registered")
		processResults(msgs, coll, conn, dialURL)
		// Если processResults завершился, значит канал закрыт – повторяем цикл реконнекта.
		log.Println("[Rabbit Consumer] Consumer ended. Reconnecting...")
	}
}

// processResults обрабатывает сообщения и если канал закрывается, выходит для реконнекта.
func processResults(msgs <-chan amqp.Delivery, coll *mongo.Collection, conn **amqp.Connection, dialURL string) {
	for m := range msgs {
		var res models.ResultMessage
		if err := json.Unmarshal(m.Body, &res); err != nil {
			log.Printf("[Rabbit Consumer] Unmarshal error: %v", err)
			m.Ack(false)
			continue
		}
		log.Printf("[Rabbit Consumer] Received result for hash %s: %s", res.Hash, res.Result)
		err := processor.ProcessResult(res, coll)
		if err != nil {
			log.Printf("[Rabbit Consumer] Failed to process result for hash %s: %v", res.Hash, err)
		}
		m.Ack(false)
	}
	log.Println("[Rabbit Consumer] Message channel closed.")
}
