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

// ProcessResult обновляет HashTask в базе данных на основе полученного результата подзадачи.
// Отмечает подзадачу как завершенную и, если пароль найден или все подзадачи завершены,
// обновляет общий статус задачи и результат.
func ProcessResult(res models.ResultMessage, task models.HashTask, coll *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Отмечаем конкретную подзадачу как COMPLETE
	subTaskFound := false
	for i := range task.SubTasks {
		if task.SubTasks[i].SubTaskNumber == res.SubTaskNumber {
			task.SubTasks[i].Status = "COMPLETE"
			task.SubTasks[i].UpdatedAt = time.Now()
			subTaskFound = true
			break
		}
	}
	if !subTaskFound {
		logger.LogTask("Processor", res.Hash, res.SubTaskNumber, task.SubTaskCount, "Подзадача не найдена в структуре задачи")
		return errors.New("subtask not found")
	}

	// Увеличиваем счетчик завершенных подзадач
	task.CompletedTaskCount++
	if task.CompletedTaskCount == task.SubTaskCount {
		logger.LogHash("Processor", task.Hash, "Все подзадачи завершены")
	}

	// Обновляем общий статус задачи на основе найденного результата
	if res.Result != "" {
		// Пароль был найден в этой подзадаче
		task.Status = "DONE"
		task.Result = res.Result
		logger.LogHash("Processor", res.Hash, fmt.Sprintf("Хэш успешно расшифрован: %s", res.Result))
	} else if task.CompletedTaskCount >= task.SubTaskCount && task.Result == "" {
		// Все подзадачи завершены и результат не найден
		task.Status = "FAIL"
		logger.LogHash("Processor", res.Hash, "Хэш не расшифрован (задача отмечена как FAIL)")
	}

	// Обновляем документ задачи в базе данных
	update := bson.M{
		"$set": bson.M{
			"subTasks":           task.SubTasks,
			"completedTaskCount": task.CompletedTaskCount,
			"status":             task.Status,
			"result":             task.Result,
		},
	}
	_, err := coll.UpdateOne(ctx, bson.M{"hash": task.Hash}, update)
	if err != nil {
		logger.LogHash("Processor", task.Hash, fmt.Sprintf("Ошибка сохранения результата в БД: %v", err))
		return err
	}
	return nil
}
