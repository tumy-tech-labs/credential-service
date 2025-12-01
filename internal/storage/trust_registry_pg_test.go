package storage

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPGTrustRegistry_AddTrustedIssuerIdempotent(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()

	registry := NewPGTrustRegistry(mockDB)

        insertRegex := regexp.QuoteMeta("INSERT INTO tenant_trusted_issuers (tenant_id, issuer_did, metadata) VALUES ($1, $2, '{}') ON CONFLICT (tenant_id, issuer_did) DO UPDATE SET metadata = EXCLUDED.metadata")
	mock.ExpectExec(insertRegex).
		WithArgs("tenant-1", "did:example:issuer").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(insertRegex).
		WithArgs("tenant-1", "did:example:issuer").
		WillReturnResult(sqlmock.NewResult(1, 0))

	if err := registry.AddTrustedIssuer(context.Background(), "tenant-1", "did:example:issuer"); err != nil {
		t.Fatalf("unexpected error on insert: %v", err)
	}
	if err := registry.AddTrustedIssuer(context.Background(), "tenant-1", "did:example:issuer"); err != nil {
		t.Fatalf("unexpected error on duplicate insert: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPGTrustRegistry_IsTrustedIssuer(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()

	registry := NewPGTrustRegistry(mockDB)

        selectRegex := regexp.QuoteMeta("SELECT 1 FROM tenant_trusted_issuers WHERE tenant_id = $1 AND issuer_did = $2")

	mock.ExpectQuery(selectRegex).
		WithArgs("tenant-1", "did:example:issuer").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))

	mock.ExpectQuery(selectRegex).
		WithArgs("tenant-1", "did:example:unknown").
		WillReturnError(sql.ErrNoRows)

	trusted, err := registry.IsTrustedIssuer(context.Background(), "tenant-1", "did:example:issuer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !trusted {
		t.Fatalf("expected issuer to be trusted")
	}

	trusted, err = registry.IsTrustedIssuer(context.Background(), "tenant-1", "did:example:unknown")
	if err != nil {
		t.Fatalf("unexpected error on missing: %v", err)
	}
	if trusted {
		t.Fatalf("expected unknown issuer to be untrusted")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
