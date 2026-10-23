import { describe, expect, it } from 'vitest'
import { buildOperationPayload } from './operations.js'

describe('buildOperationPayload', () => {
  it('converts a user-entered amount into ledger cents', () => {
    expect(buildOperationPayload({
      type: 'EXPENSE',
      status: 'CONFIRMED',
      accountId: 'account-1',
      destinationAccountId: '',
      amount: '123,45',
      description: 'Mercado',
      effectiveAt: '2026-10-23',
      dueAt: '',
    })).toEqual({
      type: 'EXPENSE',
      status: 'CONFIRMED',
      accountId: 'account-1',
      amountCents: 12345,
      description: 'Mercado',
      effectiveAt: '2026-10-23',
      dueAt: '',
    })
  })
})
