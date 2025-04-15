package processor

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"strings"

	"common/models"

	"github.com/streadway/amqp"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// Processor выполняет обработку подзадачи.
type Processor struct {
	RmqChannel *amqp.Channel
}

// NewProcessor создаёт новый процессор.
func NewProcessor(ch *amqp.Channel) *Processor {
	return &Processor{RmqChannel: ch}
}

// NumberToCandidate преобразует число в строку-кандидат заданной длины.
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

// ProcessTask перебирает кандидатов для подзадачи и публикует результат в очередь "results".
// Если публикация завершается ошибкой, она логируется (реконнект здесь не реализован, так как основной механизм реконнекта осуществляется на уровне consumer).
func (p *Processor) ProcessTask(d amqp.Delivery, msg models.TaskMessage) {
	log.Printf("[Processor] Processing task for hash %s, subtask %d/%d", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount)
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
		log.Printf("[Processor] Found candidate for hash %s: %s", msg.Hash, found)
	} else {
		log.Printf("[Processor] No candidate found for hash %s", msg.Hash)
	}
	resMsg := models.ResultMessage{
		Hash:          msg.Hash,
		SubTaskNumber: msg.SubTaskNumber,
		Result:        found,
	}
	data, err := json.Marshal(resMsg)
	if err != nil {
		log.Printf("[Processor] Error marshalling result: %v", err)
		return
	}
	if err = p.RmqChannel.Publish(
		"",
		"results",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	); err != nil {
		log.Printf("[Processor] Error publishing result: %v", err)
		// При необходимости можно добавить реконнект канала,
		// но основной механизм реконнекта реализован на уровне consumer.
		return
	}
	log.Printf("[Processor] Result published for hash %s", msg.Hash)
	d.Ack(false)
}
