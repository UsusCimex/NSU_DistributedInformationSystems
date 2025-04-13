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
	coll         *mongo.Collection
	rmqChannel   *amqp.Channel
	publishLimit int
	pollInterval time.Duration
}

func NewPublisher(coll *mongo.Collection, ch *amqp.Channel) *Publisher {
	return &Publisher{
		coll:         coll,
		rmqChannel:   ch,
		publishLimit: 100,
		pollInterval: 1 * time.Second,
	}
}

func (p *Publisher) Start() {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		p.publishBatch()
	}
}

func (p *Publisher) publishBatch() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := p.coll.Find(ctx, bson.M{"subTasks.status": "RECEIVED"})
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
				if err = p.rmqChannel.Publish(
					"",
					"tasks",
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        data,
					},
				); err != nil {
					log.Printf("[Publisher] %s: Ошибка публикации сообщения: %v", task.Hash, err)
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
			_, err = p.coll.UpdateOne(ctx, bson.M{"requestId": task.RequestId}, bson.M{"$set": bson.M{"subTasks": task.SubTasks}})
			if err != nil {
				log.Printf("[Publisher] %s: Ошибка обновления задачи: %v", task.Hash, err)
			}
		}
		if totalPublished >= p.publishLimit {
			break
		}
	}
	if totalPublished > 0 {
		log.Printf("[Publisher]: Опубликовано %d подзадач", totalPublished)
	}
}
