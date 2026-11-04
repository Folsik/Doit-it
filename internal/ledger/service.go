package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Posting describes one immutable line in a posted journal transaction.
type Posting struct {
	AccountID string
	Currency  string
	Side      string
	Amount    int64
}

// Service writes balanced journal transactions using the caller's SQL transaction.
// The caller owns commit/rollback so transfer, idempotency, projection and journal
// records remain atomic.
type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) PostTransfer(ctx context.Context, tx pgx.Tx, transferID, fromAccountID, toAccountID, currency string, amount int64) error {
	if tx == nil || transferID == "" || fromAccountID == "" || toAccountID == "" ||
		fromAccountID == toAccountID || currency == "" || amount <= 0 {
		return errors.New("invalid ledger transfer")
	}

	var journalID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO ledger_transactions(transfer_id,currency) VALUES($1,$2) RETURNING id`,
		transferID, currency,
	).Scan(&journalID); err != nil {
		return fmt.Errorf("create ledger transaction: %w", err)
	}

	postings := []Posting{
		{AccountID: fromAccountID, Currency: currency, Side: "debit", Amount: amount},
		{AccountID: toAccountID, Currency: currency, Side: "credit", Amount: amount},
	}
	for _, posting := range postings {
		if _, err := tx.Exec(ctx,
			`INSERT INTO ledger_postings(ledger_transaction_id,account_id,currency,side,amount_minor)
			 VALUES($1,$2,$3,$4,$5)`,
			journalID, posting.AccountID, posting.Currency, posting.Side, posting.Amount,
		); err != nil {
			return fmt.Errorf("insert ledger posting: %w", err)
		}
	}
	return nil
}
