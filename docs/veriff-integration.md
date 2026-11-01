# Veriff Go integration

This package provides the first server-side integration layer:

- Creates hosted verification sessions through Veriff Public API v1.
- Correlates the session to the authenticated Doit user via `vendorData`.
- Validates Veriff webhook HMAC-SHA256 signatures over the exact raw request body.
- Parses decision status after signature validation.

## Configuration

Set these values from Veriff Customer Portal (never commit credentials):

- `VERIFF_BASE_URL`: exact integration Base URL from the portal (HTTPS; do not guess production/sandbox host).
- `VERIFF_API_KEY`: integration API key sent as `X-AUTH-CLIENT`.
- `VERIFF_SHARED_SECRET`: shared secret used for webhook HMAC.
- `VERIFF_CALLBACK_URL`: HTTPS callback/return URL configured for your integration.

The client app should open the returned hosted `verification.url`. Do not expose the API key or shared secret to the client.

## Important integration boundary

This package is not yet wired into the application router or database lifecycle. The next step is to connect authenticated endpoints to persist the session ID, expose status, process signed webhooks transactionally/idempotently, derive the provider-verified identity fingerprint, and enforce KYC gating on card/account activation. Do not treat a client redirect or client-submitted status as proof of verification.

Veriff decision statuses include `approved`, `declined`, `resubmission_requested`, `expired`, and `abandoned`. Map them to internal states only after validating the webhook and correlating the stored session to the user. A pending/manual-review outcome should remain non-verified.

## Official docs

- [Create session](https://devdocs.veriff.com/v1-api/apidocs/v1sessions)
- [Webhook guide](https://devdocs.veriff.com/docs/webhooks-guide)
- [Public API guide](https://devdocs.veriff.com/apidocs/veriff-public-api-guides)
