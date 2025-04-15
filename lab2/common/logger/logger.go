package logger

import (
	"log"
)

// Log выводит общее сообщение журнала с тегом компонента.
func Log(component string, message string) {
	log.Printf("[%s] %s", component, message)
}

// LogHash выводит сообщение журнала, связанное с определенным хешем (задачей) с тегом компонента.
func LogHash(component string, hash string, message string) {
	log.Printf("[%s] [%s] %s", component, hash, message)
}

// LogTask выводит сообщение журнала, связанное с определенной подзадачей задачи взлома хеша.
func LogTask(component string, hash string, subTaskNum int, subTaskCount int, message string) {
	log.Printf("[%s] [%s:%d/%d] %s", component, hash, subTaskNum, subTaskCount, message)
}
