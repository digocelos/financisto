---
feature: goals
date: 2026-10-28
research_mode: modification
mode: automatic
auth: user
license: no
ui: yes
ui_library: React
styling: custom CSS
steps:
  feature-map: run
  api-contract: run
  data-model: run
approved: automatic 2026-10-28
topology:
  scope: fullstack
  structure: single-repo
  modules:
    backend: { path: internal, language: golang }
    frontend: { path: frontend, framework: react }
  api_pattern: none
---

# Research: Goals

## Codebase Patterns

- Members and families are the ownership boundary for all financial resources (`internal/db/migrations/0002_identity.sql`).
- Accounts and opening balances are already modeled (`internal/db/migrations/0004_accounts.sql`); investment accounts should be a distinct account type while vaults remain allocations inside real accounts.
- Operations and entries are the ledger source of truth (`internal/db/migrations/0005_ledger.sql`). Goal progress must consume confirmed entries and preserve audit history rather than maintain an opaque counter.
- Planning already contains groups, categories and recurring entries (`internal/db/migrations/0006_planning.sql`, `internal/planning/handler.go`). Goals should integrate with planning without making categories own goal progress.
- The React frontend uses direct fetches and dashboard/setup surfaces; goals need a dedicated section with clear progress and contribution history.

## Technical Approach

- Use separate entities for investment accounts, vaults, goals, goal-vault links and goal contributions.
- Record every deposit, withdrawal, return, allocation and goal contribution as an auditable financial event. Derive progress from these events plus paid purchase principal.
- Store target/deadline/priority and contribution-plan parameters on the goal; preserve historical plan changes.
- Make vault movements atomic and idempotent so moving funds to settle a purchase cannot count both the withdrawal and the purchase as progress.

## Key Findings

1. Principal and interest must remain separate when a financed purchase contributes to a goal.
2. A vault is not a bank account; its balance is an allocation constrained by its real account.
3. Multiple vaults can contribute to one goal, requiring a many-to-many relationship.
4. Completion must be a reversible state transition with history, not a destructive flag.
5. Progress should be recomputable from immutable contribution events.

## Risks & Unknowns

- Investment-account product types and return categories need a bounded MVP vocabulary.
- Goal progress semantics for withdrawals from a linked vault need explicit event ordering.
- Purchase-goal links must prevent the same paid principal from contributing twice.
