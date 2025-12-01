# Credential Service Go SDK

A lightweight Go client for issuing, delegating, verifying, and authorizing Verifiable Credentials with the Credential Service.

## Installation

```bash
go get github.com/bradtumy/credential-service/sdk/go
```

## Usage

```go
package main

import (
    "context"
    "log"

    sdk "github.com/bradtumy/credential-service/sdk/go"
)

func main() {
    client := &sdk.Client{BaseURL: "http://localhost:8080"}
    issued, err := client.IssueCredential(context.Background(), sdk.IssueRequest{
        SubjectDID: "did:example:alice",
        TTLSeconds: 600,
        Claims: map[string]interface{}{"aud": "example-api"},
    })
    if err != nil {
        log.Fatalf("issue failed: %v", err)
    }

    decision, err := client.GatewayAuthorize(context.Background(), sdk.GatewayAuthorizeRequest{
        Credential:       issued.Credential,
        ExpectedAudience: "example-api",
        WantSyntheticJWT: true,
    })
    if err != nil {
        log.Fatalf("authorize failed: %v", err)
    }

    log.Printf("allowed=%v subject=%s acting_for=%s synthetic_jwt=%s", decision.Allowed, decision.Subject, decision.ActingOnBehalfOf, decision.SyntheticJWT)
}
```

See the repository root `README.md` for the 5-minute quickstart and more examples.
