# Accounts, double-entry ledger, and internal P2P implementation plan

Status: approved design; implementation work in progress.

## Scope

Extend the existing account and transfer modules without replacing unrelated onboarding, authentication, KYC, profile, card, or session flows. Keep internal P2P transfers within the prototype boundary; no external payment rails or real funds.

## Current-state constraints

- `accounts.balance_minor` is currently mutated directly by the transfer handler.
- `ledger_entries` currently stores signed per-account amounts and is not independently enforced as a balanced journal.
- `transfers` and `transfer_idempotency` are already part of the API contract.
- Existing account creation paths include registration and card-choice onboarding.
- Migrations are embedded from `internal/store/migrations/*.sql` and applied in filename order.

## Target model

### Accounts

Retain existing account IDs, user ownership, currency, status, and API compatibility. Add a booked-balance projection that can be reconciled against posted ledger lines. Available balance is booked balance less active holds. Account lifecycle remains active/frozen/closed; frozen or closed accounts cannot initiate or receive P2P movements.

### Ledger

Use immutable journal transactions with at least two postings per transaction. Every posted transaction must balance to zero per currency. Posting records carry account, currency, side (debit/credit), and positive integer minor-unit amount. Enforce balanced posting at the database transaction boundary; do not rely only on application validation.

A correction is a compensating reversal linked to the original journal transaction. Posted journal entries are never edited or deleted by application code.

### Holds

Represent holds separately from posted movements. Holds reduce available balance but do not alter booked balance or create ledger postings. Hold creation/release must be transactional and idempotent.

### P2P flow

Within one PostgreSQL transaction:
1. Reserve/check idempotency key and payload fingerprint.
2. Lock source and destination accounts in deterministic ID order.
3. Validate ownership, status, currency, positive amount, and available funds.
4. Create transfer record.
5. Post balanced debit/credit journal transaction.
6. Update account balance projection (if retained) from the same transaction.
7. Persist idempotency result and commit.

The API response shape and existing endpoint remain compatible. Replayed requests return the original transfer; same key with a different payload returns conflict.

## Migration strategy

- Never rewrite already-applied migrations.
- Add a new forward-only migration.
- Preserve existing account/transfer IDs and historical signed ledger rows.
- Backfill historical rows into the new journal representation deterministically, grouped by transfer.
- Validate that each migrated transfer has exactly one debit and one credit of equal magnitude and currency before enabling the new write path.
- Abort migration if historical data is inconsistent; do not silently discard or fabricate accounting records.
- Keep compatibility projection columns until reconciliation tests prove they can be removed in a separate change.

## Acceptance criteria

- Double-entry invariants are enforced for every posted journal transaction.
- P2P is atomic: no transfer without both postings and no postings without transfer.
- Available balance cannot become negative, including under concurrent requests.
- Account ownership and lifecycle checks are enforced server-side.
- Idempotency handles concurrent duplicate requests and mismatched payload reuse.
- Account and transaction history API contracts remain compatible.
- Tests cover migration/backfill, balanced postings, rejection cases, retries, and concurrent spending.
- No unrelated modules or external payment rails are changed.

## Verification plan

Run targeted package tests, PostgreSQL integration tests against the migration chain, `go test ./...`, `go vet ./...`, and build. Review migration locking/backfill and compare API responses before/after. Report checks that cannot be run in the development environment explicitly.
