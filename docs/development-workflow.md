# Superpowers workflow adoption

This project uses a repository-level adaptation of [obra/superpowers](https://github.com/obra/superpowers), a skills-based software-development workflow.

## What is adopted

- Brainstorming and design approval before substantial implementation
- Small implementation plans with explicit acceptance criteria
- Test-driven development (red → green → refactor)
- Systematic root-cause debugging
- Independent code review before completion
- Verification-before-completion and explicit reporting
- Safe branch/PR finishing without implicit merges

The repository-specific rules are in the root `AGENTS.md`.

## How to use it

For each substantial task:

1. Read `AGENTS.md` and inspect the current codebase.
2. Describe the design and get approval where behavior/architecture is involved.
3. Create a focused branch and a task plan.
4. Implement through failing tests first.
5. Run targeted tests, then broader checks as appropriate.
6. Review the diff and report test evidence, limitations, and risks.
7. Open a PR for review; merge only when explicitly authorized.

## Installation note

The upstream project is a collection of agent skills, not a Go dependency or runtime library. This commit installs a lightweight, repository-level adaptation so the workflow is available to contributors and coding agents working in this repo. It does **not** install the upstream skills into a local editor/agent runtime or modify the user's laptop. For runtime-specific installation, follow the upstream instructions: https://github.com/obra/superpowers

## Fintech adaptations

The workflow adds explicit safeguards for ledger correctness, idempotency, webhook verification, sensitive data, KYC/sanctions controls, and safe database migrations. These are project-specific adaptations rather than claims about upstream defaults.
