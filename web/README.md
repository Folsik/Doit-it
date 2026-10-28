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
- Security with 2FA, biometrics and active-session UI.

## Prototype boundaries

All balances, transactions, user details and card details are illustrative demo data. Sending a transfer only adds a local demo transaction in the browser; it does not call the Go API or move money. Settings controls are visual placeholders.

No JWT is stored in localStorage. The frontend is not authenticated yet and must not be used with real customer data.

## Next integration step

Implement a Next.js BFF layer with server-side session handling and HttpOnly, Secure, SameSite cookies, then proxy requests to the Go REST API. Keep access and refresh tokens server-side; do not expose them to browser JavaScript. Add CSRF protections and strict cookie/CORS policy before connecting authenticated workflows.

The current Go API accepts Bearer tokens and does not yet issue refresh tokens or set cookies. The BFF needs an explicit backend auth contract before production integration.
