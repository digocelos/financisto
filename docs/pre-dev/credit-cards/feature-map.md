<!-- approved: automatic 2026-10-27 -->

# Feature Map: Credit Cards

## Feature Categories

| Feature | Category | Domain | Priority | Dependencies |
|---|---|---|---|---|
| Card configuration | Core | Card management | P0 | Member and family |
| Purchase capture | Core | Card activity | P0 | Card configuration |
| Installment schedule | Core | Card activity | P0 | Purchase capture |
| Financing calculation | Core | Card activity | P0 | Purchase capture |
| Invoice grouping | Core | Billing cycle | P0 | Card dates and installments |
| Limit and commitment view | Core | Card management | P0 | Installments and settlement |
| Bank-account payment | Core | Settlement | P0 | Bank account, invoice |
| Manual adjustment and audit | Supporting | Card activity | P1 | Purchase and schedule |

## Domain Map

### Card Management

Features: Card configuration, limit and commitment view.

Interactions: → Card Activity; → Billing Cycle; ← Settlement.

### Card Activity

Features: Purchase capture, installment schedule, financing calculation, manual adjustment and audit.

Interactions: → Billing Cycle; → Card Management; → Settlement.

### Billing Cycle

Features: Invoice grouping.

Interactions: ← Card Activity; → Settlement; → Card Management.

### Settlement

Features: Bank-account payment.

Interactions: ← Billing Cycle; ↔ Existing bank-account ledger.

## User Journeys

### Journey: Configure and use a Card

Select Member → Create Card → Set limit/closing/due days → Record Purchase → Review limit and invoice.

### Journey: Installment Purchase

Record principal and quantity → Review financed total and interest → Confirm schedule → Inspect installments across future Faturas.

### Journey: Pay a Fatura

Open Fatura → Select Conta bancária → Confirm payment → See bank debit, settled Card obligation and restored limit without duplicate expense.

### Journey: Adjust a Purchase

Open Purchase → Adjust installment schedule → Review difference → Confirm → Preserve original values and audit event.

## Integration Points

| Feature | Integrates With | Direction | Purpose |
|---|---|---|---|
| Card configuration | Members and family context | IN | Scope ownership and access |
| Purchase capture | Existing ledger | OUT | Record card activity and accounting effects |
| Invoice grouping | Card dates | IN | Assign each installment to one Fatura |
| Bank-account payment | Bank-account ledger | IN/OUT | Settle obligation and debit the selected account |
| Limit view | Installments and settlements | IN | Calculate current/future commitments |

## Phases

### Phase 1: Cards and purchases

**Milestone:** A user creates a Card and records one-time or installment Purchases with calculated totals and linked Parcels.

**Features:** Card configuration; Purchase capture; Installment schedule; Financing calculation.

**Relationships:** Card Management → Card Activity.

**Integration Points:** Card Activity ← Members and family context.

### Phase 2: Invoices and limits

**Milestone:** Purchases are grouped into Faturas and the Card view reports current/future commitment and available limit.

**Features:** Invoice grouping; Limit and commitment view; Manual adjustment and audit.

**Relationships:** Card Activity → Billing Cycle → Card Management.

**Integration Points:** Invoice grouping ← Card dates; Limit view ← Installments.

### Phase 3: Settlement

**Milestone:** A user pays a Fatura from a Conta bancária without duplicating the expense, and the Card limit is restored.

**Features:** Bank-account payment.

**Relationships:** Billing Cycle → Settlement ↔ Existing bank-account ledger.

**Integration Points:** Settlement ↔ Existing bank-account ledger.
