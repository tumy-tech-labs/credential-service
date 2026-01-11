package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	sdk "github.com/bradtumy/credential-service/sdk/go"
)

func main() {
	parentToken := ""
	if parentToken == "" {
		log.Fatal("parent credential required (placeholder)")
	}
	client := &sdk.Client{
		IssuerURL:   "http://localhost:8080",
		VerifierURL: "http://localhost:8081",
		GatewayURL:  "http://localhost:8081",
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
	}
	agent := &sdk.AgentClient{SDK: client}
	session, err := agent.StartAgentSession(context.Background(), parentToken, "did:example:agent", []string{"read"}, 5*time.Minute)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := agent.CallAuthorized(context.Background(), session, http.MethodGet, "http://localhost:8080/examples/s2s", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("agent call status", resp.StatusCode)
}
