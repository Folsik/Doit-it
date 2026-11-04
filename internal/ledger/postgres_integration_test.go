package ledger

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/folsik/doit-it/internal/store"
)

func postgresIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test skipped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil { t.Fatalf("connect admin database: %v", err) }
	t.Cleanup(admin.Close)

	schema := "it_" + strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), "-", "")
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	})

	u, err := url.Parse(dsn)
	if err != nil { t.Fatalf("parse TEST_DATABASE_URL: %v", err) }
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil { t.Fatalf("connect isolated schema: %v", err) }
	t.Cleanup(pool.Close)

	st := &store.Store{DB: pool}
	if err := st.Migrate(ctx); err != nil { t.Fatalf("apply all migrations: %v", err) }
	return pool
}

func seedTransfer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, amount int64) (string, string, string) {
	t.Helper()
	var userID, fromID, toID, transferID string
	if err := pool.QueryRow(ctx, `INSERT INTO users(email,password_hash) VALUES($1,'test-hash') RETURNING id`,
		fmt.Sprintf("ledger-it-%d@example.test", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO accounts(user_id,currency,balance_minor) VALUES($1,'EUR',100000) RETURNING id`, userID).Scan(&fromID); err != nil {
		t.Fatalf("insert source account: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO accounts(user_id,currency,balance_minor) VALUES($1,'EUR',0) RETURNING id`, userID).Scan(&toID); err != nil {
		t.Fatalf("insert destination account: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO transfers(reference,from_account_id,to_account_id,amount_minor,currency,status)
		VALUES($1,$2,$3,$4,'EUR','completed') RETURNING id`,
		fmt.Sprintf("ledger-it-%d", time.Now().UnixNano()), fromID, toID, amount).Scan(&transferID); err != nil {
		t.Fatalf("insert transfer: %v", err)
	}
	return transferID, fromID, toID
}

func TestPostgresLedgerIntegration(t *testing.T) {
	ctx := context.Background()
	pool := postgresIntegrationPool(t)
	service := NewService()

	t.Run("posts balanced transfer atomically", func(t *testing.T) {
		transferID, fromID, toID := seedTransfer(t, ctx, pool, 1250)
		tx, err := pool.Begin(ctx)
		if err != nil { t.Fatal(err) }
		if err := service.PostTransfer(ctx, tx, transferID, fromID, toID, "EUR", 1250); err != nil {
			_ = tx.Rollback(ctx); t.Fatalf("PostTransfer: %v", err)
		}
		if err := tx.Commit(ctx); err != nil { t.Fatalf("commit balanced journal: %v", err) }

		var postings int
		var debits, credits int64
		if err := pool.QueryRow(ctx, `SELECT COUNT(*),
			COALESCE(SUM(amount_minor) FILTER(WHERE side='debit'),0)::BIGINT,
			COALESCE(SUM(amount_minor) FILTER(WHERE side='credit'),0)::BIGINT
			FROM ledger_postings lp JOIN ledger_transactions lt ON lt.id=lp.ledger_transaction_id
			WHERE lt.transfer_id=$1`, transferID).Scan(&postings,&debits,&credits); err != nil { t.Fatal(err) }
		if postings != 2 || debits != 1250 || credits != 1250 {
			t.Fatalf("unexpected journal: postings=%d debits=%d credits=%d", postings,debits,credits)
		}
		var legacyCount int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM ledger_entries WHERE transfer_id=$1", transferID).Scan(&legacyCount); err != nil { t.Fatal(err) }
		if legacyCount != 2 { t.Fatalf("expected 2 compatibility entries, got %d", legacyCount) }

		if _, err := pool.Exec(ctx, "UPDATE ledger_postings SET amount_minor=1 WHERE ledger_transaction_id=(SELECT id FROM ledger_transactions WHERE transfer_id=$1)", transferID); err == nil {
			t.Fatal("expected immutable posting update to fail")
		}
	})

	t.Run("rolls back journal and compatibility entries", func(t *testing.T) {
		transferID, fromID, toID := seedTransfer(t, ctx, pool, 500)
		tx, err := pool.Begin(ctx)
		if err != nil { t.Fatal(err) }
		if err := service.PostTransfer(ctx, tx, transferID, fromID, toID, "EUR", 500); err != nil { t.Fatal(err) }
		if err := tx.Rollback(ctx); err != nil { t.Fatal(err) }
		var journals, postings, legacy int
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ledger_transactions WHERE transfer_id=$1", transferID).Scan(&journals)
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ledger_postings lp JOIN ledger_transactions lt ON lt.id=lp.ledger_transaction_id WHERE lt.transfer_id=$1", transferID).Scan(&postings)
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ledger_entries WHERE transfer_id=$1", transferID).Scan(&legacy)
		if journals+postings+legacy != 0 { t.Fatalf("rollback left rows: journal=%d postings=%d legacy=%d",journals,postings,legacy) }
	})

	t.Run("deferred constraint rejects unbalanced journal", func(t *testing.T) {
		transferID, fromID, _ := seedTransfer(t, ctx, pool, 700)
		tx, err := pool.Begin(ctx)
		if err != nil { t.Fatal(err) }
		var journalID string
		if err := tx.QueryRow(ctx, "INSERT INTO ledger_transactions(transfer_id,currency) VALUES($1,'EUR') RETURNING id", transferID).Scan(&journalID); err != nil { t.Fatal(err) }
		if _, err := tx.Exec(ctx, "INSERT INTO ledger_postings(ledger_transaction_id,account_id,currency,side,amount_minor) VALUES($1,$2,'EUR','debit',700)", journalID, fromID); err != nil { t.Fatal(err) }
		if err := tx.Commit(ctx); err == nil { t.Fatal("expected deferred balance constraint to reject one-sided journal") }
	})
}
