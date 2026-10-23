export const operationTypes = [
  { value: 'INCOME', label: 'Receita' },
  { value: 'EXPENSE', label: 'Despesa' },
  { value: 'TRANSFER', label: 'Transferência' },
  { value: 'ADJUSTMENT', label: 'Ajuste' },
]

export const operationStatuses = [
  { value: 'CONFIRMED', label: 'Confirmado' },
  { value: 'PLANNED', label: 'Planejado' },
  { value: 'PARTIALLY_CONFIRMED', label: 'Parcial' },
  { value: 'CANCELLED', label: 'Cancelado' },
]

export function amountToCents(value) {
  const normalized = String(value ?? '').trim().replace(/\./g, '').replace(',', '.')
  const amount = Number(normalized)
  if (!Number.isFinite(amount) || amount <= 0) return 0
  return Math.round(amount * 100)
}

export function buildOperationPayload(form) {
  const payload = {
    type: form.type,
    status: form.status,
    accountId: form.accountId,
    amountCents: amountToCents(form.amount),
    description: form.description.trim(),
    effectiveAt: form.effectiveAt,
    dueAt: form.dueAt,
  }
  if (form.destinationAccountId) payload.destinationAccountId = form.destinationAccountId
  return payload
}

export function formatCents(cents) {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format((cents ?? 0) / 100)
}
