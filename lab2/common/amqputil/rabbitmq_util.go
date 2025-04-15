package amqputil

import (
	"log"
	"time"

	"github.com/streadway/amqp"
)

const (
	DefaultMaxRetries = 5
)

// CreateChannel создаёт новый канал, объявляя очередь queueName с опциональным QoS.
// Если число попыток превышает DefaultMaxRetries, возвращается ошибка.
func CreateChannel(conn *amqp.Connection, queueName string, qos int) (*amqp.Channel, error) {
	attempts := 0
	for {
		attempts++
		ch, err := conn.Channel()
		if err != nil {
			log.Printf("[AMQPUtil] Error opening channel: %v. Retrying in 5 seconds... (attempt %d/%d)", err, attempts, DefaultMaxRetries)
			if attempts >= DefaultMaxRetries {
				return nil, err
			}
			time.Sleep(5 * time.Second)
			continue
		}
		_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			log.Printf("[AMQPUtil] Error declaring queue: %v. Retrying in 5 seconds... (attempt %d/%d)", err, attempts, DefaultMaxRetries)
			ch.Close()
			if attempts >= DefaultMaxRetries {
				return nil, err
			}
			time.Sleep(5 * time.Second)
			continue
		}
		if qos > 0 {
			if err = ch.Qos(qos, 0, false); err != nil {
				log.Printf("[AMQPUtil] Error setting QoS: %v. Retrying in 5 seconds... (attempt %d/%d)", err, attempts, DefaultMaxRetries)
				ch.Close()
				if attempts >= DefaultMaxRetries {
					return nil, err
				}
				time.Sleep(5 * time.Second)
				continue
			}
		}
		return ch, nil
	}
}

// Reconnect пытается восстановить соединение (если оно закрыто) и создать новый канал с объявлением очереди.
// dialURL: URI для подключения, queueName: имя очереди, qos: нужное значение QoS, maxRetries: максимум попыток.
func Reconnect(conn **amqp.Connection, dialURL, queueName string, qos, maxRetries int) (*amqp.Channel, error) {
	// Если соединение закрыто – восстанавливаем его.
	if (*conn).IsClosed() {
		var newConn *amqp.Connection
		var err error
		for i := 1; i <= maxRetries; i++ {
			newConn, err = amqp.Dial(dialURL)
			if err != nil {
				log.Printf("[AMQPUtil] Reconnect attempt %d/%d failed: %v", i, maxRetries, err)
				time.Sleep(5 * time.Second)
				continue
			}
			*conn = newConn
			log.Println("[AMQPUtil] RabbitMQ connection restored")
			break
		}
		if newConn == nil {
			return nil, err
		}
	}
	// Создаём новый канал.
	var ch *amqp.Channel
	var err error
	for i := 1; i <= maxRetries; i++ {
		ch, err = (*conn).Channel()
		if err != nil {
			log.Printf("[AMQPUtil] Channel reconnect attempt %d/%d failed: %v", i, maxRetries, err)
			time.Sleep(5 * time.Second)
			continue
		}
		_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			log.Printf("[AMQPUtil] Error declaring queue during channel reconnect attempt %d/%d: %v", i, maxRetries, err)
			ch.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("[AMQPUtil] RabbitMQ channel restored")
		return ch, nil
	}
	return nil, err
}
