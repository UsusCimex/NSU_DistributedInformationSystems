package rabbitmq

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"common/amqputil"
	"common/models"
	"worker/internal/processor"

	"github.com/streadway/amqp"
)

// ConsumeTasks регистрирует consumer для очереди "tasks" с реконнектом.
// В функцию передаётся указатель на указатель на соединение, что позволяет обновлять соединение при реконнекте.
func ConsumeTasks(conn **amqp.Connection, rabbitURI string, proc *processor.Processor, sem chan struct{}, wg *sync.WaitGroup) {
	for {
		// Попытка создать канал с нужной очередью.
		ch, err := amqputil.CreateChannel(*conn, "tasks", 3)
		if err != nil {
			log.Printf("[Worker Consumer] Error creating channel: %v. Attempting reconnect...", err)
			ch, err = amqputil.Reconnect(conn, rabbitURI, "tasks", 3, 5)
			if err != nil {
				log.Printf("[Worker Consumer] Reconnect failed: %v. Retrying in 5 seconds...", err)
				time.Sleep(5 * time.Second)
				continue
			}
		}

		// Обновляем канал процессора.
		proc.RmqChannel = ch

		// Регистрируем consumer.
		msgs, err := registerConsumer(ch)
		if err != nil {
			log.Printf("[Worker Consumer] Error registering consumer: %v. Retrying in 5 seconds...", err)
			ch.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("[Worker Consumer] Consumer registered")

		// Обрабатываем сообщения из очереди.
		for d := range msgs {
			sem <- struct{}{}
			wg.Add(1)
			go func(d amqp.Delivery) {
				defer wg.Done()
				defer func() { <-sem }()
				var taskMsg models.TaskMessage
				if err := json.Unmarshal(d.Body, &taskMsg); err != nil {
					log.Printf("[Worker Consumer] Error unmarshalling message: %v", err)
					d.Nack(false, false)
					return
				}
				proc.ProcessTask(d, taskMsg)
			}(d)
		}
		// Если цикл по сообщениям завершился – это признак закрытого канала, поэтому повторяем попытку реконнекта.
		log.Println("[Worker Consumer] Channel closed. Reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

// registerConsumer пытается зарегистрировать consumer для очереди "tasks" в цикле до успеха.
func registerConsumer(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	for {
		msgs, err := ch.Consume("tasks", "", false, false, false, false, nil)
		if err != nil {
			log.Printf("[Worker Consumer] Error registering consumer: %v. Retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		return msgs, nil
	}
}
