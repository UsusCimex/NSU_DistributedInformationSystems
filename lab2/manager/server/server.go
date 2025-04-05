package server

import (
	"log"
	"manager/dispatcher"
	"manager/handlers"
	"manager/queue"
	"net/http"
)

func Start(taskQueue *queue.TaskQueue, taskDispatcher *dispatcher.TaskDispatcher) {
	// Внешние маршруты API (запуск задачи и получение статуса)
	http.HandleFunc("/api/hash/crack", handlers.CrackHashHandler(taskQueue))
	http.HandleFunc("/api/hash/status", handlers.StatusHandler)

	// Внутренние маршруты для взаимодействия с воркерами
	http.HandleFunc("/internal/api/manager/hash/crack/result", handlers.ResultHandler(taskDispatcher))
	http.HandleFunc("/internal/api/worker/register", handlers.WorkerRegisterHandler)

	log.Println("Manager listening on port :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server error:", err)
	}
}
