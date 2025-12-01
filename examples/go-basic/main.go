package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/bradtumy/credential-service/sdk/go"
)

func main() {
	client := &sdk.Client{BaseURL: "http://localhost:8080"}
	ctx := context.Background()

	issued, err := client.IssueCredential(ctx, sdk.IssueRequest{
		SubjectDID: "did:example:alice",
		TTLSeconds: int64((10 * time.Minute).Seconds()),
		Claims:     map[string]interface{}{"aud": "example-api", "role": "admin"},
	})
	if err != nil {
		log.Fatalf("issue credential: %v", err)
	}

	decision, err := client.GatewayAuthorize(ctx, sdk.GatewayAuthorizeRequest{
		Credential:       issued.Credential,
		ExpectedAudience: "example-api",
		WantSyntheticJWT: true,
	})
	if err != nil {
		log.Fatalf("gateway authorize: %v", err)
	}

	fmt.Printf("Subject: %s\n", decision.Subject)
	fmt.Printf("Acting on behalf of: %s\n", decision.ActingOnBehalfOf)
	fmt.Printf("Claims: %#v\n", decision.Claims)
	if decision.SyntheticJWT != "" {
		fmt.Printf("Synthetic JWT: %s\n", decision.SyntheticJWT)
	}
}
