package main

import (
	"log"
	"net/http"
)

func main() {
	// Initialize routes
	router := initializeRoutes()

	// Start the server
	log.Println("Schema Service is running on port 8080...")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
