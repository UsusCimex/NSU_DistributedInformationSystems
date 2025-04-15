package amqputil

import (
	"fmt"
	"time"

	"common/logger"
	"github.com/streadway/amqp"
)

// Максимальное количество попыток повторения операции
const DefaultMaxRetries = 5

// ConnectRabbitMQ пытается установить соединение с RabbitMQ по указанному URI.
// Будет повторять попытки соединения до фиксированного количества раз, прежде чем вернуть ошибку.
func ConnectRabbitMQ(uri string) (*amqp.Connection, error) {
	const maxRetries = 10
	var conn *amqp.Connection
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(uri)
		if err == nil {
			logger.Log("RabbitMQ", "Соединение с RabbitMQ установлено")
			return conn, nil
		}
		logger.Log("RabbitMQ", fmt.Sprintf("Ошибка подключения (попытка %d/%d): %v", i+1, maxRetries, err))
		time.Sleep(5 * time.Second)
	}
	return nil, err
}

// CreateChannel открывает новый канал на указанном AMQP-соединении, объявляет очередь
// и применяет настройки QoS, если они указаны.
func CreateChannel(conn *amqp.Connection, queueName string, qos int) (*amqp.Channel, error) {
	var ch *amqp.Channel
	var err error
	for attempt := 1; attempt <= DefaultMaxRetries; attempt++ {
		// Пробуем открыть канал
		ch, err = conn.Channel()
		if err != nil {
			logger.Log("RabbitMQ", fmt.Sprintf("Ошибка открытия канала: %v. Повтор через 5 секунд... (Попытка %d/%d)", err, attempt, DefaultMaxRetries))
			if attempt >= DefaultMaxRetries {
				return nil, err
			}
			time.Sleep(5 * time.Second)
			continue
		}
		// Объявляем очередь
		_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			logger.Log("RabbitMQ", fmt.Sprintf("Ошибка объявления очереди '%s': %v. Повтор через 5 секунд... (Попытка %d/%d)", queueName, err, attempt, DefaultMaxRetries))
			_ = ch.Close()
			if attempt >= DefaultMaxRetries {
				return nil, err
			}
			time.Sleep(5 * time.Second)
			continue
		}
		// Устанавливаем QoS (prefetch count), если запрошен
		if qos > 0 {
			if err = ch.Qos(qos, 0, false); err != nil {
				logger.Log("RabbitMQ", fmt.Sprintf("Ошибка установки QoS: %v. Повтор через 5 секунд... (Попытка %d/%d)", err, attempt, DefaultMaxRetries))
				_ = ch.Close()
				if attempt >= DefaultMaxRetries {
					return nil, err
				}
				time.Sleep(5 * time.Second)
				continue
			}
		}
		return ch, nil
	}
	return nil, fmt.Errorf("не удалось создать канал RabbitMQ")
}

// Reconnect проверяет, открыто ли AMQP-соединение (переподключается при необходимости),
// а затем создает новый канал и объявляет очередь.
func Reconnect(connPtr **amqp.Connection, uri, queueName string, qos int, maxAttempts int) (*amqp.Channel, error) {
	// Если соединение закрыто или отсутствует, пытаемся переподключиться
	if *connPtr == nil || (*connPtr).IsClosed() {
		newConn, err := ConnectRabbitMQ(uri)
		if err != nil {
			return nil, err
		}
		*connPtr = newConn
	}
	// Создаем канал на активном соединении
	return CreateChannel(*connPtr, queueName, qos)
}
