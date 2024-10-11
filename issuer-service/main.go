package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/streadway/amqp"
)

var db *pgxpool.Pool

func initDB() {
	var err error
	db, err = pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	} else {
		log.Println("Connected to database successfully")
	}
}

func main() {

	// debug

	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqPort := os.Getenv("RABBITMQ_PORT")
	rabbitmqUser := os.Getenv("RABBITMQ_USER") // Add your RabbitMQ username here
	rabbitmqPass := os.Getenv("RABBITMQ_PASS") // Add your RabbitMQ password here

	// Attempt to connect to RabbitMQ
	log.Println("In Main Function")
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitmqUser, rabbitmqPass, rabbitmqHost, rabbitmqPort))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// If successful, log a success message
	log.Println("Successfully connected to RabbitMQ with the provided credentials.")

	// end debug

	go startCredentialIssuanceWorker() // Start the worker in the background

	// Connect to PostgreSQL database
	initDB()

	// Initialize routes
	route := InitializeRoutes()

	// Load the base schema at startup
	baseSchema, err := loadBaseSchema("configs/base-schema.json")

	if err != nil {
		log.Fatalf("Error loading base schema: %v", err)
	}

	// Use the base schema as needed in your application
	log.Printf("Loaded Base Schema: %+v", baseSchema)

	// Start HTTP server
	log.Println("Credential service running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", route))
}
