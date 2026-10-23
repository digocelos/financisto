CREATE TABLE IF NOT EXISTS operations (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('INCOME', 'EXPENSE', 'TRANSFER', 'ADJUSTMENT')),
    status TEXT NOT NULL DEFAULT 'CONFIRMED' CHECK (status IN ('PLANNED', 'CONFIRMED', 'PARTIALLY_CONFIRMED', 'CANCELLED')),
    description TEXT NOT NULL,
    effective_at DATE NOT NULL,
    due_at DATE,
    confirmed_at TIMESTAMPTZ,
    version INTEGER NOT NULL DEFAULT 1,
    deleted_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS operations_family_idx ON operations (family_id, effective_at, created_at, id);

CREATE TABLE IF NOT EXISTS operation_entries (
    id UUID PRIMARY KEY,
    operation_id UUID NOT NULL REFERENCES operations(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    amount_cents BIGINT NOT NULL,
    balance_before_cents BIGINT NOT NULL,
    balance_after_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS operation_entries_account_idx ON operation_entries (account_id, created_at, id);

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES users(id),
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    action TEXT NOT NULL,
    before_data JSONB,
    after_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS audit_events_entity_idx ON audit_events (family_id, entity_type, entity_id, created_at);
