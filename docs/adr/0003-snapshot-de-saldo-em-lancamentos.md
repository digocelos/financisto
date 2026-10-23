# Snapshot de saldo em Lançamentos

Cada Lançamento armazenará o saldo anterior e o saldo posterior da conta como snapshot materializado. Como lançamentos podem ser importados, editados ou excluídos retroativamente, alterações devem recalcular snapshots posteriores de forma transacional e auditada, em vez de tratar o snapshot como fonte imutável.
