CREATE TABLE IF NOT EXISTS financial_institutions (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    icon_url TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS account_types (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS operation_types (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES family_members(id),
    institution_id UUID NOT NULL REFERENCES financial_institutions(id),
    account_type_id UUID NOT NULL REFERENCES account_types(id),
    name TEXT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'BRL',
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS accounts_family_idx ON accounts (family_id, created_at, id);

CREATE TABLE IF NOT EXISTS account_opening_operations (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL,
    effective_at DATE NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO account_types (id, code, name) VALUES
    ('00000000-0000-0000-0000-000000000101', 'CHECKING', 'Conta corrente'),
    ('00000000-0000-0000-0000-000000000102', 'SAVINGS', 'Poupança'),
    ('00000000-0000-0000-0000-000000000103', 'CASH', 'Dinheiro')
ON CONFLICT (code) DO NOTHING;

INSERT INTO operation_types (id, code, name) VALUES
    ('00000000-0000-0000-0000-000000000201', 'OPENING_BALANCE', 'Saldo inicial'),
    ('00000000-0000-0000-0000-000000000202', 'INCOME', 'Receita'),
    ('00000000-0000-0000-0000-000000000203', 'EXPENSE', 'Despesa'),
    ('00000000-0000-0000-0000-000000000204', 'TRANSFER', 'Transferência')
ON CONFLICT (code) DO NOTHING;

INSERT INTO financial_institutions (id, name, icon_url) VALUES
    ('00000000-0000-0000-0000-000000000301', 'Banco do Brasil', 'https://raw.githubusercontent.com/empathetic-ai/bank-icons/main/icons/banco-do-brasil.svg'),
    ('00000000-0000-0000-0000-000000000302', 'Caixa Econômica Federal', 'https://raw.githubusercontent.com/empathetic-ai/bank-icons/main/icons/caixa.svg'),
    ('00000000-0000-0000-0000-000000000303', 'Itaú', 'https://raw.githubusercontent.com/empathetic-ai/bank-icons/main/icons/itau.svg'),
    ('00000000-0000-0000-0000-000000000304', 'Bradesco', 'https://raw.githubusercontent.com/empathetic-ai/bank-icons/main/icons/bradesco.svg'),
    ('00000000-0000-0000-0000-000000000305', 'Nubank', 'https://raw.githubusercontent.com/empathetic-ai/bank-icons/main/icons/nubank.svg')
ON CONFLICT (name) DO NOTHING;
