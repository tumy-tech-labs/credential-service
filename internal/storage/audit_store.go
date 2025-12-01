package storage

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/bradtumy/credential-service/internal/logging"
)

// AuditStore persists audit events for compliance.
type AuditStore interface {
	InsertAuditEvent(ctx context.Context, event logging.AuditEvent) error
}

// PGAuditStore implements AuditStore against Postgres.
type PGAuditStore struct {
	db *sql.DB
}

// NewPGAuditStore creates a new Postgres-backed audit store.
func NewPGAuditStore(db *sql.DB) *PGAuditStore {
	return &PGAuditStore{db: db}
}

// InsertAuditEvent writes an audit event row.
func (s *PGAuditStore) InsertAuditEvent(ctx context.Context, event logging.AuditEvent) error {
	metadataBytes, _ := json.Marshal(event.Metadata)
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO audit_events (
                        timestamp, correlation_id, event_type, tenant_id, user_id, subject_did, issuer_did,
                        resource, action, outcome, reason, ip_address, user_agent, metadata
                ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		event.Timestamp, event.CorrelationID, event.EventType, event.TenantID, event.UserID,
		event.SubjectDID, event.IssuerDID, event.Resource, event.Action, event.Outcome,
		event.Reason, event.IPAddress, event.UserAgent, metadataBytes,
	)
	return err
}
