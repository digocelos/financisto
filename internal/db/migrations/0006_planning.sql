CREATE TABLE IF NOT EXISTS planning_groups (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (family_id, name)
);

CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES categories(id),
    name TEXT NOT NULL,
    nature TEXT NOT NULL CHECK (nature IN ('INCOME', 'EXPENSE', 'BOTH')),
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (family_id, parent_id, name)
);

CREATE INDEX IF NOT EXISTS categories_family_idx ON categories (family_id, parent_id, created_at);

CREATE TABLE IF NOT EXISTS recurring_rules (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    category_id UUID REFERENCES categories(id),
    planning_group_id UUID REFERENCES planning_groups(id),
    type TEXT NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),
    description TEXT NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    frequency TEXT NOT NULL CHECK (frequency IN ('MONTHLY')),
    day_of_month INTEGER NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
    starts_on DATE NOT NULL,
    ends_on DATE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS recurring_occurrences (
    id UUID PRIMARY KEY,
    recurring_rule_id UUID NOT NULL REFERENCES recurring_rules(id) ON DELETE CASCADE,
    operation_id UUID NOT NULL UNIQUE REFERENCES operations(id) ON DELETE CASCADE,
    effective_at DATE NOT NULL,
    overridden_at TIMESTAMPTZ,
    UNIQUE (recurring_rule_id, effective_at)
);

ALTER TABLE operations ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id);
ALTER TABLE operations ADD COLUMN IF NOT EXISTS planning_group_id UUID REFERENCES planning_groups(id);
ALTER TABLE operations ADD COLUMN IF NOT EXISTS recurring_rule_id UUID REFERENCES recurring_rules(id);

CREATE UNIQUE INDEX IF NOT EXISTS operations_recurring_idx ON operations (recurring_rule_id, effective_at)
    WHERE recurring_rule_id IS NOT NULL AND deleted_at IS NULL;
