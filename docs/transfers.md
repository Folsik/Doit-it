# Transfers — product and backend design

## Scope

This feature is a transfer orchestration layer. The existing internal transfer endpoint remains the only rail that can move prototype ledger balances. External rails are provider-backed instructions and must not be represented as completed until a provider confirms settlement.

| Rail | UX estimate | MVP behavior |
|---|---:|---|
| In-app P2P | seconds, up to ~20s | Atomic internal ledger transfer; EUR; free; idempotent |
| SEPA | up to 2 business days | Create external payment instruction via licensed provider |
| SEPA Instant | seconds | Provider capability/eligibility check; pending until confirmed |
| Local / international bank | provider-dependent | Country + currency + beneficiary fields; quote/fees before submit |
| SWIFT | ~3–5 business days | Provider route; fees, FX quote, intermediary details disclosed |
| Card transfer | seconds to 30 minutes | Provider-supported push-to-card only; never collect/store PAN in our API |
| Payment link / bill split | link expiry / request flow | Payment request; payer authenticates and initiates a transfer |

These timing ranges are indicative UX copy, not a guarantee. Availability depends on country, currency, provider, bank, cut-off, compliance checks and recipient institution.

## Safety and accounting invariants

- All money values are integer minor units plus an ISO-4217 currency; never use floating point.
- Internal P2P is free, same-currency, and atomic. Debit and credit ledger entries must balance to zero.
- Use a client-generated idempotency key scoped to the authenticated user and persist a request fingerprint. Same key + same payload returns the original result; same key + different payload is a conflict.
- Lock source/destination accounts in deterministic UUID order; validate ownership, active status, currency, sufficient funds and limits inside one DB transaction.
- External transfers are separate from internal ledger postings. Persist a payment instruction, provider reference, immutable quote/fee snapshot and lifecycle events. Do not debit a spendable balance twice on retries.
- External state machine: created → pending_submission → submitted → pending/processing → completed; terminal alternatives: rejected, failed, cancelled, returned. Only verified provider callbacks or reconciliation may mark settlement.
- Webhooks require signature verification, timestamp/replay protection and provider-event idempotency. Unknown/duplicate events are safe no-ops.
- Never log full account/card credentials, beneficiary sensitive fields, access tokens, webhook secrets or provider payloads.
- Require step-up authentication, risk checks, limits and sanctions/AML screening before external submission. KYC must be verified before any account is activated for payments.
- Card PAN/CVV must be entered only in PCI-compliant provider-hosted/tokenized components; backend stores provider token/reference only.
- Payment links are random, high-entropy, expiring, revocable, single-use or explicitly multi-use, and reveal no payer/recipient private data.

## Suggested API

Authenticated:
- POST /v1/transfers/internal — P2P transfer; idempotency key required.
- POST /v1/payment-instructions/quote — route, fees, FX and estimated delivery.
- POST /v1/payment-instructions — create SEPA/SEPA Instant/local/international/SWIFT/card instruction from a valid quote.
- GET /v1/payment-instructions — own history, paginated.
- GET /v1/payment-instructions/{id} — own instruction/status.
- POST /v1/payment-links — create request link.
- GET /v1/payment-links/{token} — public, minimal request details.
- POST /v1/payment-links/{token}/pay — authenticated payer initiates payment.
- POST /v1/bill-splits — create split request; POST /v1/bill-splits/{id}/participants for requests.

Provider callbacks:
- POST /v1/webhooks/payments/{provider} — signature-verified, idempotent event ingestion.

## Data model direction

Keep internal transfers/ledger as-is. Add external payment instructions, quote snapshots, provider events, beneficiary references and payment links as separate tables. Use provider-neutral rail/status enums with provider-specific details stored in encrypted JSON only when necessary. Persist currency, amount, fee, FX rate, recipient snapshot, quote expiry, idempotency key, provider reference and timestamps.

## Delivery plan

1. Harden/refactor internal P2P endpoint and tests (atomic ledger, idempotency, account ownership/status/currency).
2. Add payment-instruction persistence and API contract.
3. Integrate one licensed EU payment provider for SEPA/SEPA Instant first; implement signed webhooks + reconciliation.
4. Add local/international/SWIFT and card rails only where the provider contract and compliance scope support them.
5. Add payment links and bill splitting on top of the same transfer orchestration API.

## Current implementation status

This repository currently has a prototype internal EUR ledger transfer. No external payment provider is connected. Creating an instruction is not equivalent to sending real money. Production launch requires a licensed provider/partner, KYC/AML/sanctions controls, operational reconciliation, limits, dispute/returns handling and compliance approval.
