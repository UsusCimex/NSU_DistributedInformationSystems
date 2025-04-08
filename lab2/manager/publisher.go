package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
)

// TaskMessage теперь не содержит RequestId, а только необходимые поля.
type TaskMessage struct {
	Hash          string `json:"hash"`
	MaxLength     int    `json:"maxLength"`
	SubTaskNumber int    `json:"subTaskNumber"`
	SubTaskCount  int    `json:"subTaskCount"`
}

func publisherLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cursor, err := hashTaskCollection.Find(ctx, bson.M{"subTasks.status": "RECEIVED"})
		if err != nil {
			log.Printf("Ошибка извлечения задач: %v", err)
			cancel()
			continue
		}

		totalPublished := 0
		// Обходим все найденные hash-задачи.
		for cursor.Next(ctx) {
			var task HashTask
			if err := cursor.Decode(&task); err != nil {
				log.Printf("Ошибка декодирования задачи: %v", err)
				continue
			}

			changed := false
			for i, subTask := range task.SubTasks {
				if subTask.Status == "RECEIVED" && totalPublished < 50 {
					task.SubTasks[i].Status = "PUBLISHED"
					task.SubTasks[i].UpdatedAt = time.Now()

					msg := TaskMessage{
						Hash:          subTask.Hash,
						MaxLength:     task.MaxLength,
						SubTaskNumber: subTask.SubTaskNumber,
						SubTaskCount:  task.SubTaskCount,
					}
					data, err := json.Marshal(msg)
					if err != nil {
						log.Printf("Ошибка маршалинга сообщения: %v", err)
						continue
					}

					err = rabbitMQChannel.Publish(
						"",
						"tasks",
						false,
						false,
						amqp.Publishing{
							ContentType: "application/json",
							Body:        data,
						},
					)
					if err != nil {
						log.Printf("Ошибка публикации сообщения: %v", err)
						continue
					}
					totalPublished++
					changed = true
				}
				if totalPublished >= 50 {
					break
				}
			}

			if changed {
				_, err = hashTaskCollection.UpdateOne(ctx, bson.M{"requestId": task.RequestId}, bson.M{"$set": bson.M{"subTasks": task.SubTasks}})
				if err != nil {
					log.Printf("Ошибка обновления задачи: %v", err)
				}
			}

			if totalPublished >= 50 {
				break
			}
		}
		cursor.Close(ctx)
		cancel()

		if totalPublished > 0 {
			log.Printf("Менеджер: опубликовано %d подзадач", totalPublished)
		}
	}
}
