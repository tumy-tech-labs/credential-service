package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/bradtumy/credential-service/sdk/go"
)

func main() {
	ctx := context.Background()
	client := sdk.NewMagicLocalClient()

	parentClaims := map[string]any{"aud": "sample-api", "scope": []string{"orders:read", "orders:write"}}
	parentVC, err := client.IssueVC(ctx, "did:example:human-demo", 600, parentClaims)
	if err != nil {
		log.Fatalf("issue parent vc: %v", err)
	}

	agentClaims := map[string]any{"aud": "sample-api"}
	agentVC, err := client.SpawnAgent(ctx, parentVC, "did:example:agent-demo", []string{"orders:read"}, 120, agentClaims)
	if err != nil {
		log.Fatalf("delegate agent vc: %v", err)
	}

	decision, err := client.Authorize(ctx, []string{parentVC, agentVC}, "orders", "read", "sample-api", true)
	if err != nil {
		log.Fatalf("authorize chain: %v", err)
	}

	tokenPreview := decision.SyntheticJWT
	if len(tokenPreview) > 24 {
		tokenPreview = tokenPreview[:24]
	}

	fmt.Printf("allowed=%v acting_on_behalf_of=%s synthetic_jwt_prefix=%s...\n", decision.Allowed, decision.ActingOnBehalfOf, tokenPreview)
}
