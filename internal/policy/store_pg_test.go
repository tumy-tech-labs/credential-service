package policy

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestPGStoreCreateAndGet(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	store := NewPGStore(db)
	policy := &Policy{TenantID: "tenant", Name: "test", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"orders"}, Subjects: []string{"any"}, Priority: 1, Enabled: true}

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO policies (tenant_id, name, description, effect, actions, resources, subjects, conditions, priority, enabled) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at, updated_at")).
		WithArgs(policy.TenantID, policy.Name, policy.Description, policy.Effect, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), policy.Priority, policy.Enabled).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(1), now, now))

	if err := store.CreatePolicy(context.Background(), policy); err != nil {
		t.Fatalf("create policy: %v", err)
	}

	condJSON := []byte(`{"claim_equals":{"role":"admin"}}`)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tenant_id, name, description, effect, actions, resources, subjects, conditions, priority, enabled, created_at, updated_at FROM policies WHERE tenant_id = $1 AND id = $2")).
		WithArgs(policy.TenantID, policy.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "description", "effect", "actions", "resources", "subjects", "conditions", "priority", "enabled", "created_at", "updated_at"}).
			AddRow(policy.ID, policy.TenantID, policy.Name, policy.Description, policy.Effect, pq.StringArray{"read"}, pq.StringArray{"orders"}, pq.StringArray{"any"}, condJSON, policy.Priority, policy.Enabled, now, now))

	fetched, err := store.GetPolicy(context.Background(), policy.TenantID, policy.ID)
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if fetched.ID != policy.ID || fetched.Effect != policy.Effect {
		t.Fatalf("unexpected fetched policy: %+v", fetched)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPGStoreUpdateAndDelete(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	store := NewPGStore(db)
	policy := &Policy{ID: 3, TenantID: "tenant", Name: "test", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"orders"}, Subjects: []string{"any"}, Priority: 1, Enabled: true}

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE policies SET name = $1, description = $2, effect = $3, actions = $4, resources = $5, subjects = $6, conditions = $7, priority = $8, enabled = $9, updated_at = NOW() WHERE tenant_id = $10 AND id = $11 RETURNING updated_at")).
		WithArgs(policy.Name, policy.Description, policy.Effect, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), policy.Priority, policy.Enabled, policy.TenantID, policy.ID).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(time.Now()))

	if err := store.UpdatePolicy(context.Background(), policy); err != nil {
		t.Fatalf("update policy: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM policies WHERE tenant_id = $1 AND id = $2")).
		WithArgs(policy.TenantID, policy.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.DeletePolicy(context.Background(), policy.TenantID, policy.ID); err != nil {
		t.Fatalf("delete policy: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
