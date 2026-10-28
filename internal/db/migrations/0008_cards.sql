-- approved: automatic 2026-10-27
CREATE TABLE cards (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES family_members(id),
    name TEXT NOT NULL,
    limit_cents BIGINT NOT NULL CHECK (limit_cents > 0),
    closing_day SMALLINT NOT NULL CHECK (closing_day BETWEEN 1 AND 31),
    due_day SMALLINT NOT NULL CHECK (due_day BETWEEN 1 AND 31),
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX cards_family_member_idx ON cards(family_id, member_id, status);

CREATE TABLE card_purchases (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    card_id UUID NOT NULL REFERENCES cards(id),
    description TEXT NOT NULL,
    purchased_at DATE NOT NULL,
    principal_cents BIGINT NOT NULL CHECK (principal_cents > 0),
    financed_total_cents BIGINT NOT NULL CHECK (financed_total_cents > 0),
    interest_cents BIGINT NOT NULL CHECK (interest_cents >= 0),
    installment_count INTEGER NOT NULL CHECK (installment_count > 0),
    version INTEGER NOT NULL DEFAULT 1,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX card_purchases_card_idx ON card_purchases(card_id, purchased_at, id);

CREATE TABLE card_invoices (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    card_id UUID NOT NULL REFERENCES cards(id),
    reference_month DATE NOT NULL,
    closing_at DATE NOT NULL,
    due_at DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED', 'PAID', 'PARTIALLY_PAID')),
    total_cents BIGINT NOT NULL DEFAULT 0,
    paid_cents BIGINT NOT NULL DEFAULT 0,
    UNIQUE (card_id, reference_month)
);

CREATE TABLE card_installments (
    id UUID PRIMARY KEY,
    purchase_id UUID NOT NULL REFERENCES card_purchases(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES card_invoices(id),
    number INTEGER NOT NULL CHECK (number > 0),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    status TEXT NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CONFIRMED', 'PAID', 'CANCELLED')),
    UNIQUE (purchase_id, number)
);

CREATE TABLE card_payments (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES card_invoices(id),
    account_id UUID NOT NULL REFERENCES accounts(id),
    operation_id UUID NOT NULL UNIQUE REFERENCES operations(id),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    idempotency_key TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (family_id, idempotency_key)
);
