package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// Initialize routes
	routes := InitializeRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Holder Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), routes))
}
