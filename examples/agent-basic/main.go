package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sdk "github.com/bradtumy/credential-service/sdk/go"
)

func main() {
	client := &sdk.Client{
		IssuerURL:   "http://localhost:8080",
		VerifierURL: "http://localhost:8081",
		GatewayURL:  "http://localhost:8081",
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
	}
	agent := &sdk.AgentClient{SDK: client}

	parent := ""
	session, err := agent.StartAgentSession(context.Background(), parent, "did:example:agent", []string{"read"}, 2*time.Minute)
	if err != nil {
		panic(err)
	}

	resp, err := agent.CallAuthorized(context.Background(), session, http.MethodGet, "http://localhost:8080/healthz", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("status", resp.StatusCode)
}
