<!-- approved: automatic 2026-10-28 -->

# Technical Requirements: Goals

## Assumptions

- ⚠️ UX risk: no standalone design validation ran; reuse the dashboard/card workspace visual language.
- Progress is derived from immutable contribution/movement events.
- Monetary values remain integer cents and family-scoped.

## Contexts

| Context | Responsibility | Owns | Talks to |
|---|---|---|---|
| `investments` | Investment account and returns lifecycle | investment accounts and movements | auth, ledger |
| `vaults` | Allocations inside real accounts | vaults and vault movements | accounts, ledger |
| `goals` | Targets, links, plans and progress | goals, goal links, progress events | vaults, ledger, cards |

## Data Flow

Create resource → validate Member/Family ownership → write immutable movement/link event → derive balance/progress query → audit transition. Goal purchase integration consumes paid principal only and uses a unique purchase-goal contribution key.

## Failure Modes

- Cross-family resource: reject without exposing data.
- Vault allocation exceeds account availability: reject atomically.
- Duplicate purchase contribution: return existing contribution.
- Reversing completion: create a new state transition; never delete history.
- Concurrent movement: transaction/version conflict, no silent overwrite.

## API Surface

- Investment account CRUD and movements.
- Vault CRUD, movement and goal links.
- Goal CRUD, plan updates, progress and purchase links.
- All endpoints require authenticated family membership and existing error conventions.

## Dependencies

Go 1.27, PostgreSQL 16 local, pgx/v5 5.8.0 and existing React/Vite frontend.
