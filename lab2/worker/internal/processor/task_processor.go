package processor

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"strings"

	"worker/internal/models"

	"github.com/streadway/amqp"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// Processor выполняет обработку задач.
type Processor struct {
	WorkerID   string
	RmqChannel *amqp.Channel
}

func NewProcessor(workerID string, ch *amqp.Channel) *Processor {
	return &Processor{
		WorkerID:   workerID,
		RmqChannel: ch,
	}
}

func NumberToCandidate(n, length int) string {
	base := len(alphabet)
	var candidate strings.Builder
	for i := 0; i < length; i++ {
		candidate.WriteByte(alphabet[n%base])
		n /= base
	}
	// Разворачиваем строку.
	runes := []rune(candidate.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func (p *Processor) ProcessTask(d amqp.Delivery, msg models.TaskMessage) {
	log.Printf("%s: обработка задачи hash=%s, подзадача %d из %d", p.WorkerID, msg.Hash, msg.SubTaskNumber, msg.SubTaskCount)
	found := ""
	totalCandidates := int(math.Pow(float64(len(alphabet)), float64(msg.MaxLength)))
	progressInterval := 25000000
	iterationCount := 0

	for i := msg.SubTaskNumber - 1; i < totalCandidates; i += msg.SubTaskCount {
		iterationCount++
		if iterationCount%progressInterval == 0 {
			log.Printf("%s: обработано %d кандидатов для подзадачи %d", p.WorkerID, iterationCount, msg.SubTaskNumber)
		}
		candidate := NumberToCandidate(i, msg.MaxLength)
		hash := md5.Sum([]byte(candidate))
		if hex.EncodeToString(hash[:]) == msg.Hash {
			found = candidate
			break
		}
	}
	if found != "" {
		log.Printf("%s: кандидат найден для подзадачи %d: %s", p.WorkerID, msg.SubTaskNumber, found)
	} else {
		log.Printf("%s: кандидат не найден для подзадачи %d, итераций: %d", p.WorkerID, msg.SubTaskNumber, iterationCount)
	}

	resMsg := models.ResultMessage{
		Hash:          msg.Hash,
		SubTaskNumber: msg.SubTaskNumber,
		WorkerId:      p.WorkerID,
		Result:        found,
	}
	data, err := json.Marshal(resMsg)
	if err != nil {
		log.Printf("%s: ошибка маршалинга результата: %v", p.WorkerID, err)
		return
	}
	if err = p.RmqChannel.Publish("", "results", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	}); err != nil {
		log.Printf("%s: ошибка публикации результата: %v", p.WorkerID, err)
	} else {
		log.Printf("%s: отправлено сообщение результата по подзадаче %d", p.WorkerID, msg.SubTaskNumber)
	}
	d.Ack(false)
}
