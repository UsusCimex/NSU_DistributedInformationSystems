package constants

import "time"

const (
	// Очереди
	TasksQueue   = "tasks"
	ResultsQueue = "results"

	// Worker Consumer
	DefaultPrefetchCount  = 3
	DefaultMaxConcurrency = 3

	// AMQP utils
	DefaultConnectRetries = 10
	DefaultChannelRetries = 5

	// URI и БД
	DefaultRabbitURI = "amqp://guest:guest@rabbitmq:5672/"
	DefaultMongoURI  = "mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0"
	DefaultDBName    = "hash_cracker"

	// Publisher
	PublishLimit = 100

	// Таймауты
	ContextTimeout     = 5 * time.Second
	LongContextTimeout = 10 * time.Second

	// Алфавит для перебора
	Alphabet     = "abcdefghijklmnopqrstuvwxyz0123456789"
	AlphabetSize = len(Alphabet)

	// Параметры расчёта подзадач
	MaxCandidatesPerSubTask = 15_000_000.0
	MinMaxLength            = 1
	MaxMaxLength            = 7
)
