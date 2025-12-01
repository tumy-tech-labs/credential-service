# Agent Sample

This sample demonstrates a basic AI agent flow:

1. Load a parent (human) credential (placeholder in `main.go`).
2. Start an agent session with delegated scope/TTL using the Go SDK helper.
3. Request gateway authorization to obtain a synthetic JWT.
4. Call a downstream API acting on behalf of the parent.
5. Refresh automatically when expired.

Run with Docker Compose using the existing stack:

```bash
docker compose up -d issuer gateway
GO_PARENT_TOKEN=<human-vc> go run ./samples/agent
```

Expected output includes the acting-on-behalf-of subject, scope, TTL, and downstream API response.
