# Doit Web

Next.js 15 + React 19 + **Tailwind CSS v4 and shadcn/ui**.

The dashboard, onboarding, profile, payments, insights, cards, security, and transaction screens share the same Tailwind design tokens and reusable components in `components/ui/`. The shadcn-style source components live in the repo (not a prepackaged UI runtime). Radix primitives handle accessible dialogs, switches and labels.

## Local development

```bash
cd web
npm install
npm run dev
```

Use `docker compose up -d --build` from the repository root to start Next.js, Go API, and PostgreSQL together. The browser accesses the Go API through the same-origin `/api/v1/*` rewrite. Profile operations go through the authenticated `/api/banking/v1/*` Next.js route handler.

## Design system

- `app/globals.css`: Tailwind v4 imports, design tokens, light/dark themes and minimal base styles.
- `components/ui/`: Button, Card, Input, Label, Select, Badge, Dialog, Separator, Switch.
- `lib/utils.ts`: class merge helper.
- `components.json`: shadcn CLI configuration.

To add more shadcn components:

```bash
npx shadcn@latest add <component>
```

Design components live as local source code, so changes should be reviewed and tested like application code.

## Known prototype boundaries

The profile screen reads and saves actual authenticated user settings through Go API. Legal identity and residence country changes require Veriff re-verification, which is currently an unconfigured stub. EUR is the only supported account currency.

**Home, Accounts, Transactions, and monthly transfer insights are read from the Go REST API through an authenticated Next.js BFF session.** Balances are stored as integer minor units, projected in `accounts.balance_minor` and updated atomically with posted ledger entries. Monthly statistics are completed internal-transfer volumes, not spending categories. A newly created account displays EUR 0.00 and an empty transfer list; there are no injected demo funds or fake transactions.\n\nReal card issuance, historical balance charts, spending categorization, external settlement providers, token refresh, production SMS, and full i18n outside profile remain unimplemented. The Payments page does not initiate transfers until its recipient and confirmation flow is wired to the backend safely.

## Checks

```bash
npm run lint
npm run typecheck
npm run build
```

GitHub Actions also builds the Next.js container and tests the onboarding, authentication, profile and API with a PostgreSQL-backed development fixture.
