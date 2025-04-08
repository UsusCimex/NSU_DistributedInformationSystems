package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// ResultMessage теперь содержит Hash вместо RequestId.
type ResultMessage struct {
	Hash          string `json:"hash"`
	SubTaskNumber int    `json:"subTaskNumber"`
	WorkerId      string `json:"workerId"`
	Result        string `json:"result"`
}

func resultConsumer() {
	_, err := rabbitMQChannel.QueueDeclare(
		"results",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Ошибка объявления очереди results: %v", err)
	}

	msgs, err := rabbitMQChannel.Consume(
		"results",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Ошибка регистрации consumer для results: %v", err)
	}

	for d := range msgs {
		var res ResultMessage
		if err := json.Unmarshal(d.Body, &res); err != nil {
			log.Printf("Ошибка декодирования сообщения результата: %v", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// Находим задачу по hash.
		var task HashTask
		err := hashTaskCollection.FindOne(ctx, bson.M{"hash": res.Hash}).Decode(&task)
		if err != nil {
			log.Printf("Задача с hash %s не найдена при обработке результата", res.Hash)
			cancel()
			continue
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
			cancel()
			continue
		}

		task.CompletedTaskCount++
		if res.Result != "" {
			task.Status = "DONE"
			task.Result = res.Result
			log.Printf("Хэш расшифрован! Задача с hash %s, результат: %s", res.Hash, res.Result)
		} else if task.CompletedTaskCount >= task.SubTaskCount && task.Result == "" {
			task.Status = "FAIL"
			log.Printf("Хэш не расшифрован. Задача с hash %s помечена как FAIL", res.Hash)
		}

		_, err = hashTaskCollection.UpdateOne(ctx, bson.M{"hash": task.Hash}, bson.M{
			"$set": bson.M{
				"subTasks":           task.SubTasks,
				"completedTaskCount": task.CompletedTaskCount,
				"status":             task.Status,
				"result":             task.Result,
			},
		})
		if err != nil {
			log.Printf("Ошибка обновления задачи с результатом: %v", err)
		}
		cancel()
	}
}
