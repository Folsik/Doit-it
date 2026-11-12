# Doit Web

Responsive Next.js + TypeScript prototype for the Doit personal-finance app.

## Run locally

Requires Node.js 20.9 or newer.

    cd web
    npm install
    npm run dev

Open http://localhost:3000.

## Screens

- Home dashboard with balance overview, account cards, chart, quick actions, budget and recent activity.
- Cards & accounts with masked account details and prototype card controls.
- Transactions with search and category filters.
- Payments with send-money modal and payment actions.
- Insights with category breakdown and spending summary.
- Profile with locale and preferences.
- Security with six-digit PIN sign-in guidance and explicit unavailable states for PIN change, PIN recovery, and device biometrics. No authenticator-app TOTP.

## Phone onboarding and PIN sign-in

Open `http://localhost:3000/auth` to create an account or sign in with a six-digit app PIN. Set `NEXT_PUBLIC_API_URL` to your Go API origin (defaults to `http://localhost:8080`). The flow uses `/v1/onboarding/countries`, `/v1/onboarding/send-otp`, `/v1/onboarding/verify-otp`, `/v1/onboarding/set-pin`, `PUT /v1/me/email`, `/v1/cards/choose`, and `/v1/auth/phone-login`.

**Development only:** The backend returns a `development_code` for its simulated SMS OTP if not running in production; no real SMS delivery is configured. The browser never stores the bearer token in localStorage, and navigation to `/` leaves the authenticated session behind. The dashboard is a separate demo view, not live user data. PIN recovery/change and true device-bound biometric sign-in require backend support and native platform integration or WebAuthn device credentials; they are not active on this web prototype.

## Prototype boundaries

All balances, transactions, user details and card details are illustrative demo data. Sending a transfer only adds a local demo transaction in the browser; it does not call the Go API or move money. Security controls not supported by the Go backend are explicitly shown as unavailable rather than pretending to work.

No JWT is stored in localStorage. The frontend is not authenticated yet and must not be used with real customer data.

## Next integration step

Implement a Next.js BFF layer with server-side session handling and HttpOnly, Secure, SameSite cookies, then proxy requests to the Go REST API. Keep access and refresh tokens server-side; do not expose them to browser JavaScript. Add CSRF protections and strict cookie/CORS policy before connecting authenticated workflows.

The current Go API accepts Bearer tokens and does not yet issue refresh tokens or set cookies. The BFF needs an explicit backend auth contract before production integration.
