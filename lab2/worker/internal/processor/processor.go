package processor

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"common/constants"
	"common/logger"
	"common/models"

	"github.com/streadway/amqp"
)

// NumberToCandidate преобразует число в строку в системе счисления с основанием constants.AlphabetSize.
func NumberToCandidate(n int, length int) string {
	base := constants.AlphabetSize
	var candidateBuilder strings.Builder
	for i := 0; i < length; i++ {
		candidateBuilder.WriteByte(constants.Alphabet[n%base])
		n /= base
	}
	// Переворачиваем строку
	runes := []rune(candidateBuilder.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ProcessTask перебирает пространство поиска для данной подзадачи, проверяет каждого кандидата на соответствие хешу и публикует результат.
func ProcessTask(ch *amqp.Channel, msg models.TaskMessage) {
	logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, "Начало обработки задачи")
	found := ""
	totalCandidates := int(math.Pow(float64(constants.AlphabetSize), float64(msg.MaxLength)))

	// Перебираем часть пространства поиска, назначенную этой подзадаче
	for i := msg.SubTaskNumber - 1; i < totalCandidates; i += msg.SubTaskCount {
		candidate := NumberToCandidate(i, msg.MaxLength)
		sum := md5.Sum([]byte(candidate))
		if hex.EncodeToString(sum[:]) == msg.Hash {
			found = candidate
			break
		}
	}

	if found != "" {
		logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, "Найден пароль: "+found)
	} else {
		logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, "Пароль не найден в данной подзадаче")
	}

	resMsg := models.ResultMessage{
		Hash:          msg.Hash,
		SubTaskNumber: msg.SubTaskNumber,
		Result:        found,
	}
	data, err := json.Marshal(resMsg)
	if err != nil {
		logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount,
			fmt.Sprintf("Ошибка маршалинга результата: %v", err))
		return
	}

	err = ch.Publish(
		"",
		constants.ResultsQueue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
	if err != nil {
		logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount,
			fmt.Sprintf("Ошибка публикации результата: %v", err))
		return
	}
	logger.LogTask("Processor", msg.Hash, msg.SubTaskNumber, msg.SubTaskCount, "Результат отправлен в очередь 'results'")
}
