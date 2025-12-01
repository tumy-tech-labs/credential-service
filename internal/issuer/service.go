package issuer

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/streadway/amqp"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpx"
)

// ed25519PrivateKeySigner implements crypto.Signer interface for Ed25519 private keys
type ed25519PrivateKeySigner struct {
	privateKey ed25519.PrivateKey
}

func (s *ed25519PrivateKeySigner) Public() crypto.PublicKey {
	return s.privateKey.Public()
}

func (s *ed25519PrivateKeySigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.privateKey.Sign(rand, digest, opts)
}

// QueueConfig describes RabbitMQ connection inputs.
type QueueConfig struct {
	Host string
	Port string
	User string
	Pass string
}

// Service contains dependencies required by the issuer HTTP handlers.
type Service struct {
	DB          *pgxpool.Pool
	VaultClient *api.Client
	BaseSchema  domain.BaseSchema
	HTTPClient  *http.Client
	Queue       QueueConfig
	ResolverURL string
}

// NewService constructs a Service with sensible defaults.
func NewService(cfg config.Config, db *pgxpool.Pool, vaultClient *api.Client) *Service {
	if vaultClient == nil {
		client, err := defaultVaultClient()
		if err == nil {
			vaultClient = client
		}
	}

	return &Service{
		DB:          db,
		VaultClient: vaultClient,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
		Queue: QueueConfig{
			Host: cfg.RabbitMQHost,
			Port: cfg.RabbitMQPort,
			User: cfg.RabbitMQUser,
			Pass: cfg.RabbitMQPass,
		},
		ResolverURL: cfg.ResolverURL,
	}
}

// IssueCredential handles issuing one or more credentials.
func (s *Service) IssueCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req domain.CredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		httpx.WriteAPIError(w, http.StatusBadRequest, "bad_request", "invalid request payload")
		return
	}

	if len(req.Subjects) == 0 {
		httpx.WriteAPIError(w, http.StatusBadRequest, "invalid_request", "no subjects provided")
		return
	}

	if err := s.enqueueBulkIssuance(req); err != nil {
		log.Printf("Failed to enqueue bulk issuance: %v", err)
		httpx.WriteAPIError(w, http.StatusInternalServerError, "queue_error", "failed to process request")
		return
	}

	resolverURL := fmt.Sprintf("%s?did=%s", s.ResolverURL, url.QueryEscape(req.IssuerDid))
	resp, err := s.HTTPClient.Get(resolverURL)
	if err != nil {
		log.Printf("Failed to fetch DID document from resolver: %v", err)
		httpx.WriteAPIError(w, http.StatusInternalServerError, "resolver_error", "failed to resolve DID")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Received non-OK response from resolver: %s", resp.Status)
		httpx.WriteAPIError(w, http.StatusInternalServerError, "resolver_error", "failed to resolve DID")
		return
	}

	vaultKey, err := s.getPrivateKeyFromVault(req.IssuerDid)
	if err != nil {
		log.Printf("Failed to retrieve private key from Vault: %v", err)
		httpx.WriteAPIError(w, http.StatusInternalServerError, "vault_error", "failed to issue credential")
		return
	}

	privateKey, err := domain.ParseEd25519PrivateKeyFromBase64(vaultKey)
	if err != nil {
		log.Printf("Failed to parse private key: %v", err)
		httpx.WriteAPIError(w, http.StatusInternalServerError, "key_error", "failed to issue credential")
		return
	}

	issuanceDate := time.Now().UTC().Format(time.RFC3339)
	expirationDate := time.Now().AddDate(1, 0, 0).UTC().Format(time.RFC3339)

	var credentials []domain.VerifiableCredential

	for _, subject := range req.Subjects {
		credentialID := uuid.New().String()
		credential, err := domain.BuildCredential(credentialID, req.IssuerDid, issuanceDate, expirationDate, subject)
		if err != nil {
			log.Printf("Failed to build credential: %v", err)
			httpx.WriteAPIError(w, http.StatusBadRequest, "credential_error", "failed to build credential")
			return
		}

		credentialJSON, err := json.Marshal(credential)
		if err != nil {
			log.Printf("Failed to marshal credential to JSON: %v", err)
			httpx.WriteAPIError(w, http.StatusInternalServerError, "marshal_error", "failed to process credential")
			return
		}

		// Create a signer from the private key
		signer := &ed25519PrivateKeySigner{privateKey: ed25519.PrivateKey(privateKey)}
		signature, err := domain.SignCredential(r.Context(), signer, credentialJSON)
		if err != nil {
			log.Printf("Failed to sign credential: %v", err)
			httpx.WriteAPIError(w, http.StatusInternalServerError, "signing_error", "failed to issue credential")
			return
		}

		credential = domain.AttachProof(credential, signature)

		proofJSON, err := json.Marshal(credential.Proof)
		if err != nil {
			log.Printf("Failed to marshal proof to JSON: %v", err)
			httpx.WriteAPIError(w, http.StatusInternalServerError, "marshal_error", "failed to process credential")
			return
		}

		if s.DB != nil {
			if err := s.storeCredential(r.Context(), credential, subject, credentialJSON, proofJSON); err != nil {
				log.Printf("Error inserting credential for subject %v: %v", subject, err)
				httpx.WriteAPIError(w, http.StatusInternalServerError, "db_error", "failed to store credential")
				return
			}
		}

		credentials = append(credentials, credential)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(credentials); err != nil {
		log.Printf("Failed to encode response: %v", err)
		httpx.WriteAPIError(w, http.StatusInternalServerError, "encode_error", "failed to issue credential")
		return
	}
}

func (s *Service) storeCredential(ctx context.Context, credential domain.VerifiableCredential, subject map[string]interface{}, credentialJSON []byte, proofJSON []byte) error {
	if s.DB == nil {
		return nil
	}
	_, err := s.DB.Exec(ctx,
		"INSERT INTO verifiable_credentials (did, issuer, credential, subject, issuance_date, expiration_date, proof) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		subject["id"],
		credential.Issuer,
		credentialJSON,
		subject,
		credential.IssuanceDate,
		credential.ExpirationDate,
		proofJSON,
	)
	return err
}

func (s *Service) getPrivateKeyFromVault(issuerDid string) (string, error) {
	client := s.VaultClient
	if client == nil {
		c, err := defaultVaultClient()
		if err != nil {
			return "", err
		}
		client = c
	}

	secretPath := fmt.Sprintf("secret/data/dids/%s", issuerDid)
	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault: %w", err)
	}
	if secret == nil {
		return "", fmt.Errorf("no secret found at path: %s", secretPath)
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("secret data is not in the expected format")
	}

	privateKeyBase64, ok := data["private_key"].(string)
	if !ok {
		return "", fmt.Errorf("private key not found in secret data")
	}

	return privateKeyBase64, nil
}

func (s *Service) enqueueBulkIssuance(req domain.CredentialRequest) error {
	conn, ch, queue, err := s.connectQueue()
	if err != nil {
		return err
	}
	defer conn.Close()
	defer ch.Close()

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	return ch.Publish(
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{ContentType: "application/json", Body: body},
	)
}

func (s *Service) connectQueue() (*amqp.Connection, *amqp.Channel, amqp.Queue, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", s.Queue.User, s.Queue.Pass, s.Queue.Host, s.Queue.Port)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, amqp.Queue{}, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, amqp.Queue{}, fmt.Errorf("failed to open a channel: %w", err)
	}

	queue, err := ch.QueueDeclare(
		"credential_issuance_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, amqp.Queue{}, fmt.Errorf("failed to declare queue: %w", err)
	}

	return conn, ch, queue, nil
}

// StartCredentialIssuanceWorker processes queued issuance jobs.
func (s *Service) StartCredentialIssuanceWorker(ctx context.Context) {
	conn, ch, queue, err := s.connectQueue()
	if err != nil {
		log.Printf("Failed to start issuance worker: %v", err)
		return
	}

	defer conn.Close()
	defer ch.Close()

	msgs, err := ch.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Printf("Failed to register a consumer: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-msgs:
			var req domain.CredentialRequest
			if err := json.Unmarshal(msg.Body, &req); err != nil {
				log.Printf("Failed to decode bulk issuance request: %v", err)
				continue
			}

			for _, subject := range req.Subjects {
				if err := s.issueCredentialForSubject(ctx, req.IssuerDid, subject); err != nil {
					log.Printf("Failed to issue credential for subject: %+v, error: %v", subject, err)
				}
			}
		}
	}
}

func (s *Service) issueCredentialForSubject(ctx context.Context, issuerDid string, subject map[string]interface{}) error {
	log.Printf("Issuing credential for subject: %+v with Issuer: %s", subject, issuerDid)
	// Placeholder for real issuance. Ensure context is used for future operations.
	_ = ctx
	return nil
}

func defaultVaultClient() (*api.Client, error) {
	config := api.DefaultConfig()
	if err := config.ReadEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to read Vault environment: %w", err)
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	return client, nil
}
