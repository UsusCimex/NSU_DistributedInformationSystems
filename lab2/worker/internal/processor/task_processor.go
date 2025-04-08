package processor

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
	"go.mongodb.org/mongo-driver/mongo"

	"worker/internal/models"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// Processor инкапсулирует зависимости для обработки подзадач.
type Processor struct {
	WorkerID        string
	Collection      *mongo.Collection
	RabbitMQChannel *amqp.Channel
}

// NewProcessor создаёт новый экземпляр Processor.
func NewProcessor(workerID string, coll *mongo.Collection, channel *amqp.Channel) *Processor {
	return &Processor{
		WorkerID:        workerID,
		Collection:      coll,
		RabbitMQChannel: channel,
	}
}

// NumberToCandidate преобразует число в строкового кандидата заданной длины.
func NumberToCandidate(n, length int) string {
	base := len(alphabet)
	var candidate strings.Builder
	for i := 0; i < length; i++ {
		candidate.WriteByte(alphabet[n%base])
		n = n / base
	}
	runes := []rune(candidate.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ProcessTask выполняет обработку подзадачи:
// 1. Обновляет статус на "WORKING" в MongoDB (и подтверждает сообщение из RabbitMQ).
// 2. Перебирает кандидатов для вычисления MD5.
// 3. Периодически логирует прогресс.
// 4. Обновляет статус на "COMPLETE" и отправляет результат через RabbitMQ.
func (p *Processor) ProcessTask(d amqp.Delivery, msg models.TaskMessage) {
	ctx := context.Background()

	// Фильтр для поиска подзадачи по hash и subTaskNumber
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
		"subTasks.$.workerId":  p.WorkerID,
		"subTasks.$.heartbeat": time.Now(),
		"subTasks.$.updatedAt": time.Now(),
	}}

	res, err := p.Collection.UpdateOne(ctx, filter, update)
	if err != nil || res.MatchedCount == 0 {
		log.Printf("Worker %s: ошибка обновления подзадачи до WORKING: %v", p.WorkerID[:5], err)
		d.Nack(false, true)
		return
	}
	d.Ack(false)

	log.Printf("Worker %s: получил задачу с hash=%s, подзадача %d из %d", p.WorkerID[:5], msg.Hash, msg.SubTaskNumber, msg.SubTaskCount)

	foundResult := ""
	totalCandidates := int(math.Pow(float64(len(alphabet)), float64(msg.MaxLength)))
	progressInterval := 25000000
	iterationCount := 0

	for i := msg.SubTaskNumber - 1; i < totalCandidates; i += msg.SubTaskCount {
		iterationCount++
		if iterationCount%progressInterval == 0 {
			log.Printf("Worker %s: обработано %d кандидатов для подзадачи %d из %d (hash=%s)", p.WorkerID[:5], iterationCount, msg.SubTaskNumber, msg.SubTaskCount, msg.Hash)
		}
		candidate := NumberToCandidate(i, msg.MaxLength)
		hashBytes := md5.Sum([]byte(candidate))
		if hex.EncodeToString(hashBytes[:]) == msg.Hash {
			foundResult = candidate
			break
		}
	}

	if foundResult != "" {
		log.Printf("Worker %s: найден результат для подзадачи %d из %d: %s", p.WorkerID[:5], msg.SubTaskNumber, msg.SubTaskCount, foundResult)
	} else {
		log.Printf("Worker %s: результат не найден для подзадачи %d из %d (обработано итераций: %d)", p.WorkerID[:5], msg.SubTaskNumber, msg.SubTaskCount, iterationCount)
	}

	update = bson.M{"$set": bson.M{
		"subTasks.$.status":    "COMPLETE",
		"subTasks.$.updatedAt": time.Now(),
	}}
	_, err = p.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Worker %s: ошибка обновления подзадачи до COMPLETE: %v", p.WorkerID[:5], err)
	}

	resMsg := models.ResultMessage{
		Hash:          msg.Hash,
		SubTaskNumber: msg.SubTaskNumber,
		WorkerId:      p.WorkerID,
		Result:        foundResult,
	}
	data, err := json.Marshal(resMsg)
	if err != nil {
		log.Printf("Worker %s: ошибка маршалинга сообщения результата: %v", p.WorkerID[:5], err)
		return
	}

	err = p.RabbitMQChannel.Publish(
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
		log.Printf("Worker %s: ошибка публикации сообщения результата: %v", p.WorkerID[:5], err)
	} else {
		log.Printf("Worker %s: отправлено сообщение результата по подзадаче %d из %d", p.WorkerID[:5], msg.SubTaskNumber, msg.SubTaskCount)
	}
}
