# Doit Backend — Go MVP scaffold

A Go-only backend starter for a mobile-first, EU-oriented personal-finance app. It exposes a JSON REST API for future iOS, Android and web clients. The application is a modular monolith with PostgreSQL as its persistent store.

> **Prototype only — not a real bank and not production-ready.** Transfers are internal prototype ledger movements. No SEPA, card issuer, open banking, KYC/AML, sanctions screening, safeguarding, licensing, external payment processor or real funds are connected. Do not accept customer money or deploy with real users/funds.

## Stack

- Go 1.23+ and standard net/http
- PostgreSQL + pgx
- SQL migrations embedded in the binary
- bcrypt password hashing
- Ed25519-signed short-lived JWT access tokens
- User-scoped idempotency for internal transfer requests
- Docker Compose for local development

## Run

### Docker Compose

    docker compose up --build

The API listens on http://localhost:8080. Check health with curl http://localhost:8080/healthz.

The Dockerfile builds the root main.go entrypoint. You can also run the app directly from the repository root after starting PostgreSQL:

    cp .env.example .env
    # Set the variables from .env in your shell
    go run .

### Quick test

    curl -s http://localhost:8080/healthz
    curl -s -X POST http://localhost:8080/v1/auth/register \
      -H 'Content-Type: application/json' \
      -d '{"email":"demo@example.com","password":"change-this-password-123","full_name":"Demo User","country_code":"DE"}'

Use the returned access_token as Authorization: Bearer <token> for protected endpoints.

## API outline

| Method | Endpoint | Purpose |
|---|---|---|
| GET | /healthz | Health check + database ping |
| GET | /v1/onboarding/countries | Initial supported EU phone regions |\n| POST | /v1/onboarding/send-otp | Request phone verification code (development delivery only) |\n| POST | /v1/onboarding/verify-otp | Verify OTP and receive short-lived setup token |\n| POST | /v1/onboarding/set-pin | Create phone-first user with a six-digit app PIN |\n| POST | /v1/auth/phone-login | Sign in with phone number and app PIN |\n| PUT | /v1/me/email | Add optional email to authenticated profile |\n| POST | /v1/cards/choose | Select Visa/Mastercard prototype and open EUR account |\n| POST | /v1/auth/register | Legacy email/password registration (compatibility) |
| POST | /v1/auth/login | Verify credentials and issue access token |
| POST | /v1/auth/logout | Revoke current access token until expiry |
| GET | /v1/me | Current profile |
| PATCH | /v1/me | Update profile preferences |
| GET | /v1/dashboard | Balance summary for home screen |
| GET | /v1/accounts | Accounts and balances in minor units |
| GET | /v1/transactions?limit=50&offset=0&status=completed | Transaction history |
| GET | /v1/transactions/{id} | Transaction details |
| POST | /v1/transfers | Internal transfer between accounts |
| GET | /v1/security/sessions | List persisted sessions (currently no session issuance flow) |
| DELETE | /v1/security/sessions/{id} | Revoke a persisted session |

### Internal transfer body

    {
      "from_account_id": "<source-account-uuid>",
      "to_account_id": "<destination-account-uuid>",
      "amount_minor": 1250,
      "currency": "EUR",
      "note": "Dinner",
      "idempotency_key": "client-generated-unique-key-001"
    }

Amounts are integers in minor units (EUR cents); money values are not represented as floating-point numbers by the API. Repeating the same key and request returns the original result; reusing a key with a different request returns a conflict. This is still a prototype ledger, not an external payment service.


## Phone-first onboarding (prototype)

The initial supported phone-region list is an explicit EU product-availability allowlist. It currently excludes Russia and Belarus, but **it is not a sanctions screening engine and does not decide eligibility based on nationality**. Production eligibility must additionally assess residence, identity, beneficial ownership, counterparties and transactions against the applicable sanctions and AML rules.

1. Fetch `GET /v1/onboarding/countries` and let the user select a supported country/phone calling code.
2. Send `POST /v1/onboarding/send-otp` with `{"country_code":"DE","phone_e164":"+4915112345678"}`.
3. Verify `POST /v1/onboarding/verify-otp` with `{"challenge_id":"<uuid>","code":"<six digits>"}`.
4. Set the app PIN with `POST /v1/onboarding/set-pin` and `{"verification_token":"<short-lived token>","pin":"482915"}`. PINs consisting of one repeated digit or ascending/descending sequences are rejected. The PIN is bcrypt-hashed and login failures are throttled.
5. The response includes an access token. Email is optional and can be added with authenticated `PUT /v1/me/email`.
6. On the home screen, choose `visa` or `mastercard` using authenticated `POST /v1/cards/choose`. The backend atomically creates an EUR account and a card-issuance request record.

In local development the OTP is returned as `development_code` and written to the development log. **Production OTP sending is deliberately disabled** until an SMS provider is integrated. Never expose the development code in a production client. Card selection creates a prototype issuance request only; actual card issuance requires a licensed issuer/programme and provider integration. This flow does not perform KYC, sanctions screening, age checks or regulated account-opening checks.
\n## Configuration

See .env.example. Important settings:

- DATABASE_URL: PostgreSQL connection string.
- HTTP_ADDR: listen address, default :8080.
- ACCESS_TOKEN_TTL: short-lived access token TTL, default 15m, maximum 1h.
- JWT_ISSUER: issuer claim for access tokens.
- JWT_PRIVATE_KEY: base64url-encoded 64-byte Ed25519 private key. Optional in development; **required when APP_ENV=production**. Keep it in a secret manager, never in source control. Development without this key generates an ephemeral key and invalidates tokens on restart.

The HTTP server does not terminate TLS. Production deployments must run behind a correctly configured TLS-terminating reverse proxy or load balancer and use a secure database connection.

## Security improvements in this branch

- Strict Bearer token parsing and JWT header/claim validation.
- Persistent Ed25519 key configuration for production.
- Access-token revocation on logout.
- JSON content-type enforcement, unknown-field rejection, trailing JSON rejection and a 1 MiB body limit.
- Input validation for registration/profile fields and generic invalid-credential responses with a dummy bcrypt check for unknown users.
- User-scoped transfer idempotency, deterministic account locking, integer money amounts and destination-balance overflow protection.
- Security response headers, request IDs and graceful HTTP shutdown.
- SQL migrations for transfer idempotency and revoked tokens.

## Remaining gaps before production

1. Implement refresh-token rotation, durable login sessions, device management and MFA enrollment/challenge/recovery.
2. Add distributed rate limiting, email verification, password reset, breached-password checks, and abuse monitoring.
3. Add comprehensive unit/integration/concurrency tests, API contract/OpenAPI, CI security scanning and dependency updates.
4. Add audit events, monitoring, backups, disaster recovery and managed secret/key rotation.
5. Add fraud/risk controls, transfer limits, robust double-entry ledger reconciliation and independent security review.
6. For an EU launch, obtain qualified legal/compliance advice on licensing/partner-bank model, PSD2/PSD3/PSR applicability, AML/KYC, GDPR, sanctions, SEPA/payment-provider integration and safeguarding.

Redis, NATS/Kafka and separate microservices should be introduced only when product and scale requirements justify them. For the current MVP, PostgreSQL transactions provide a simpler starting point.
