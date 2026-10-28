<!-- approved: automatic 2026-10-27 -->

# Implementation Plan: Credit Cards

**Target:** `ticket/10-cards`

## Phase 1: Cards and purchases

1. Add card, purchase and installment domain tables and migration.
2. Add family-scoped card CRUD endpoints with Member ownership validation.
3. Add purchase creation/listing with deterministic financing and installment calculations.
4. Add unit tests for money calculations, installment remainder handling and validation.
5. Add the card and purchase UI to the authenticated workspace.

## Phase 2: Invoices and limits

1. Add invoice assignment by closing day and due-date calculation for short months/year boundaries.
2. Add invoice and limit query endpoints.
3. Add audit/version handling for purchase and installment adjustments.
4. Add invoice/limit UI and tests for current/future commitment.

## Phase 3: Settlement

1. Add idempotent invoice payment command linked to a bank account and ledger operation.
2. Ensure payment reduces bank balance and card obligation without a second expense.
3. Add settlement audit and concurrency tests.
4. Add payment UI and end-to-end API coverage.

## Verification Gates

- `go test ./...`
- Frontend unit/build checks
- Migration applies to an existing database without rewriting ledger history.
- Family isolation tests cover every card, purchase, invoice and payment route.
- A payment scenario proves one purchase expense and one bank debit, not two expenses.
