package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
)

// StartCredentialIssuanceWorker starts a worker to process credential issuance from RabbitMQ
func startCredentialIssuanceWorker() {
	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqPort := os.Getenv("RABBITMQ_PORT")
	rabbitmqUser := os.Getenv("RABBITMQ_USER") // Add your RabbitMQ username here
	rabbitmqPass := os.Getenv("RABBITMQ_PASS") // Add your RabbitMQ password here

	// Attempt to connect to RabbitMQ

	log.Println("In startCredentialIssuanceWorker Function: ")

	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitmqUser, rabbitmqPass, rabbitmqHost, rabbitmqPort))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"credential_issuance_queue",
		true,  // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	log.Printf("Worker started, waiting for bulk issuance jobs...")

	// Process bulk issuance requests
	for msg := range msgs {
		var req CredentialRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			log.Printf("Failed to decode bulk issuance request: %v", err)
			continue
		}

		log.Printf("Processing bulk issuance request: %+v", req)
		for _, subject := range req.Subjects {
			if err := issueCredentialForSubject(req.IssuerDid, subject); err != nil {
				log.Printf("Failed to issue credential for subject: %+v, error: %v", subject, err)
			}
		}
	}
}

// issueCredentialForSubject processes an individual credential issuance request
func issueCredentialForSubject(issuerDid string, subject map[string]interface{}) error {
	// Replace with your credential issuance logic for each subject
	log.Printf("Issuing credential for subject: %+v with Issuer: %s", subject, issuerDid)
	return nil
}
