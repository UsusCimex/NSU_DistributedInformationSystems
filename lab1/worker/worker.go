package main

import (
	"log"
	"net/http"
	"worker/handlers"
)

func main() {
	http.HandleFunc("/internal/api/worker/hash/crack/task", handlers.CrackTaskHandler)

	log.Printf("Starting worker server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
