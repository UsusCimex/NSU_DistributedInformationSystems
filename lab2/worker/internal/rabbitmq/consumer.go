package rabbitmq

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"common/amqputil"
	"common/logger"
	"common/models"
	"worker/internal/processor"

	"github.com/streadway/amqp"
)

// ConsumeTasks регистрирует consumer для очереди "tasks" с реконнектом.
func ConsumeTasks(conn **amqp.Connection, rabbitURI string, proc *processor.Processor, sem chan struct{}, wg *sync.WaitGroup) {
	for {
		ch, err := amqputil.CreateChannel(*conn, "tasks", 3)
		if err != nil {
			logger.Log("Worker Consumer", fmt.Sprintf("Ошибка создания канала: %v. Пытаемся реконнект...", err))
			ch, err = amqputil.Reconnect(conn, rabbitURI, "tasks", 3, 5)
			if err != nil {
				logger.Log("Worker Consumer", fmt.Sprintf("Реконнект не удался: %v. Повтор через 5 секунд...", err))
				time.Sleep(5 * time.Second)
				continue
			}
		}

		proc.RmqChannel = ch

		msgs, err := registerConsumer(ch)
		if err != nil {
			logger.Log("Worker Consumer", fmt.Sprintf("Ошибка регистрации consumer: %v. Повтор через 5 секунд...", err))
			ch.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		logger.Log("Worker Consumer", "Consumer зарегистрирован")

		for d := range msgs {
			sem <- struct{}{}
			wg.Add(1)
			go func(d amqp.Delivery) {
				defer wg.Done()
				defer func() { <-sem }()
				var taskMsg models.TaskMessage
				if err := json.Unmarshal(d.Body, &taskMsg); err != nil {
					logger.Log("Worker Consumer", fmt.Sprintf("Ошибка декодирования сообщения: %v", err))
					d.Nack(false, false)
					return
				}
				proc.ProcessTask(d, taskMsg)
			}(d)
		}
		logger.Log("Worker Consumer", "Канал закрыт. Переподключение через 5 секунд...")
		time.Sleep(5 * time.Second)
	}
}

func registerConsumer(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	for {
		msgs, err := ch.Consume("tasks", "", false, false, false, false, nil)
		if err != nil {
			logger.Log("Worker Consumer", fmt.Sprintf("Ошибка регистрации consumer: %v. Повтор через 5 секунд...", err))
			time.Sleep(5 * time.Second)
			continue
		}
		return msgs, nil
	}
}
