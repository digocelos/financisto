<!-- approved: automatic 2026-10-27 -->

# Product Requirements: Credit Cards

## Problem

The system currently tracks bank-account activity but cannot represent a Member's credit card obligation, invoice cycle or installment schedule. Users therefore cannot see how much of a card limit is committed, what is due in future invoices, or record a card payment without risking duplicate expenses.

## Executive Summary

Users can create Member-owned Cards with limit and invoice-cycle settings, record one-time or installment Purchases, and inspect current and future Faturas. The system calculates principal, financed total, interest and committed limit while preserving the original Purchase when installments are adjusted. Payments from a Conta bancária settle the Card obligation and restore available limit without creating a second expense.

## Functional Requirements

### FR1 — Configure Cards

- A user can create, view, edit and archive a Card owned by a Member.
- A Card requires credit limit, closing day and due day.
- Acceptance: invalid days and non-positive limits are rejected; archived Cards cannot receive new Purchases; cards are isolated by Família.

### FR2 — Record Purchases

- A user can record a one-time or installment Purchase on a Card with date, description, principal, quantity and installment value.
- Acceptance: one-time purchases create one installment; installment purchases create the requested schedule; the original Purchase remains linked to every installment.

### FR3 — Calculate financing

- The system displays principal, financed total and total interest.
- Acceptance: values use integer cents; installment value × quantity equals financed total unless an explicit adjustment is recorded; adjustments preserve the original values and are auditable.

### FR4 — Generate and group Faturas

- Each installment belongs to one Fatura according to the Card's closing day and purchase/processing date.
- Acceptance: purchases before or on closing belong to the applicable invoice; later purchases belong to the next invoice; due dates respect the configured due day and short months.

### FR5 — Report limits

- The Card view shows total limit, current used limit, future committed limit and available limit.
- Acceptance: confirmed and future installments are included in the appropriate totals; cancelled or settled obligations reduce commitment; available limit never becomes negative without an explicit over-limit state.

### FR6 — Settle payments

- A user can pay a Fatura from a selected bank account.
- Acceptance: the bank account records the debit, the Card obligation records settlement, available limit is restored, and no second expense is created for the payment.

### FR7 — Manual operation and audit

- Users can manually enter Purchases when no statement export exists and can inspect the source Purchase, installments, Fatura and payments.
- Acceptance: all changes are family-scoped, authenticated and auditable; optimistic version conflicts do not silently overwrite changes.

## Scope

### In scope

Card configuration, manual purchases, installments, interest calculation, invoice grouping, limits, future commitments, bank-account payments and audit history.

### Out of scope

Automatic card statement import, bank connectivity, recurring card subscriptions, rewards/points, foreign-currency conversion, revolving credit interest after the first invoice, and automatic categorization.

## Validation

The requirements cover the issue's acceptance criteria, preserve the glossary distinction between Card, Fatura, Parcela, Operação financeira and Lançamento, and keep Card obligations separate from bank-account balances until a payment is explicitly recorded.
