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

// StartPublisher проверяет базу данных на наличие задач с подзадачами в статусе "RECEIVED" и публикует их в очередь "tasks".
func StartPublisher(coll *mongo.Collection, connPtr **amqp.Connection, rabbitURI string, ch *amqp.Channel) {
	const publishLimit = 100 // максимальное количество подзадач для публикации за один цикл
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cursor, err := coll.Find(ctx, models.BsonFilterReceived())
		if err != nil {
			cancel()
			logger.Log("Publisher", fmt.Sprintf("Ошибка получения задач: %v", err))
			time.Sleep(5 * time.Second)
			continue
		}
		publishedCount := 0
		var task models.HashTask
		for cursor.Next(ctx) {
			if err := cursor.Decode(&task); err != nil {
				logger.Log("Publisher", fmt.Sprintf("Ошибка декодирования задачи: %v", err))
				continue
			}
			taskUpdated := false
			// Перебираем подзадачи этой задачи
			for i := range task.SubTasks {
				subTask := &task.SubTasks[i]
				if subTask.Status != "RECEIVED" {
					continue // публикуем только подзадачи, ожидающие публикации
				}
				if publishedCount >= publishLimit {
					break // достигли лимита публикации для этого цикла
				}
				// Создаем сообщение для этой подзадачи
				msg := models.TaskMessage{
					Hash:          subTask.Hash,
					MaxLength:     task.MaxLength,
					SubTaskNumber: subTask.SubTaskNumber,
					SubTaskCount:  task.SubTaskCount,
				}
				data, err := json.Marshal(msg)
				if err != nil {
					logger.LogTask("Publisher", subTask.Hash, subTask.SubTaskNumber, task.SubTaskCount,
						fmt.Sprintf("Ошибка маршалинга: %v", err))
					continue
				}
				// Публикуем сообщение в очередь "tasks"
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
					// Логируем ошибку и пытаемся переподключиться к RabbitMQ (получить новый канал)
					logger.LogTask("Publisher", subTask.Hash, subTask.SubTaskNumber, task.SubTaskCount,
						fmt.Sprintf("Ошибка публикации: %v", err))
					newCh, recErr := amqputil.Reconnect(connPtr, rabbitURI, "tasks", 0, amqputil.DefaultMaxRetries)
					if recErr != nil {
						logger.Log("Publisher", fmt.Sprintf("Не удалось восстановить соединение RabbitMQ для задачи %s: %v", task.RequestId, recErr))
					} else {
						ch = newCh
					}
					break
				}
				// Отмечаем подзадачу как опубликованную
				subTask.Status = "PUBLISHED"
				subTask.UpdatedAt = time.Now()
				publishedCount++
				taskUpdated = true
			}
			if taskUpdated {
				// Сохраняем обновленные статусы подзадач в базе данных
				_, err := coll.UpdateOne(ctx,
					bson.M{"requestId": task.RequestId},
					bson.M{"$set": bson.M{"subTasks": task.SubTasks}},
				)
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
			logger.Log("Publisher", fmt.Sprintf("Опубликовано %d подзадач(и) в очередь", publishedCount))
		}
		time.Sleep(2 * time.Second)
	}
}

// StartResultConsumer слушает очередь "results" для получения результатов подзадач и обновляет базу данных соответствующим образом.
func StartResultConsumer(ch *amqp.Channel, coll *mongo.Collection, connPtr **amqp.Connection, rabbitURI string) {
	for {
		// Проверяем существование очереди "results"
		_, err := ch.QueueDeclare("results", true, false, false, false, nil)
		if err != nil {
			logger.Log("Consumer", fmt.Sprintf("Ошибка объявления очереди 'results': %v", err))
			// Пытаемся переподключиться если объявление очереди не удалось
			newCh, recErr := amqputil.Reconnect(connPtr, rabbitURI, "results", 0, 5)
			if recErr != nil {
				logger.Log("Consumer", fmt.Sprintf("Не удалось восстановить подключение RabbitMQ: %v", recErr))
				time.Sleep(5 * time.Second)
				continue
			}
			ch = newCh
			continue
		}
		// Регистрируем потребителя для очереди "results"
		msgs, err := ch.Consume("results", "", false, false, false, false, nil)
		if err != nil {
			logger.Log("Consumer", fmt.Sprintf("Ошибка регистрации consumer на очереди 'results': %v", err))
			// Пытаемся переподключиться если потребление не удалось
			newCh, recErr := amqputil.Reconnect(connPtr, rabbitURI, "results", 0, 5)
			if recErr != nil {
				logger.Log("Consumer", fmt.Sprintf("Не удалось восстановить подключение RabbitMQ: %v", recErr))
				time.Sleep(5 * time.Second)
				continue
			}
			ch = newCh
			continue
		}
		logger.Log("Consumer", "Consumer для очереди 'results' запущен")
		// Обрабатываем сообщения из очереди "results"
		processResults(msgs, coll)
		logger.Log("Consumer", "Обработка результатов завершена, перезапуск consumer...")
	}
}

// processResults читает сообщения из канала results и обновляет задачи в базе данных для каждого результата.
func processResults(msgs <-chan amqp.Delivery, coll *mongo.Collection) {
	for msg := range msgs {
		// Разбираем сообщение с результатом
		var res models.ResultMessage
		if err := json.Unmarshal(msg.Body, &res); err != nil {
			logger.Log("Consumer", fmt.Sprintf("Ошибка декодирования результата: %v", err))
			msg.Ack(false) // подтверждаем плохое сообщение для удаления из очереди
			continue
		}
		// Находим соответствующую задачу в базе данных по хешу
		var task models.HashTask
		err := coll.FindOne(context.Background(), bson.M{"hash": res.Hash}).Decode(&task)
		if err != nil {
			logger.LogHash("Consumer", res.Hash, "Задача для данного хэша не найдена")
			msg.Ack(false)
			continue
		}
		// Логируем полученный результат
		logger.LogTask("Consumer", res.Hash, res.SubTaskNumber, task.SubTaskCount, fmt.Sprintf("Получен результат: %s", res.Result))
		// Обрабатываем результат: обновляем статус подзадачи, статус задачи и т.д.
		if err := processor.ProcessResult(res, task, coll); err != nil {
			logger.LogTask("Consumer", res.Hash, res.SubTaskNumber, task.SubTaskCount, fmt.Sprintf("Ошибка обновления задачи: %v", err))
		}
		msg.Ack(false) // подтверждаем обработку сообщения
	}
	logger.Log("Consumer", "Канал результатов закрыт")
}
