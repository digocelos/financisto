CREATE TABLE IF NOT EXISTS investment_accounts (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES family_members(id),
    name TEXT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'BRL',
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS investment_accounts_family_idx ON investment_accounts(family_id, member_id, status);

CREATE TABLE IF NOT EXISTS investment_movements (
    id UUID PRIMARY KEY,
    investment_account_id UUID NOT NULL REFERENCES investment_accounts(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('CONTRIBUTION', 'WITHDRAWAL', 'DIVIDEND', 'INTEREST', 'ADJUSTMENT')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    effective_at DATE NOT NULL,
    description TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS investment_movements_account_idx ON investment_movements(investment_account_id, effective_at, created_at);

CREATE TABLE IF NOT EXISTS vaults (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    name TEXT NOT NULL,
    balance_cents BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS vaults_family_idx ON vaults(family_id, account_id, status);

CREATE TABLE IF NOT EXISTS vault_movements (
    id UUID PRIMARY KEY,
    vault_id UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('DEPOSIT', 'WITHDRAWAL', 'ADJUSTMENT')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    effective_at DATE NOT NULL,
    description TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS vault_movements_vault_idx ON vault_movements(vault_id, effective_at, created_at);
