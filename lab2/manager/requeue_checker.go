package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func requeueChecker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		filter := bson.M{
			"subTasks": bson.M{
				"$elemMatch": bson.M{
					"status":    "WORKING",
					"heartbeat": bson.M{"$lt": time.Now().Add(-20 * time.Second)},
				},
			},
		}

		cursor, err := hashTaskCollection.Find(ctx, filter)
		if err != nil {
			log.Printf("Ошибка извлечения задач для переотправки: %v", err)
			cancel()
			continue
		}
		for cursor.Next(ctx) {
			var task HashTask
			if err := cursor.Decode(&task); err != nil {
				log.Printf("Ошибка декодирования задачи: %v", err)
				continue
			}

			changed := false
			for i, subTask := range task.SubTasks {
				if subTask.Status == "WORKING" && subTask.Heartbeat.Before(time.Now().Add(-20*time.Second)) {
					task.SubTasks[i].Status = "PUBLISHED"
					task.SubTasks[i].UpdatedAt = time.Now()
					changed = true
					log.Printf("Переотправлена подзадача (subTaskNumber=%d) для задачи с hash=%s", subTask.SubTaskNumber, task.Hash)
				}
			}
			if changed {
				_, err = hashTaskCollection.UpdateOne(ctx, bson.M{"requestId": task.RequestId}, bson.M{"$set": bson.M{"subTasks": task.SubTasks}})
				if err != nil {
					log.Printf("Ошибка обновления задачи при переотправке: %v", err)
				}
			}
		}
		cursor.Close(ctx)
		cancel()
	}
}
