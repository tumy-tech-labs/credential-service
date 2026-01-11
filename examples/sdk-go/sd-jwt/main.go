package main

import (
	"context"
	"fmt"
	sdk "github.com/bradtumy/credential-service/sdk/go"
	"log"
	"os"
)

func main() {
	issuerURL := env("ISSUER_URL", "http://localhost:8080")
	verifierURL := env("VERIFIER_URL", "http://localhost:8081")
	subjectDID := os.Getenv("ALICE_DID")
	if subjectDID == "" {
		log.Fatalf("ALICE_DID is required. Generate one via ./bin/keygen -did-only and export ALICE_DID before running.")
	}

	client := &sdk.Client{
		IssuerURL:   issuerURL,
		VerifierURL: verifierURL,
	}
	ctx := context.Background()

	issued, err := client.IssueSDJWT(ctx, sdk.IssueRequest{
		SubjectDID: subjectDID,
		TTLSeconds: 600,
		Claims: map[string]interface{}{
			"email":      "alice@example.com",
			"department": "engineering",
			"scope":      "read:orders",
		},
	})
	if err != nil {
		log.Fatalf("issue sd-jwt: %v", err)
	}
	fmt.Println("SD-JWT:", issued.Credential)
	fmt.Println("Disclosures:", issued.Disclosures)

	verify, err := client.Verify(ctx, issued.Credential)
	if err != nil {
		log.Fatalf("verify sd-jwt: %v", err)
	}
	fmt.Println("Verification valid:", verify.Valid)
}

func env(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
