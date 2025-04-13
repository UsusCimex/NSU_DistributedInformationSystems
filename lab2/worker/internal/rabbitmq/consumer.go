package rabbitmq

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"worker/internal/models"
	"worker/internal/processor"

	"github.com/streadway/amqp"
)

// ConsumeTasks регистрирует consumer для очереди "tasks" и передаёт сообщения Processor-у.
func ConsumeTasks(conn *amqp.Connection, proc *processor.Processor, sem chan struct{}, wg *sync.WaitGroup) {
	ch, err := createChannel(conn)
	if err != nil {
		log.Printf("[Consumer]: Ошибка создания канала: %v", err)
		return
	}
	proc.RmqChannel = ch
	msgs, err := registerConsumer(ch)
	if err != nil {
		log.Printf("[Consumer]: Ошибка регистрации consumer: %v", err)
		return
	}
	log.Printf("[Consumer]: Consumer зарегистрирован")
	for d := range msgs {
		sem <- struct{}{}
		wg.Add(1)
		go func(d amqp.Delivery) {
			defer wg.Done()
			defer func() { <-sem }()
			var taskMsg models.TaskMessage
			if err := json.Unmarshal(d.Body, &taskMsg); err != nil {
				log.Printf("[Consumer]: Ошибка декодирования задачи: %v", err)
				d.Nack(false, false)
				return
			}
			proc.ProcessTask(d, taskMsg)
		}(d)
	}
}

func createChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	for {
		ch, err := conn.Channel()
		if err != nil {
			log.Printf("[Consumer]: Ошибка открытия канала: %v. Повтор через 5 секунд", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if _, err = ch.QueueDeclare("tasks", true, false, false, false, nil); err != nil {
			ch.Close()
			log.Printf("[Consumer]: Ошибка объявления очереди: %v. Повтор через 5 секунд", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if err = ch.Qos(3, 0, false); err != nil {
			ch.Close()
			log.Printf("[Consumer]: Ошибка установки QoS: %v. Повтор через 5 секунд", err)
			time.Sleep(5 * time.Second)
			continue
		}
		return ch, nil
	}
}

func registerConsumer(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	for {
		msgs, err := ch.Consume("tasks", "", false, false, false, false, nil)
		if err != nil {
			log.Printf("[Consumer]: Ошибка регистрации consumer: %v. Повтор через 5 секунд", err)
			time.Sleep(5 * time.Second)
			continue
		}
		return msgs, nil
	}
}
