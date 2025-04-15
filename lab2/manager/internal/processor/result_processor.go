package processor

import (
	"common/models"
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func ProcessResult(res models.ResultMessage, coll *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var task models.HashTask
	if err := coll.FindOne(ctx, bson.M{"hash": res.Hash}).Decode(&task); err != nil {
		log.Printf("[ResultConsumer] %s: Задача не найдена", res.Hash)
		return errors.New("task not found")
	}
	updated := false
	for i, subTask := range task.SubTasks {
		if subTask.SubTaskNumber == res.SubTaskNumber {
			task.SubTasks[i].Status = "COMPLETE"
			task.SubTasks[i].UpdatedAt = time.Now()
			updated = true
			break
		}
	}
	if !updated {
		log.Printf("[ResultConsumer] %s: Подзадача %d не найдена в задаче", res.Hash, res.SubTaskNumber)
		return errors.New("subtask not found")
	}
	task.CompletedTaskCount++

	if task.CompletedTaskCount == task.SubTaskCount {
		log.Printf("[ResultConsumer] %s: Все подзадачи выполнены!", task.Hash)
	}

	if res.Result != "" {
		task.Status = "DONE"
		task.Result = res.Result
		log.Printf("[ResultConsumer] %s: Хэш расшифрован, результат: %s", res.Hash, res.Result)
	} else if task.CompletedTaskCount >= task.SubTaskCount && task.Result == "" {
		task.Status = "FAIL"
		log.Printf("[ResultConsumer] %s: Хэш не расшифрован, задача отмечена как FAIL", res.Hash)
	}

	_, err := coll.UpdateOne(ctx, bson.M{"hash": task.Hash}, bson.M{
		"$set": bson.M{
			"subTasks":           task.SubTasks,
			"completedTaskCount": task.CompletedTaskCount,
			"status":             task.Status,
			"result":             task.Result,
		},
	})

	if err != nil {
		log.Printf("[ResultConsumer] %s: Ошибка обновления задачи с результатом: %v", task.Hash, err)
		return err
	}

	return nil
}
