package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	// Load environment variables
	_ = godotenv.Load()

	// Initialize routes
	routes := InitializeRoutes()

	// Initialize Port from the env
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Holder Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), routes))
}
