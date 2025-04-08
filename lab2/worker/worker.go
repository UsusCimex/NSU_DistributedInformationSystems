package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"strings"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// TaskMessage больше не содержит RequestId.
type TaskMessage struct {
	Hash          string `json:"hash"`
	MaxLength     int    `json:"maxLength"`
	SubTaskNumber int    `json:"subTaskNumber"`
	SubTaskCount  int    `json:"subTaskCount"`
}

// ResultMessage теперь содержит Hash вместо RequestId.
type ResultMessage struct {
	Hash          string `json:"hash"`
	SubTaskNumber int    `json:"subTaskNumber"`
	WorkerId      string `json:"workerId"`
	Result        string `json:"result"`
}

func numberToCandidate(n, length int) string {
	base := len(alphabet)
	var candidate strings.Builder
	for i := 0; i < length; i++ {
		candidate.WriteByte(alphabet[n%base])
		n = n / base
	}
	// Реверсируем строку
	runes := []rune(candidate.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// processTask теперь не принимает WaitGroup и не вызывает wg.Done()
// Управление завершением горутины происходит в main.
func processTask(d amqp.Delivery, msg TaskMessage) {
	// Используем context.Background без таймаута
	ctx := context.Background()

	// Фильтр ищет задачу по hash и подзадачу по subTaskNumber.
	filter := bson.M{
		"hash": msg.Hash,
		"subTasks": bson.M{
			"$elemMatch": bson.M{
				"subTaskNumber": msg.SubTaskNumber,
				"hash":          msg.Hash,
			},
		},
	}
	update := bson.M{"$set": bson.M{
		"subTasks.$.status":    "WORKING",
		"subTasks.$.workerId":  workerId,
		"subTasks.$.heartbeat": time.Now(),
		"subTasks.$.updatedAt": time.Now(),
	}}

	res, err := hashTaskCollection.UpdateOne(ctx, filter, update)
	if err != nil || res.MatchedCount == 0 {
		log.Printf("Worker %s: ошибка обновления подзадачи до WORKING: %v", workerId, err)
		d.Nack(false, true)
		return
	}
	d.Ack(false)

	log.Printf("Worker %s: получил задачу с hash=%s, подзадача %d из %d", workerId, msg.Hash, msg.SubTaskNumber, msg.SubTaskCount)

	foundResult := ""
	totalCandidates := int(math.Pow(float64(len(alphabet)), float64(msg.MaxLength)))
	progressInterval := 25_000_000
	iterationCount := 0

	for i := msg.SubTaskNumber - 1; i < totalCandidates; i += msg.SubTaskCount {
		iterationCount++
		if iterationCount%progressInterval == 0 {
			log.Printf("Worker %s: обработано %d кандидатов для подзадачи %d из %d (hash=%s)", workerId, iterationCount, msg.SubTaskNumber, msg.SubTaskCount, msg.Hash)
		}
		candidate := numberToCandidate(i, msg.MaxLength)
		hashBytes := md5.Sum([]byte(candidate))
		if hex.EncodeToString(hashBytes[:]) == msg.Hash {
			foundResult = candidate
			break
		}
	}

	if foundResult != "" {
		log.Printf("Worker %s: найден результат для подзадачи %d из %d: %s", workerId, msg.SubTaskNumber, msg.SubTaskCount, foundResult)
	} else {
		log.Printf("Worker %s: результат не найден для подзадачи %d из %d (обработано итераций: %d)", workerId, msg.SubTaskNumber, msg.SubTaskCount, iterationCount)
	}

	update = bson.M{"$set": bson.M{
		"subTasks.$.status":    "COMPLETE",
		"subTasks.$.updatedAt": time.Now(),
	}}
	_, err = hashTaskCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Worker %s: ошибка обновления подзадачи до COMPLETE: %v", workerId, err)
	}

	resMsg := ResultMessage{
		Hash:          msg.Hash,
		SubTaskNumber: msg.SubTaskNumber,
		WorkerId:      workerId,
		Result:        foundResult,
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
		log.Printf("Worker %s: отправлено сообщение результата по подзадаче %d из %d", workerId, msg.SubTaskNumber, msg.SubTaskCount)
	}
}

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
