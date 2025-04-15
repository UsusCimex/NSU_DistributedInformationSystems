package main

import (
	"log"
	"sync"
	"time"

	"worker/internal/connection"
	"worker/internal/processor"
	"worker/internal/rabbitmq"
)

func main() {
	log.Println("[Worker] Starting...")

	conn, rabbitURI, err := connection.ConnectRabbitMQ()
	if err != nil {
		log.Fatalf("[Worker] Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	proc := processor.NewProcessor(nil)

	for {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 3)
		log.Println("[Worker] Starting consumer...")
		rabbitmq.ConsumeTasks(&conn, rabbitURI, proc, sem, &wg)
		wg.Wait()
		log.Println("[Worker] Consumer finished, restarting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}
