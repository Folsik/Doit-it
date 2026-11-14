# Development test account and environment isolation

The demo dashboard must not be treated as an authenticated banking account. The existing `/` page currently uses mock data; `/auth` provides real prototype registration. Removing the mock UI and wiring an authenticated session is a separate release blocker.

## Create an actual development test user

Start the local development stack:

```sh
docker compose up -d --build
DEV_TEST_EMAIL=dev-tester@example.test DEV_TEST_PASSWORD='replace-with-a-unique-strong-secret' bash scripts/seed-dev-test-user.sh
```

The script calls the existing registration API, creating a real user and zero-balance EUR account in the **local development PostgreSQL database**. It is idempotent for an existing email. It never prints the password or the issued access token. Choose a unique secret and do not commit it.

The script only accepts `localhost:8080` or `127.0.0.1:8080` and refuses `APP_ENV` other than `development`. It is **not** an automatic database migration or a production seed. It does not configure a phone/PIN login; the current `/auth` phone login cannot authenticate this email/password fixture. A proper unified login screen is still required.

## Future dev/prod split

- Development and production must have separate PostgreSQL instances, credentials, networks, secrets, storage volumes, and backup policies; never promote a development database dump to production.
- Production must never execute the development seeding script or import development fixture data. Keep production credentials out of development CI.
- Apply the same versioned schema migrations independently in each environment; migrate schema, not dev users.
- Explicitly distinguish fixture accounts in development. Do not rely on email names or UI labels as the only security boundary.
- Veriff remains an unimplemented stub for legal identity updates: no mock approval may mutate verified data.
- Before release: replace hardcoded demo dashboard data with authenticated API responses and test onboarding-to-dashboard end to end.
