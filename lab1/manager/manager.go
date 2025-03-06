package main

import (
	"log"
	"manager/balancer"
	"manager/dispatcher"
	"manager/handlers"
	"manager/queue"
	"manager/store"
	"net/http"
)

func main() {
	store.Init()
	balancer.Init()

	taskQueue := queue.NewTaskQueue()
	taskDispatcher := dispatcher.NewTaskDispatcher(taskQueue)
	taskDispatcher.Start()

	http.HandleFunc("/api/hash/crack", handlers.CrackHashHandler(taskQueue))
	http.HandleFunc("/api/hash/status", handlers.StatusHandler)
	http.HandleFunc("/internal/api/manager/hash/crack/result", handlers.ResultHandler(taskDispatcher))
	http.HandleFunc("/internal/api/worker/register", handlers.WorkerRegisterHandler)

	log.Println("Starting manager server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
