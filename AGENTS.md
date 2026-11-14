# Repository development instructions

This repository adopts the development workflow from [Superpowers](https://github.com/obra/superpowers) (MIT), adapted for this Go fintech backend.

## Required workflow

1. **Understand before changing**: inspect the relevant code, tests, configuration, migrations, and current branch state. Identify constraints and ask before making assumptions that affect product behavior, money movement, compliance, or data migrations.
2. **Brainstorm/design for non-trivial work**: state the problem, proposed design, alternatives, edge cases, and acceptance criteria. Get user approval before implementing a new feature or materially changing architecture.
3. **Plan**: break work into small, verifiable tasks. Identify files, dependencies, tests, migration/rollback implications, and security concerns.
4. **Use a focused branch/worktree**: keep each feature or fix isolated; do not merge or push to `dev-integration` without explicit approval.
5. **TDD**: write a failing test for the behavior/bug first; run it to confirm failure; implement the smallest change; run tests again; refactor only while tests stay green.
6. **Debug systematically**: reproduce, inspect evidence, identify the root cause, formulate a hypothesis, test it, then fix. Avoid speculative multi-change patches.
7. **Verify before claiming completion**: run relevant tests, `go test ./...`, `go vet ./...`, and applicable lint/build/migration checks. Report exactly what ran and what did not. Never claim tests passed if they were not executed.
8. **Review**: inspect the diff for correctness, regressions, security, observability, maintainability, and scope creep. Address review findings before calling work complete.
9. **Finish safely**: summarize changes, verification, risks, and next steps. Present merge/PR options; do not merge without explicit authorization.

## Fintech-specific invariants

- Treat balances and transfers as ledger/accounting operations, not ordinary CRUD.
- Preserve idempotency, transactional consistency, auditability, and immutable financial records.
- Never log secrets, authentication tokens, full payment-card data, identity-document contents, or unnecessary personal data.
- Validate authorization at every operation boundary; assume client input and provider webhooks are untrusted.
- Verify webhook signatures and handle replay/idempotency before processing provider events.
- Use integer minor units or an exact decimal representation for money; never floating-point arithmetic.
- Keep provider integrations behind interfaces and do not enable real-money movement without explicit product, compliance, and operational approval.
- Migrations must be reviewed for locking, backfill, rollback, and compatibility with deployed code.
- Do not weaken KYC/AML, sanctions, fraud, or country eligibility controls to make tests pass.

## Go conventions

- Prefer small packages with clear ownership and explicit dependencies.
- Use context-aware I/O, wrapped errors, and structured logging.
- Avoid global mutable state and unnecessary abstractions.
- Test business rules at the package level and critical flows with integration tests.
- Keep generated code generated; do not hand-edit generated protobuf files.

## Scope and communication

- Make the smallest change that satisfies the approved task.
- Do not silently alter unrelated files or product requirements.
- Explain assumptions and unresolved questions.
- A task is not complete until verification results and remaining limitations are reported.
