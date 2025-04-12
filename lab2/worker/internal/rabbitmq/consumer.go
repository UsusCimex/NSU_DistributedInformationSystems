package rabbitmq

import (
	"encoding/json"
	"log"
	"sync"

	"worker/internal/models"
	"worker/internal/processor"

	"github.com/streadway/amqp"
)

// ConsumeTasks получает сообщения из очереди "tasks" и передаёт их процессору.
// Ограничение одновременной обработки реализовано через канал-семафор.
func ConsumeTasks(channel *amqp.Channel, proc *processor.Processor, sem chan struct{}, wg *sync.WaitGroup) {
	msgs, err := channel.Consume(
		"tasks",
		"",
		false, // auto-ack отключён
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Ошибка регистрации consumer: %v", err)
	}

	for d := range msgs {
		sem <- struct{}{} // если занято 5 слотов – блокировка
		wg.Add(1)
		go func(d amqp.Delivery) {
			defer wg.Done()
			defer func() { <-sem }()

			var taskMsg models.TaskMessage
			if err := json.Unmarshal(d.Body, &taskMsg); err != nil {
				log.Printf("Ошибка декодирования сообщения задачи: %v", err)
				d.Nack(false, false)
				return
			}
			proc.ProcessTask(d, taskMsg)
		}(d)
	}
}
