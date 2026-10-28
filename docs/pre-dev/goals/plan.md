<!-- approved: automatic 2026-10-28 -->

# Implementation Plan: Goals

**Target:** `ticket/11-goals`

## Phase 1: Investments and vaults

1. Add investment account, movement, vault and vault movement migrations.
2. Add family-scoped APIs and deterministic balance validation.
3. Add unit tests for deposits, withdrawals, returns and over-allocation.
4. Add frontend forms and history views.

## Phase 2: Goals and progress

1. Add goals, plans, vault links and contribution events.
2. Implement recomputable progress and reversible completion.
3. Add APIs, tests and progress UI.

## Phase 3: Purchase integration

1. Link eligible purchases to goals.
2. Add unique principal contribution records and reversal behavior.
3. Verify principal-only progress and no double counting after settlement.

## Verification

- `go test ./...`
- Frontend tests/build
- Migration against existing database
- Family isolation and idempotency tests
