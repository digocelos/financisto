---
feature: credit-cards
date: 2026-10-27
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
approved: automatic 2026-10-27
topology:
  scope: fullstack
  structure: single-repo
  modules:
    backend:
      path: internal
      language: golang
    frontend:
      path: frontend
      framework: react
  api_pattern: none
---

# Research: Credit Cards

## Codebase Patterns

- Accounts are family-scoped and owned by a Member (`internal/db/migrations/0004_accounts.sql:23-36`). A card must follow the same family and Member ownership boundary, but should not be represented as a bank account.
- Financial activity is represented by Operations and Entries, with statuses for planned, confirmed and partially confirmed activity (`internal/db/migrations/0005_ledger.sql:1-29`). Card purchases and payments must integrate with this ledger rather than creating a parallel balance system.
- The ledger already supports transfers and audit events (`internal/ledger/handler.go:103-164`), which is the relevant pattern for a bank-account payment settling a card obligation without creating a second expense.
- The dashboard calculates current and projected balances from confirmed/planned entries (`internal/dashboard/handler.go:74-120`). Card committed limit and invoice totals need separate queries and must not inflate bank balances.
- The frontend is a React/Vite single-page app with direct API calls and custom CSS. The existing `Dashboard.jsx` and setup surface are the natural entry points for card navigation and forms.

## Technical Approach & Prior Art

- Model a Card as a Member-owned credit instrument with immutable purchase records, generated installments and invoices. Keep the purchase as the source record and installments as its schedule; an installment belongs to exactly one invoice.
- Use integer cents for principal, installment value, interest and totals. Do not recompute historical installment values from floating-point percentages after a purchase is confirmed.
- Invoice grouping is deterministic: a purchase processed on or before the closing day belongs to the current invoice; later purchases belong to the next invoice. Due date is derived from the configured due day and invoice month.
- Card payment is a settlement operation linking a bank debit entry to a card credit/settlement record. The payment must reduce the card obligation and bank balance while the purchase remains the only expense.
- Limit reporting should distinguish total limit, confirmed used limit, future committed limit and available limit. Cancelled purchases and settled invoices must not remain committed.

## Framework Constraints

- Go 1.27, pgx/v5 and PostgreSQL are the current backend stack; migrations are embedded and applied in filename order (`internal/db/migrations.go:15-60`).
- Existing operation APIs use UUIDs, ISO dates, integer cents, optimistic versions and family context headers. Card APIs should preserve those conventions.
- React has no router or form library. The first UI can use local state and a dedicated card workspace component, but domain calculations belong on the server and should be covered by Go tests.

## Key Findings

1. Cards are not bank accounts and must not reuse account opening balances or appear as ordinary bank accounts.
2. Purchase, installment, invoice and payment are separate concepts in the glossary and need separate ownership/lifecycle data.
3. A card payment must be modeled as settlement/transfer behavior, not a new expense, to avoid double counting.
4. Installment adjustments require preserving the original purchase and an audit trail of the changed schedule.
5. Limit availability depends on both current and future installments, so it cannot be derived from the current invoice alone.
6. Closing and due day calculations need date-boundary tests, including months shorter than the configured day and year transitions.

## Risks & Unknowns

- The product has not yet defined whether interest is entered as a total financed amount, a rate, or both; the first plan should support explicit principal, quantity and installment value and calculate the implied total interest.
- The exact operation types for card purchase, invoice settlement and payment are not present in the current global catalog.
- Existing ledger balance rebuilding only knows bank accounts; payment settlement must extend it without corrupting historical snapshots.
- A card can be configured without a linked bank account; payment selection should be optional until settlement.
