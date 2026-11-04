-- Forward-only double-entry journal migration.
-- Preserve the legacy signed ledger_entries table for read compatibility.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM transfers t
        LEFT JOIN ledger_entries le ON le.transfer_id = t.id
        WHERE t.status = 'completed'
        GROUP BY t.id, t.from_account_id, t.to_account_id, t.amount_minor, t.currency
        HAVING COUNT(le.id) <> 2
            OR COUNT(*) FILTER (
                WHERE le.account_id = t.from_account_id
                  AND le.amount_minor = -t.amount_minor
                  AND le.currency = t.currency
            ) <> 1
            OR COUNT(*) FILTER (
                WHERE le.account_id = t.to_account_id
                  AND le.amount_minor = t.amount_minor
                  AND le.currency = t.currency
            ) <> 1
    ) THEN
        RAISE EXCEPTION 'double-entry migration aborted: completed transfer history is inconsistent';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM accounts a
        LEFT JOIN (
            SELECT account_id, SUM(amount_minor)::BIGINT AS ledger_balance
            FROM ledger_entries
            GROUP BY account_id
        ) l ON l.account_id = a.id
        WHERE a.balance_minor <> COALESCE(l.ledger_balance, 0)
    ) THEN
        RAISE EXCEPTION 'double-entry migration aborted: account balance does not reconcile to legacy ledger';
    END IF;
END $$;

CREATE TABLE ledger_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_id UUID NOT NULL UNIQUE REFERENCES transfers(id),
    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'posted' CHECK (status IN ('posted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ledger_postings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ledger_transaction_id UUID NOT NULL REFERENCES ledger_transactions(id),
    account_id UUID NOT NULL REFERENCES accounts(id),
    currency CHAR(3) NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('debit', 'credit')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_postings_account_created
    ON ledger_postings(account_id, created_at DESC);
CREATE INDEX idx_ledger_postings_transaction
    ON ledger_postings(ledger_transaction_id);

INSERT INTO ledger_transactions (transfer_id, currency, created_at)
SELECT t.id, t.currency, t.created_at
FROM transfers t
WHERE t.status = 'completed'
ORDER BY t.created_at, t.id;

INSERT INTO ledger_postings (
    ledger_transaction_id, account_id, currency, side, amount_minor, created_at
)
SELECT lt.id, le.account_id, le.currency,
       CASE WHEN le.amount_minor < 0 THEN 'debit' ELSE 'credit' END,
       ABS(le.amount_minor), le.created_at
FROM ledger_entries le
JOIN ledger_transactions lt ON lt.transfer_id = le.transfer_id
ORDER BY le.created_at, le.id;

CREATE FUNCTION enforce_balanced_ledger_transaction() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    target_id UUID;
    posting_count BIGINT;
    debit_total NUMERIC;
    credit_total NUMERIC;
    currency_count BIGINT;
BEGIN
    IF TG_TABLE_NAME = 'ledger_transactions' THEN
        target_id := NEW.id;
    ELSE
        target_id := COALESCE(NEW.ledger_transaction_id, OLD.ledger_transaction_id);
    END IF;
    SELECT COUNT(*),
           COALESCE(SUM(amount_minor) FILTER (WHERE side = 'debit'), 0),
           COALESCE(SUM(amount_minor) FILTER (WHERE side = 'credit'), 0),
           COUNT(DISTINCT currency)
      INTO posting_count, debit_total, credit_total, currency_count
      FROM ledger_postings
     WHERE ledger_transaction_id = target_id;

    IF posting_count < 2 OR debit_total <> credit_total OR currency_count <> 1 THEN
        RAISE EXCEPTION 'ledger transaction % is not balanced', target_id;
    END IF;
    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER ledger_transaction_must_balance
AFTER INSERT OR UPDATE OR DELETE ON ledger_postings
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_balanced_ledger_transaction();

CREATE CONSTRAINT TRIGGER ledger_transaction_must_have_postings
AFTER INSERT ON ledger_transactions
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_balanced_ledger_transaction();

CREATE FUNCTION prevent_ledger_posting_mutation() RETURNS trigger
LANGUAGE plpgsql AS $
BEGIN
    RAISE EXCEPTION 'posted ledger records are immutable; use a compensating transaction';
END;
$;

CREATE TRIGGER ledger_postings_are_immutable
BEFORE UPDATE OR DELETE ON ledger_postings
FOR EACH ROW EXECUTE FUNCTION prevent_ledger_posting_mutation();

CREATE TRIGGER ledger_transactions_are_immutable
BEFORE UPDATE OR DELETE ON ledger_transactions
FOR EACH ROW EXECUTE FUNCTION prevent_ledger_posting_mutation();

-- Verify migrated journal balances before allowing startup to continue.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM ledger_transactions lt
        LEFT JOIN ledger_postings lp ON lp.ledger_transaction_id = lt.id
        GROUP BY lt.id
        HAVING COUNT(lp.id) < 2
            OR COALESCE(SUM(lp.amount_minor) FILTER (WHERE lp.side = 'debit'), 0)
               <> COALESCE(SUM(lp.amount_minor) FILTER (WHERE lp.side = 'credit'), 0)
            OR COUNT(DISTINCT lp.currency) <> 1
    ) THEN
        RAISE EXCEPTION 'double-entry migration aborted: migrated journal is not balanced';
    END IF;
END $$;
