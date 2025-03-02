package main

import (
	"log"
	"manager/handlers"
	"manager/store"
	"net/http"
)

func main() {
	store.Init()

	http.HandleFunc("/api/hash/crack", handlers.CrackHashHandler)
	http.HandleFunc("/api/hash/status", handlers.StatusHandler)
	http.HandleFunc("/internal/api/manager/hash/crack/request", handlers.ResultTaskHandler)

	log.Println("Starting manager server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
