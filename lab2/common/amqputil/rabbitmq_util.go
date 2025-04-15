package amqputil

import (
	"fmt"
	"time"

	"common/logger"
	"github.com/streadway/amqp"
)

const (
	DefaultMaxRetries = 5
)

// ConnectRabbitMQ пытается установить соединение с RabbitMQ с заданным числом повторов.
func ConnectRabbitMQ(uri string) (*amqp.Connection, error) {
	const maxRetries = 10
	var conn *amqp.Connection
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(uri)
		if err == nil {
			logger.Log("RabbitMQ", "Подключение к RabbitMQ установлено")
			return conn, nil
		}
		logger.Log("RabbitMQ", fmt.Sprintf("Ошибка подключения (попытка %d/%d): %v", i+1, maxRetries, err))
		time.Sleep(5 * time.Second)
	}
	return nil, err
}

// CreateChannel создает канал и объявляет очередь с указанным QoS.
// При ошибке повторяет попытки до достижения лимита.
func CreateChannel(conn *amqp.Connection, queueName string, qos int) (*amqp.Channel, error) {
	attempts := 0
	for {
		attempts++
		ch, err := conn.Channel()
		if err != nil {
			logger.Log("RabbitMQ", fmt.Sprintf("Ошибка открытия канала: %v. Повтор через 5 секунд... (Попытка %d/%d)", err, attempts, DefaultMaxRetries))
			if attempts >= DefaultMaxRetries {
				return nil, err
			}
			time.Sleep(5 * time.Second)
			continue
		}
		_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			logger.Log("RabbitMQ", fmt.Sprintf("Ошибка объявления очереди: %v. Повтор через 5 секунд... (Попытка %d/%d)", err, attempts, DefaultMaxRetries))
			ch.Close()
			if attempts >= DefaultMaxRetries {
				return nil, err
			}
			time.Sleep(5 * time.Second)
			continue
		}
		if qos > 0 {
			if err = ch.Qos(qos, 0, false); err != nil {
				logger.Log("RabbitMQ", fmt.Sprintf("Ошибка установки QoS: %v. Повтор через 5 секунд... (Попытка %d/%d)", err, attempts, DefaultMaxRetries))
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

// Reconnect пытается восстановить соединение и создать новый канал с очередью.
// Если соединение закрыто, оно восстанавливается.
func Reconnect(conn **amqp.Connection, dialURL, queueName string, qos, maxRetries int) (*amqp.Channel, error) {
	if (*conn).IsClosed() {
		var newConn *amqp.Connection
		var err error
		for i := 1; i <= maxRetries; i++ {
			newConn, err = amqp.Dial(dialURL)
			if err != nil {
				logger.Log("RabbitMQ", fmt.Sprintf("Ошибка подключения (попытка %d/%d): %v", i, maxRetries, err))
				time.Sleep(5 * time.Second)
				continue
			}
			*conn = newConn
			logger.Log("RabbitMQ", "Соединение восстановлено")
			break
		}
		if newConn == nil {
			return nil, err
		}
	}
	var ch *amqp.Channel
	var err error
	for i := 1; i <= maxRetries; i++ {
		ch, err = (*conn).Channel()
		if err != nil {
			logger.Log("RabbitMQ", fmt.Sprintf("Ошибка создания канала (попытка %d/%d): %v", i, maxRetries, err))
			time.Sleep(5 * time.Second)
			continue
		}
		_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			logger.Log("RabbitMQ", fmt.Sprintf("Ошибка объявления очереди (попытка %d/%d): %v", i, maxRetries, err))
			ch.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		logger.Log("RabbitMQ", "Канал восстановлен")
		return ch, nil
	}
	return nil, err
}
