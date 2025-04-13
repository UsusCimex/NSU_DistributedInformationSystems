package publisher

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"manager/internal/models"
)

// Publisher публикует подзадачи из БД в очередь RabbitMQ.
type Publisher struct {
	Collection          *mongo.Collection
	RabbitMQConn        *amqp.Connection
	RabbitMQChannel     *amqp.Channel
	RabbitmqURL         string
	publishLimit        int
	pollInterval        time.Duration
	reconnectMaxRetries int
}

// NewPublisher создаёт новый Publisher, принимая подключение, канал и строку подключения (rabbit URL).
func NewPublisher(coll *mongo.Collection, conn *amqp.Connection, ch *amqp.Channel, rabbitmqURL string) *Publisher {
	return &Publisher{
		Collection:          coll,
		RabbitMQConn:        conn,
		RabbitMQChannel:     ch,
		RabbitmqURL:         rabbitmqURL,
		publishLimit:        50,
		pollInterval:        1 * time.Second,
		reconnectMaxRetries: 5,
	}
}

// Start запускает периодическую публикацию подзадач.
func (p *Publisher) Start() {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		p.publishBatch()
	}
}

func (p *Publisher) publishBatch() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cursor, err := p.Collection.Find(ctx, bson.M{"subTasks.status": "RECEIVED"})
	if err != nil {
		log.Printf("[Publisher]: Ошибка извлечения задач: %v", err)
		return
	}
	defer cursor.Close(ctx)

	totalPublished := 0
	for cursor.Next(ctx) {
		var task models.HashTask
		if err := cursor.Decode(&task); err != nil {
			log.Printf("[Publisher]: Ошибка декодирования задачи: %v", err)
			continue
		}
		changed := false
		for i, subTask := range task.SubTasks {
			if subTask.Status == "RECEIVED" && totalPublished < p.publishLimit {
				task.SubTasks[i].Status = "PUBLISHED"
				task.SubTasks[i].UpdatedAt = time.Now()

				msg := models.TaskMessage{
					Hash:          subTask.Hash,
					MaxLength:     task.MaxLength,
					SubTaskNumber: subTask.SubTaskNumber,
					SubTaskCount:  task.SubTaskCount,
				}
				data, err := json.Marshal(msg)
				if err != nil {
					log.Printf("[Publisher] %s: Ошибка маршалинга сообщения: %v", task.Hash, err)
					continue
				}
				err = p.RabbitMQChannel.Publish(
					"",
					"tasks",
					false,
					false,
					amqp.Publishing{
						ContentType:  "application/json",
						Body:         data,
						DeliveryMode: amqp.Persistent, // Флаг для персистентного хранения
					},
				)
				if err != nil {
					log.Printf("[Publisher] %s: Ошибка публикации сообщения: %v", task.Hash, err)
					// Если произошла ошибка публикации, пробуем восстановить соединение.
					if recErr := p.Reconnect(); recErr != nil {
						log.Printf("[Publisher] %s: Не удалось восстановить соединение: %v", task.Hash, recErr)
						continue
					}
					continue
				}
				totalPublished++
				changed = true
			}
			if totalPublished >= p.publishLimit {
				break
			}
		}
		if changed {
			_, err = p.Collection.UpdateOne(ctx, bson.M{"requestId": task.RequestId}, bson.M{"$set": bson.M{"subTasks": task.SubTasks}})
			if err != nil {
				log.Printf("[Publisher] %s: Ошибка обновления задачи: %v", task.Hash, err)
			}
		}
		if totalPublished >= p.publishLimit {
			break
		}
	}
	if totalPublished > 0 {
		log.Printf("[Publisher] Опубликовано %d подзадач", totalPublished)
	}
}

// Reconnect пытается создать новый канал из существующего подключения и переобъявить очередь.
func (p *Publisher) Reconnect() error {
	// Если соединение закрыто – восстановим его.
	if p.RabbitMQConn.IsClosed() {
		var newConn *amqp.Connection
		var err error
		for i := 1; i <= p.reconnectMaxRetries; i++ {
			newConn, err = amqp.Dial(p.RabbitmqURL)
			if err != nil {
				log.Printf("[Publisher] Попытка восстановления соединения %d/%d не удалась: %v", i, p.reconnectMaxRetries, err)
				time.Sleep(5 * time.Second)
				continue
			}
			p.RabbitMQConn = newConn
			log.Println("[Publisher] Восстановлено соединение с RabbitMQ")
			break
		}
		if newConn == nil {
			return err
		}
	}

	// Создаём новый канал
	var ch *amqp.Channel
	var err error
	for i := 1; i <= p.reconnectMaxRetries; i++ {
		ch, err = p.RabbitMQConn.Channel()
		if err != nil {
			log.Printf("[Publisher] Попытка восстановления канала %d/%d не удалась: %v", i, p.reconnectMaxRetries, err)
			time.Sleep(5 * time.Second)
			continue
		}
		// Переобъявляем очередь "tasks"
		_, err = ch.QueueDeclare("tasks", true, false, false, false, nil)
		if err != nil {
			log.Printf("[Publisher] Ошибка объявления очереди при восстановлении канала %d/%d: %v", i, p.reconnectMaxRetries, err)
			ch.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		p.RabbitMQChannel = ch
		log.Println("[Publisher] Восстановлено соединение с каналом RabbitMQ")
		return nil
	}
	return err
}
