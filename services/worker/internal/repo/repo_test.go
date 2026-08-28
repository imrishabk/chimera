package repo

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/database"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("skip repo integration in short mode")
	}
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOSTNAME"), os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"))
	if os.Getenv("DB_HOSTNAME") == "" {
		t.Skip("DB env not set — skip repo test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := database.NewPostgresConnection(ctx, dsn)
	if err != nil {
		t.Skipf("db not reachable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("db ping failed: %v", err)
	}
	return pool
}

func TestUserRepoCreateFetch(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	r := NewUserRepository(pool)
	ctx := context.Background()
	uniq := uuid.New().String()[:8]
	u, err := r.CreateUser(ctx, "testuser_"+uniq, "test_"+uniq+"@example.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == uuid.Nil {
		t.Error("expected uuid")
	}
	fetched, err := r.FetchUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if fetched.Username != u.Username {
		t.Errorf("mismatch %s vs %s", fetched.Username, u.Username)
	}
}

func TestIngestJobRepoCRUD(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	// need a user and session first
	ur := NewUserRepository(pool)
	sr := NewSessionRepository(pool)
	jr := NewIngestJobRepository(pool)
	uniq := uuid.New().String()[:8]
	u, err := ur.CreateUser(ctx, "ingestuser_"+uniq, "ingest_"+uniq+"@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ses, err := sr.CreateSession(ctx, u.ID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	job := &model.IngestJob{SessionID: ses.ID, UserID: u.ID, Status: "pending", Source: "test", DocCount: 1}
	created, err := jr.Create(ctx, job)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if created.Status != "pending" {
		t.Errorf("want pending got %s", created.Status)
	}
	got, err := jr.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != created.ID {
		t.Error("id mismatch")
	}
	list, err := jr.ListBySession(ctx, ses.ID, 10, 0)
	if err != nil || len(list) == 0 {
		t.Fatalf("list: %v len %d", err, len(list))
	}
	updated, err := jr.UpdateStatus(ctx, created.ID, "completed", 5, "")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Status != "completed" || updated.DocCount != 5 {
		t.Errorf("update mismatch %+v", updated)
	}
}
