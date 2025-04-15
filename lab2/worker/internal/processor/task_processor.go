package processor

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"common/logger"
	"common/models"

	"github.com/streadway/amqp"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// Processor выполняет обработку подзадачи.
type Processor struct {
	RmqChannel *amqp.Channel
}

// NewProcessor создает новый Processor.
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
	runes := []rune(candidate.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ProcessTask перебирает кандидатов для подзадачи и публикует результат.
func (p *Processor) ProcessTask(msg models.TaskMessage) {
	logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, "Начало обработки задачи")
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
		logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, "Найден кандидат: "+found)
	} else {
		logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, "Кандидат не найден")
	}
	resMsg := models.ResultMessage{
		Hash:          msg.Hash,
		SubTaskNumber: msg.SubTaskNumber,
		Result:        found,
	}
	data, err := json.Marshal(resMsg)
	if err != nil {
		logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, fmt.Sprintf("Ошибка маршалинга результата: %v", err))
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
		logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, fmt.Sprintf("Ошибка публикации результата: %v", err))
		return
	}
	logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, "Результат опубликован")
}
