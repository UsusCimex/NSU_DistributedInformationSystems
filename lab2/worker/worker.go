package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// Структура сообщения задачи, полученной из очереди "tasks".
type TaskMessage struct {
	RequestId     string `json:"requestId"`
	SubTaskId     string `json:"subTaskId"`
	Hash          string `json:"hash"`
	MaxLength     int    `json:"maxLength"`
	SubTaskNumber int    `json:"subTaskNumber"`
	SubTaskCount  int    `json:"subTaskCount"`
}

// Структура сообщения результата, которое отправляется в очередь "results".
type ResultMessage struct {
	RequestId string `json:"requestId"`
	SubTaskId string `json:"subTaskId"`
	WorkerId  string `json:"workerId"`
	Result    string `json:"result"`
}

// numberToCandidate преобразует число в строку длины length с использованием алфавита.
func numberToCandidate(n, length int) string {
	base := len(alphabet)
	var candidate strings.Builder
	for i := 0; i < length; i++ {
		candidate.WriteByte(alphabet[n%base])
		n /= base
	}
	runes := []rune(candidate.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// processTask выполняет обработку полученной подзадачи:
// – обновляет статус subTask в MongoDB до "WORKING" (и подтверждает сообщение RabbitMQ);
// – перебирает кандидатов согласно схеме: начиная с (subTaskNumber-1) и шагом равным количеству подзадач;
// – по завершении обновляет subTask до "COMPLETE" и отправляет результат в очередь "results".
func processTask(d amqp.Delivery, msg TaskMessage, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	filter := bson.M{"requestId": msg.RequestId, "subTasks.id": msg.SubTaskId}
	update := bson.M{"$set": bson.M{
		"subTasks.$.status":    "WORKING",
		"subTasks.$.workerId":  workerId,
		"subTasks.$.heartbeat": time.Now(),
		"subTasks.$.updatedAt": time.Now(),
	}}

	res, err := hashTaskCollection.UpdateOne(ctx, filter, update)
	if err != nil || res.MatchedCount == 0 {
		log.Printf("Worker %s: ошибка обновления subTask до WORKING: %v", workerId, err)
		d.Nack(false, true)
		return
	}
	d.Ack(false)

	foundResult := ""
	totalCandidates := int(math.Pow(float64(len(alphabet)), float64(msg.MaxLength)))
	for i := msg.SubTaskNumber - 1; i < totalCandidates; i += msg.SubTaskCount {
		candidate := numberToCandidate(i, msg.MaxLength)
		hash := md5.Sum([]byte(candidate))
		if hex.EncodeToString(hash[:]) == msg.Hash {
			foundResult = candidate
			break
		}
	}

	update = bson.M{"$set": bson.M{
		"subTasks.$.status":    "COMPLETE",
		"subTasks.$.updatedAt": time.Now(),
	}}
	_, err = hashTaskCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Worker %s: ошибка обновления subTask до COMPLETE: %v", workerId, err)
	}

	resMsg := ResultMessage{
		RequestId: msg.RequestId,
		SubTaskId: msg.SubTaskId,
		WorkerId:  workerId,
		Result:    foundResult,
	}
	data, err := json.Marshal(resMsg)
	if err != nil {
		log.Printf("Worker %s: ошибка маршалинга сообщения результата: %v", workerId, err)
		return
	}

	err = rabbitMQChannel.Publish(
		"",
		"results",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
	if err != nil {
		log.Printf("Worker %s: ошибка публикации сообщения результата: %v", workerId, err)
	} else {
		log.Printf("Worker %s: обработана подзадача %s", workerId, msg.SubTaskId)
	}
}

// heartbeatUpdater каждую 10‑секунд обновляет поле Heartbeat для всех подзадач этого воркера.
func heartbeatUpdater() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		filter := bson.M{
			"subTasks": bson.M{
				"$elemMatch": bson.M{
					"workerId": workerId,
					"status":   "WORKING",
				},
			},
		}
		update := bson.M{"$set": bson.M{
			"subTasks.$[elem].heartbeat": time.Now(),
		}}
		opts := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{bson.M{"elem.workerId": workerId, "elem.status": "WORKING"}},
		})
		_, err := hashTaskCollection.UpdateMany(ctx, filter, update, opts)
		if err != nil {
			log.Printf("Worker %s: ошибка обновления heartbeat: %v", workerId, err)
		}
		cancel()
	}
}
