# Profile settings implementation plan

Target branch: `feat/profile-settings-api` (based on `dev-integration`). Merge to `dev-integration` only after checks; do not merge this work into `main`.

## Current API inventory
- `GET /v1/me` returns email, full name, country code, preferred currency and language.
- `PATCH /v1/me` updates full name, country code, preferred currency and language. It currently allows changes to country_code without KYC review: **do not expose this field as an unrestricted self-service edit**.
- `GET /v1/security/sessions` and `DELETE /v1/security/sessions/{id}` already exist.
- There are no dedicated endpoints for address, notification preferences, or app preferences.
- The Next.js dashboard is a demo with hardcoded Alex Carter and no persisted authenticated session. A real settings screen must authenticate before sending mutations; never substitute a demo identity.

## Proposed work
1. Define API contract for profile read/write and user preferences; validate against current Go handlers.
2. Add additive PostgreSQL migrations for user address and preferences (with sensible defaults, uniqueness and user ownership).
3. Add authenticated GET/PATCH endpoints, strict input validation, allowlists for language/currency/theme, user-scoped DB access, and tests for unauthorized access, invalid data and persistence.
4. Connect Next.js profile settings to the API with loading/error/success states; eliminate placeholder notifications and hardcoded account identity in authenticated views.
5. Protect regulated identity data: KYC-verified name, date of birth, nationality and country of residence require separate verification workflow. Never silently update verified identity.
6. Document native iOS/Android clients using the same API. Biometrics belong to native device-bound credential/session flow, not browser imitation.
7. Review Dockerfiles, Compose, migration startup and frontend API origin; change only if needed. Run Go tests, frontend lint/build, DB integration tests and security checks before PR merge.

## Confirmed KYC decision
- Legal name, date of birth, nationality, and country of residence changes require a new Veriff verification session and an approved result before any verified identity record is changed.
- Re-verification must be bound to the authenticated user and requested change; verify provider webhook signatures and decision state, prevent replay and duplicate applications, audit old/new values, and handle rejection without changing existing identity.
- Until that end-to-end workflow exists, generic `PATCH /v1/me` rejects attempts to change legal name or country with `403 kyc_required`; the API does not claim re-verification is already implemented.

## Open product questions
- Which currencies and languages are supported at launch? Defaults EUR/en.
- Which notification channels are implemented now? Persisting preferences does not by itself deliver push/email/SMS.

## Branch policy
- Release current tested `dev-integration` to `main` separately (PR #10 merged).
- Keep new profile-settings changes in feature branch, then PR into `dev-integration`.
- Never merge unfinished settings functionality into `main`.
