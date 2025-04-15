package logger

import "log"

// Log выводит сообщение компонента без контекста задачи.
func Log(component, message string) {
	log.Printf("[%s] : %s", component, message)
}

// LogHash выводит сообщение с указанием хэша задачи.
func LogHash(component, hash, message string) {
	log.Printf("[%s] \"%s\": %s", component, hash, message)
}

// LogTask выводит сообщение с указанием хэша, номера подзадачи и общего количества подзадач.
func LogTask(component, hash string, taskNum, total int, message string) {
	log.Printf("[%s] \"%s(%d/%d)\": %s", component, hash, taskNum, total, message)
}
