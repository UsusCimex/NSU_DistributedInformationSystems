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
	RmqChannel *amqp.Channel
}

func NewProcessor(ch *amqp.Channel) *Processor {
	return &Processor{
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
	log.Printf("[Processor] %s(%d): обработка задачи, подзадача %d из %d",
		msg.Hash, msg.SubTaskNumber, msg.SubTaskNumber, msg.SubTaskCount)

	found := ""
	totalCandidates := int(math.Pow(float64(len(alphabet)), float64(msg.MaxLength)))

	for i := msg.SubTaskNumber - 1; i < totalCandidates; i += msg.SubTaskCount {
		candidate := NumberToCandidate(i, msg.MaxLength)
		hash := md5.Sum([]byte(candidate))
		if hex.EncodeToString(hash[:]) == msg.Hash {
			found = candidate
			break
		}
	}

	if found != "" {
		log.Printf("[Processor] %s(%d): кандидат найден: %s",
			msg.Hash, msg.SubTaskNumber, found)
	} else {
		log.Printf("[Processor] %s(%d): кандидат не найден",
			msg.Hash, msg.SubTaskNumber)
	}

	resMsg := models.ResultMessage{
		Hash:          msg.Hash,
		SubTaskNumber: msg.SubTaskNumber,
		Result:        found,
	}

	data, err := json.Marshal(resMsg)
	if err != nil {
		log.Printf("[Processor] %s(%d): ошибка маршалинга результата: %v",
			msg.Hash, msg.SubTaskNumber, err)
		return
	}

	if err = p.RmqChannel.Publish("", "results", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	}); err != nil {
		log.Printf("[Processor] %s(%d): ошибка публикации результата: %v",
			msg.Hash, msg.SubTaskNumber, err)
	} else {
		log.Printf("[Processor] %s(%d): отправлено сообщение результата",
			msg.Hash, msg.SubTaskNumber)
	}

	d.Ack(false)
}
