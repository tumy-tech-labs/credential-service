package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
)

// Enqueue bulk issuance requests to RabbitMQ
func enqueueBulkIssuance(req CredentialRequest) error {

	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqPort := os.Getenv("RABBITMQ_PORT")
	rabbitmqUser := os.Getenv("RABBITMQ_USER") // Add your RabbitMQ username here
	rabbitmqPass := os.Getenv("RABBITMQ_PASS") // Add your RabbitMQ password here

	// Attempt to connect to RabbitMQ

	log.Println("In enqueueBulkIssuance Function: ")

	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitmqUser, rabbitmqPass, rabbitmqHost, rabbitmqPort))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// If successful, log a success message
	log.Println("Successfully connected to RabbitMQ with the provided credentials.")

	// You can also verify other actions, like checking for existing queues, etc.

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"credential_issuance_queue", // Name of the queue
		true,                        // Durable
		false,                       // Delete when unused
		false,                       // Exclusive
		false,                       // No-wait
		nil,                         // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %v", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	err = ch.Publish(
		"",     // Exchange
		q.Name, // Routing key
		false,  // Mandatory
		false,  // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %v", err)
	}

	log.Printf("Enqueued bulk issuance request: %+v", req)
	return nil
}
