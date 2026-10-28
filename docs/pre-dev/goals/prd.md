<!-- approved: automatic 2026-10-28 -->

# Product Requirements: Goals

## Problem

Users cannot separate money inside an account for a purpose, track investments independently, or measure progress toward a financial target. Without explicit Goals, Vaults and contribution history, balances do not explain whether money is available for a planned objective.

## Requirements

### FR1 — Investment accounts

Users can create Member-owned investment accounts and record contributions, withdrawals, dividends/JCP and returns.

Acceptance: account ownership is family-scoped; every movement records date, amount and type; returns are not confused with principal.

### FR2 — Vaults

Users can create Vaults allocated within real accounts and deposit or withdraw funds.

Acceptance: a vault cannot allocate more than its real-account available balance; movements preserve account and vault history; moving funds does not create money.

### FR3 — Goals

Users can create Goals with target, deadline, priority and contribution plan.

Acceptance: invalid targets/deadlines are rejected; plan changes are historical; completion can be reversed or extended explicitly.

### FR4 — Goal relationships

A Goal can link to multiple Vaults and eligible financed purchases.

Acceptance: links are family-scoped; paid principal contributes to progress while interest does not; one purchase principal is counted once.

### FR5 — Progress

The system displays progress from linked vault balances, direct contributions and paid purchase principal, less withdrawals and adjustments.

Acceptance: progress is recomputable; vault movements used for settlement do not double-count; contribution history explains every change.

## Scope

In scope: investment accounts, returns, vaults, goals, many-to-many vault links, purchase links, contribution plans, progress and reversible completion.

Out of scope: brokerage connectivity, automatic valuation/pricing, tax calculation, multi-currency investment valuation and automatic goal recommendations.
