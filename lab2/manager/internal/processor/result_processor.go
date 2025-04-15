package processor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"common/logger"
	"common/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func ProcessResult(res models.ResultMessage, coll *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var task models.HashTask
	if err := coll.FindOne(ctx, bson.M{"hash": res.Hash}).Decode(&task); err != nil {
		logger.LogHash("Результаты", res.Hash, "Задача не найдена")
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
		logger.LogTask("Результаты", res.Hash, res.SubTaskNumber, task.SubTaskCount, "Подзадача не найдена")
		return errors.New("subtask not found")
	}
	task.CompletedTaskCount++

	if task.CompletedTaskCount == task.SubTaskCount {
		logger.LogHash("Результаты", task.Hash, "Все подзадачи выполнены")
	}

	if res.Result != "" {
		task.Status = "DONE"
		task.Result = res.Result
		logger.LogHash("Результаты", res.Hash, fmt.Sprintf("Хэш расшифрован, результат: %s", res.Result))
	} else if task.CompletedTaskCount >= task.SubTaskCount && task.Result == "" {
		task.Status = "FAIL"
		logger.LogHash("Результаты", res.Hash, "Хэш не расшифрован, задача отмечена как FAIL")
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
		logger.LogHash("Результаты", task.Hash, fmt.Sprintf("Ошибка обновления задачи: %v", err))
		return err
	}

	return nil
}
