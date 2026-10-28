# Doit Backend — Go MVP scaffold

A Go-only backend starter for a mobile-first, EU-oriented personal-finance app. It exposes a JSON REST API for a future iOS/Android/web client. The starter is intentionally a **modular monolith**: one deployable Go service, PostgreSQL as the persistent store.

> **Not a real bank / not production-ready.** Transfers are internal prototype ledger movements only. No SEPA, card issuer, open-banking, KYC/AML, sanctions-screening, safeguarding, licensing, external payment processor, push notification provider, or real funds are connected. Do not accept customer money or deploy with real users/funds in this state.

## Included

- Go `net/http` API (Go 1.23+)
- PostgreSQL schema and automatic migrations
- Password hashing with bcrypt
- Short-lived Ed25519-signed JWT access tokens
- Registration/login, profile, dashboard, accounts, transaction history/details, internal transfers
- Basic session listing/revocation endpoints (session creation/refresh and token revocation still need implementation)
- Docker Compose for local development

## Run locally

```bash
cp .env.example .env
# Start PostgreSQL only; if you use Docker Compose for both services, `docker compose up --build` is enough.
docker compose up --build
```

API listens at `http://localhost:8080`. Check `GET /healthz`.

## Quick test

```bash
curl -s http://localhost:8080/healthz
curl -s -X POST http://localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"change-this-password-123","full_name":"Demo User","country_code":"DE"}'
```

Copy the returned `access_token` into `Authorization: Bearer <token>` for protected endpoints.

## API outline

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/healthz` | Health check + database ping |
| POST | `/v1/auth/register` | Create user and empty EUR account |
| POST | `/v1/auth/login` | Verify credentials and issue access token |
| POST | `/v1/auth/logout` | Prototype logout response (does not revoke token yet) |
| GET | `/v1/me` | Current profile |
| PATCH | `/v1/me` | Update profile preferences |
| GET | `/v1/dashboard` | Balance summary for home screen |
| GET | `/v1/accounts` | Accounts and balances |
| GET | `/v1/transactions?limit=50&offset=0&status=completed` | Transaction history |
| GET | `/v1/transactions/{id}` | Transaction details |
| POST | `/v1/transfers` | Internal transfer between two accounts owned by different users |
| GET | `/v1/security/sessions` | List active sessions (empty until session persistence is wired into auth) |
| DELETE | `/v1/security/sessions/{id}` | Revoke persisted session |

### Internal transfer body

```json
{
  "from_account_id": "<source-account-uuid>",
  "to_account_id": "<destination-account-uuid>",
  "amount_minor": 1250,
  "currency": "EUR",
  "note": "Dinner",
  "idempotency_key": "client-generated-unique-key-001"
}
```

Amounts are integers in minor units (EUR cents); never use floating point for money in business logic. The current transfer implementation is a prototype and needs a dedicated idempotency table, audit trail, stronger recipient ownership rules, transaction limits, risk controls, and independent review before production.

## Security and production gaps to close

1. Persist and rotate Ed25519 JWT keys; currently generated at process startup so tokens become invalid after restart. Use a KMS/HSM or secret manager.
2. Implement refresh-token rotation, server-side session creation/revocation, device metadata, logout semantics, MFA enrollment/challenge/recovery, and rate limits.
3. Add email verification, password reset, login throttling, breached-password checks, audit events, secrets management, TLS, observability, backups and disaster recovery.
4. Harden transfer logic with a dedicated idempotency table scoped to user+endpoint, ledger invariants, account ownership/recipient checks, limits, fraud/risk checks and concurrency tests.
5. Add API contract/OpenAPI, unit/integration tests, CI, dependency scanning and threat modeling.
6. For EU launch, obtain qualified legal/compliance advice on licensing/partner-bank model, PSD2/PSD3/PSR applicability, AML/KYC, GDPR, sanctions, SEPA/payment-provider integration and safeguarding. Eligibility should be based on applicable law and documented risk policy, not a blanket citizenship heuristic.

## Architecture

```text
Mobile clients (iOS / Android) + Web client
                  |
              HTTPS / JSON
                  |
          Go REST API (modular monolith)
          |       |       |       |
         Auth   Accounts Transactions Profile/Security
                  |
              PostgreSQL
```

Redis, NATS/Kafka and external payment providers should be introduced only when requirements justify them. For a first MVP, Postgres transactions provide a simpler and safer starting point for internal ledger consistency.
