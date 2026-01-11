package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	sdk "github.com/bradtumy/credential-service/sdk/go"
)

func main() {
	client := sdk.NewClientFromEnv()
	if client.IssuerURL == "" || client.VerifierURL == "" {
		log.Fatal("ISSUER_URL and VERIFIER_URL (or GATEWAY_URL) must be set")
	}
	if client.GatewayURL == "" {
		client.GatewayURL = client.VerifierURL
	}

	issued, err := client.IssueVC(context.Background(), sdk.IssueRequest{
		SubjectDID: "did:example:alice",
		TTLSeconds: 600,
		Claims:     map[string]any{"aud": "sample-api", "role": "member"},
	})
	if err != nil {
		log.Fatalf("issue failed: %v", err)
	}
	fmt.Println("Issued VC-JWT:", issued.Credential)

	verified, err := client.Verify(context.Background(), issued.Credential)
	if err != nil {
		log.Fatalf("verify failed: %v", err)
	}
	fmt.Printf("Verified subject=%s valid=%v\n", verified.Subject, verified.Valid)

	decision, err := client.Authorize(context.Background(), sdk.AuthorizeRequest{
		Credential:       issued.Credential,
		ExpectedAudience: "sample-api",
		WantSyntheticJWT: true,
	})
	if err != nil {
		log.Fatalf("authorize failed: %v", err)
	}
	if !decision.Allowed {
		log.Fatalf("not authorized: %s", decision.Reason)
	}
	fmt.Println("Synthetic JWT:", decision.SyntheticJWT)

	if err := callSampleAPI(decision.SyntheticJWT); err != nil {
		log.Fatalf("sample API call failed: %v", err)
	}
}

func callSampleAPI(token string) error {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8090/api/resource", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}

	pretty, _ := json.MarshalIndent(body, "", "  ")
	fmt.Fprintf(os.Stdout, "Sample API response:\n%s\n", string(pretty))
	return nil
}
