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
- Security settings prototype (2FA and biometrics are visual placeholders; no authenticator setup or PIN-change API exists yet).
- `/auth` phone onboarding and sign-in flow connected to the existing Go onboarding endpoints.

## Prototype boundaries

Dashboard balances, transactions, user details and card details remain illustrative demo data. Sending a transfer only adds a local demo transaction in the browser; it does not call the Go API or move money. The `/auth` route does call the Go API for phone OTP, PIN creation/login, optional email and card preference. Development OTP is displayed only when the API returns `development_code`; real SMS delivery is not configured.

No JWT is stored in localStorage. The onboarding route keeps the short-lived access token in component memory only. Refreshing the page loses that in-memory session. The dashboard itself is still not authenticated and must not be used with real customer data.

## Next integration step

Implement a Next.js BFF layer with server-side session handling and HttpOnly, Secure, SameSite cookies, then proxy requests to the Go REST API. Keep access and refresh tokens server-side; do not expose them to browser JavaScript. Add CSRF protections and strict cookie/CORS policy before connecting authenticated workflows.

The current Go API accepts Bearer tokens and does not yet issue refresh tokens or set cookies. The BFF needs an explicit backend auth contract before production integration.
