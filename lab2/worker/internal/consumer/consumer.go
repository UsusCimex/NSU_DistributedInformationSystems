package consumer

import (
	"encoding/json"
	"sync"
	"time"

	"common/amqputil"
	"common/logger"
	"common/models"
	"worker/internal/processor"

	"github.com/streadway/amqp"
)

// Consume создаёт канал, регистрирует consumer-а очереди "tasks" и обрабатывает входящие сообщения.
func Consume(conn **amqp.Connection, rabbitURI string) error {
	// Создаем канал через общий пакет, используя актуальное соединение
	ch, err := amqputil.CreateChannel(*conn, "tasks", 3)
	if err != nil {
		// Если ошибка вызвана закрытым соединением, пробуем реконнект
		if (*conn).IsClosed() {
			var newCh *amqp.Channel
			newCh, err = amqputil.Reconnect(conn, rabbitURI, "tasks", 3, 5)
			if err != nil {
				logger.Log("Worker Consumer", "Не удалось восстановить соединение: "+err.Error())
				return err
			}
			ch = newCh
		} else {
			logger.Log("Worker Consumer", "Ошибка создания канала: "+err.Error())
			return err
		}
	}
	defer ch.Close()

	// Регистрируем consumer для очереди "tasks"
	msgs, err := ch.Consume("tasks", "", false, false, false, false, nil)
	if err != nil {
		logger.Log("Worker Consumer", "Ошибка регистрации consumer: "+err.Error())
		return err
	}
	logger.Log("Worker Consumer", "Consumer зарегистрирован")

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)

	// Обрабатываем входящие сообщения
	for d := range msgs {
		sem <- struct{}{}
		wg.Add(1)
		go func(d amqp.Delivery) {
			defer wg.Done()
			defer func() { <-sem }()

			var taskMsg models.TaskMessage
			if err := json.Unmarshal(d.Body, &taskMsg); err != nil {
				logger.Log("Worker Consumer", "Ошибка декодирования сообщения: "+err.Error())
				d.Nack(false, false)
				return
			}
			// Обработка подзадачи через пакет processor
			processor.ProcessTask(ch, taskMsg)
			d.Ack(false)
		}(d)
	}
	wg.Wait()
	logger.Log("Worker Consumer", "Канал закрыт, переподключаемся через 5 секунд...")
	time.Sleep(5 * time.Second)
	return nil
}
