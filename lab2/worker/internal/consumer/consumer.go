package consumer

import (
	"encoding/json"
	"sync"

	"common/amqputil"
	"common/constants"
	"common/logger"
	"common/models"
	"worker/internal/processor"

	"github.com/streadway/amqp"
)

// Consume подключается к очереди "tasks", потребляет сообщения и обрабатывает их.
// Возвращает ошибку при закрытии канала или соединения для обработки переподключения.
func Consume(connPtr **amqp.Connection, rabbitURI string) error {
	// Создаем канал для очереди "tasks" с предварительной выборкой 3 сообщений за раз
	ch, err := amqputil.CreateChannel(*connPtr, constants.TasksQueue, constants.DefaultPrefetchCount)
	if err != nil {
		if *connPtr != nil && (*connPtr).IsClosed() {
			newCh, recErr := amqputil.Reconnect(connPtr, rabbitURI, constants.TasksQueue, constants.DefaultPrefetchCount, 5)
			if recErr != nil {
				logger.Log("Worker Consumer", "Не удалось восстановить канал: "+recErr.Error())
				return recErr
			}
			ch = newCh
		} else {
			logger.Log("Worker Consumer", "Ошибка создания канала: "+err.Error())
			return err
		}
	}
	defer ch.Close()

	// Подписываемся на очередь "tasks"
	msgs, err := ch.Consume(constants.TasksQueue, "", false, false, false, false, nil)
	if err != nil {
		logger.Log("Worker Consumer", "Ошибка регистрации consumer: "+err.Error())
		return err
	}
	logger.Log("Worker Consumer", "Consumer для очереди 'tasks' зарегистрирован")

	// Используем WaitGroup для синхронизации обработки сообщений
	var wg sync.WaitGroup
	// Ограничиваем количество одновременно запущенных горутин
	maxConcurrent := constants.DefaultMaxConcurrency
	sem := make(chan struct{}, maxConcurrent)

	// Обработка входящих сообщений
	for d := range msgs {
		sem <- struct{}{}
		wg.Add(1)
		go func(delivery amqp.Delivery) {
			defer wg.Done()
			defer func() { <-sem }()

			var taskMsg models.TaskMessage
			if err := json.Unmarshal(delivery.Body, &taskMsg); err != nil {
				logger.Log("Worker Consumer", "Ошибка декодирования сообщения: "+err.Error())
				delivery.Ack(false)
				return
			}
			processor.ProcessTask(ch, taskMsg)
			delivery.Ack(false)
		}(d)
	}

	wg.Wait()
	logger.Log("Worker Consumer", "Канал 'tasks' закрыт")
	return nil
}
