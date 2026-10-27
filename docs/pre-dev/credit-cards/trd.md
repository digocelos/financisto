<!-- approved: automatic 2026-10-27 -->

---
feature: credit-cards
deployment:
  model: local-and-cloud-compatible
tech_stack:
  primary: Go + React
  standards_loaded: [existing-repository-patterns]
project_technologies:
  - category: backend
    prd_requirement: Card and invoice lifecycle
    choice: Existing Go HTTP service with relational transactions
    rationale: Matches current repository
  - category: frontend
    prd_requirement: Card workspace
    choice: React/Vite with existing CSS
    rationale: Matches current product surface
---

# Technical Requirements: Credit Cards

## Assumptions

- ⚠️ UX risk: no standalone design validation ran; the card workspace should follow the existing dashboard/setup language.
- Card purchases are recorded as expense activity only when the purchase is confirmed; invoice payment is a settlement transfer and not another expense.
- The first version uses BRL and calendar-day closing/due configuration.

## Architecture

Extend the modular monolith with a `cards` domain. The domain owns Cards, Purchases, Parcels, Faturas, payment links and limit calculations. It calls existing auth, accounts and ledger boundaries through service functions and commits cross-domain settlement in one transaction.

### Bounded contexts

| Context | New / Existing | Owns | Talks to |
|---|---|---|---|
| `cards` | New | cards, purchases, installments, invoices, card payments | auth, accounts, ledger |
| `accounts` | Existing | bank accounts and members | cards payment boundary |
| `ledger` | Existing | operations, entries, balances and audit events | cards purchase/payment boundary |

## Program Design

```text
internal/cards/
├── handler.go          # card, purchase, invoice and payment HTTP routes
├── service.go          # lifecycle commands and transaction coordination
├── billing.go          # closing/due date and invoice assignment rules
├── limits.go           # current/future commitment calculations
├── models.go           # card, purchase, installment, invoice types
└── *_test.go
```

Handlers validate input and family scope; services own commands; billing and limits remain deterministic and unit-testable; persistence stays behind the service transaction boundary.

## Data Flow

1. User creates a Member-owned Card with limit and cycle days.
2. Purchase command validates Card ownership, principal, quantity and dates.
3. Service computes financed total, interest and installment dates, assigns each Parcel to one Fatura and persists the source Purchase plus schedule.
4. Invoice query aggregates installments and payment state.
5. Payment command validates bank account access, creates a settlement operation and links it to the Fatura in one transaction.
6. Limit query derives total, current used, future committed and available values from active installments and settled payments.

## Failure Modes

| Failure | Behavior |
|---|---|
| Invalid limit or cycle day | Reject request with field errors |
| Card archived or outside family | Return not found/forbidden; expose no data |
| Invalid installment totals | Reject unless an explicit adjustment is supplied |
| Date at closing boundary | Apply deterministic inclusive closing rule and test it |
| Concurrent purchase/payment update | Use version/transaction conflict; do not overwrite |
| Bank payment transaction failure | Roll back bank debit, settlement link and limit changes |
| Repeated payment request | Idempotency key or unique invoice payment constraint returns existing result |

## API Patterns

- `GET/POST/PATCH /v1/families/{familyID}/cards`
- `GET/POST/PATCH /v1/families/{familyID}/cards/{cardID}/purchases`
- `GET /v1/families/{familyID}/cards/{cardID}/invoices`
- `GET /v1/families/{familyID}/cards/{cardID}/invoices/{invoiceID}`
- `POST /v1/families/{familyID}/cards/{cardID}/invoices/{invoiceID}/payments`
- All routes require authenticated family membership and use existing problem errors, UUIDs, ISO dates and integer cents.

## Security and Quality

Card data is family-scoped and Member-owned. Imported descriptions and user labels are untrusted text. Payment commands must validate both Card and bank account ownership server-side. Limit and invoice queries should remain bounded to the selected Card and use indexed family/card/date access paths.

## Dependencies

| Category | Choice | Version | Rationale |
|---|---|---|---|
| Backend | Go | 1.27 | Existing repository |
| Database | PostgreSQL | 16 local | Existing relational transaction model |
| Driver | pgx/v5 | 5.8.0 | Existing dependency |
| Frontend | React/Vite | Existing lockfile | Existing UI stack |
