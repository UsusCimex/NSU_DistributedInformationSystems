package receiver

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

// StartResultConsumer регистрирует consumer для очереди "results" и обрабатывает полученные сообщения.
func StartResultConsumer(rmqChannel *amqp.Channel, coll *mongo.Collection) {
	_, err := rmqChannel.QueueDeclare("results", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка объявления очереди results: %v", err)
	}
	msgs, err := rmqChannel.Consume("results", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка регистрации consumer для results: %v", err)
	}
	go func() {
		for d := range msgs {
			var res models.ResultMessage
			if err := json.Unmarshal(d.Body, &res); err != nil {
				log.Printf("Ошибка декодирования сообщения результата: %v", err)
				continue
			}
			processResult(res, coll)
		}
	}()
}

func processResult(res models.ResultMessage, coll *mongo.Collection) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var task models.HashTask
	if err := coll.FindOne(ctx, bson.M{"hash": res.Hash}).Decode(&task); err != nil {
		log.Printf("Задача с hash %s не найдена", res.Hash)
		return
	}
	updated := false
	for i, subTask := range task.SubTasks {
		if subTask.SubTaskNumber == res.SubTaskNumber {
			task.SubTasks[i].Status = "COMPLETE"
			task.SubTasks[i].WorkerId = res.WorkerId
			task.SubTasks[i].UpdatedAt = time.Now()
			updated = true
			break
		}
	}
	if !updated {
		log.Printf("Подзадача %d не найдена в задаче с hash %s", res.SubTaskNumber, res.Hash)
		return
	}
	task.CompletedTaskCount++

	if task.CompletedTaskCount == task.SubTaskCount {
		log.Printf("Все подзадачи для hash %s выполнены!", task.Hash)
	}

	if res.Result != "" {
		task.Status = "DONE"
		task.Result = res.Result
		log.Printf("Хэш расшифрован: %s, результат: %s", res.Hash, res.Result)
	} else if task.CompletedTaskCount >= task.SubTaskCount && task.Result == "" {
		task.Status = "FAIL"
		log.Printf("Хэш не расшифрован, задача %s отмечена как FAIL", res.Hash)
	}
	if _, err := coll.UpdateOne(ctx, bson.M{"hash": task.Hash}, bson.M{
		"$set": bson.M{
			"subTasks":           task.SubTasks,
			"completedTaskCount": task.CompletedTaskCount,
			"status":             task.Status,
			"result":             task.Result,
		},
	}); err != nil {
		log.Printf("Ошибка обновления задачи с результатом: %v", err)
	}
}
