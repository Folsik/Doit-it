# Identity verification (KYC) — product and security design

## Onboarding placement

The single verification flow is mandatory after the user selects Visa/Mastercard and before the account is treated as fully opened / usable:

1. Phone verification
2. Set app PIN
3. Optional email
4. Choose card scheme
5. **Verify identity document + liveness**
6. Provider review / decision
7. Activate the prototype account and continue to home

There are no verification tiers. Every eligible user completes the same risk-based identity verification flow. The UX can show progress/status, but not “Level 1/2/3”.

## Recommended UX

- Explain why identity verification is required and how data is handled.
- Use a hosted SDK / secure capture flow from a regulated KYC provider. Ask for document issuing country and document type (national ID, passport, residence permit), then capture front/back where applicable.
- Capture a short liveness/selfie step and compare it to the document portrait.
- Show a neutral “Checking your documents” state while the provider processes.
- Outcomes: verified → continue; pending review → wait; retry required → show actionable capture guidance; rejected → stop onboarding and offer support/appeal.
- Never reveal detailed fraud rules or provider signals to the applicant.

## Anti-counterfeit controls

Use a specialist provider with document-authenticity checks, not custom image heuristics alone. Require provider support for:
- document template/security-feature validation and image tampering detection;
- MRZ/barcode consistency and checksum validation, OCR field consistency;
- document expiry and supported-document checks;
- selfie-to-document face match and active/passive liveness / presentation-attack detection;
- NFC chip verification when supported (especially e-passports / eID);
- risk signals for screenshots, replays, virtual cameras, injection and manipulated images;
- manual review and auditable decision reasons for borderline cases.

A successful OCR read is not proof that a document is genuine. Treat provider result as one input to the bank's KYC decision; maintain sanctions/PEP screening and AML controls separately.

## Duplicate-account / cross-document prevention

Do **not** deduplicate by document number alone: a national ID and passport have different numbers. After the provider verifies the person, use its stable verified-person subject identifier where contractually available, plus a keyed HMAC fingerprint derived from normalized, provider-verified identity attributes (for example full legal name, date of birth and issuing jurisdiction). Store only the fingerprint and provider references, not the source attributes or document number.

Enforce uniqueness in PostgreSQL so concurrent registrations cannot race into duplicate verified identities. A uniqueness conflict must place the case into a controlled duplicate-review/rejection path; do not disclose which existing account matched. Provide an appeal/manual-review process for legitimate cases (name changes, twins, provider false positives). Do not automatically merge accounts.

Provider subject IDs are provider-specific. For cross-provider migration, preserve a bank-controlled fingerprint version and secret management strategy; do not assume two vendors' IDs are comparable.

## API / integration contract

The backend should expose authenticated endpoints to:
- create or resume a provider verification session for the signed-in user;
- retrieve the user's current verification status;
- receive provider webhooks with signature verification, timestamp/replay checks and idempotent event IDs.

The provider's hosted SDK handles document and selfie capture. The Go backend must never trust a client-supplied “verified” flag. Only a verified, authenticated provider webhook may transition a session to verified. Validate that the provider session belongs to the user and that required checks passed.

Card/account activation must be gated on verified identity. Do not allow the current card-selection endpoint to mark the account fully opened before this step.

## Data protection / compliance

- Prefer provider-hosted capture; do not proxy/store raw images unless a documented legal and security basis requires it.
- Encrypt provider references and sensitive metadata at rest; keep access tightly scoped and audited.
- Define retention/deletion periods with legal counsel and provider contract; keep only required evidence/audit trail.
- Obtain explicit notices/consents as required, support data-subject rights, and assess GDPR/DPIA obligations before EU production.
- KYC is not a substitute for sanctions screening, PEP/adverse-media checks, AML monitoring, age/residency eligibility or issuer approval.

## Implementation status

This change adds the persistence model and integration/security requirements. A production-ready provider adapter and signed webhook must be implemented after selecting a KYC vendor and obtaining sandbox credentials. Until then, identity verification must fail closed and the app must not represent accounts as fully opened.
