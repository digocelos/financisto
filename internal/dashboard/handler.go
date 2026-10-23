package dashboard

import (
	"fmt"
	"net/http"
	"time"

	"github.com/digocelos/financisto/internal/auth"
	"github.com/digocelos/financisto/internal/httpapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Service
}

func NewHandler(db *pgxpool.Pool, authService *auth.Service) *Handler {
	return &Handler{db: db, auth: authService}
}

func (h *Handler) MountRoutes(mux *http.ServeMux) {
	mux.Handle("GET /v1/families/{familyID}/dashboard", h.auth.RequireUser(http.HandlerFunc(h.get)))
}

type dashboardResponse struct {
	From                  time.Time          `json:"from"`
	To                    time.Time          `json:"to"`
	CurrentBalanceCents   int64              `json:"currentBalanceCents"`
	ProjectedBalanceCents int64              `json:"projectedBalanceCents"`
	IncomeCents           int64              `json:"incomeCents"`
	ExpenseCents          int64              `json:"expenseCents"`
	Accounts              []accountSummary   `json:"accounts"`
	Operations            []operationSummary `json:"operations"`
}

type accountSummary struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	MemberID              uuid.UUID `json:"memberId"`
	CurrentBalanceCents   int64     `json:"currentBalanceCents"`
	ProjectedBalanceCents int64     `json:"projectedBalanceCents"`
	Status                string    `json:"status"`
}

type operationSummary struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	EffectiveAt time.Time `json:"effectiveAt"`
	AmountCents int64     `json:"amountCents"`
	AccountID   uuid.UUID `json:"accountId"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	from, to, err := dateRange(r)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	memberID, accountID, status, err := filters(r)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	response := dashboardResponse{From: from, To: to, Accounts: make([]accountSummary, 0), Operations: make([]operationSummary, 0)}
	accountRows, err := h.db.Query(r.Context(), `SELECT a.id,a.name,a.member_id,a.status,COALESCE(o.amount,0)+COALESCE(e.current_amount,0),COALESCE(o.amount,0)+COALESCE(e.projected_amount,0) FROM accounts a LEFT JOIN LATERAL (SELECT amount_cents AS amount FROM account_opening_operations WHERE account_id=a.id) o ON TRUE LEFT JOIN LATERAL (SELECT SUM(CASE WHEN op.status IN ('CONFIRMED','PARTIALLY_CONFIRMED') AND op.deleted_at IS NULL THEN e.amount_cents ELSE 0 END) AS current_amount,SUM(CASE WHEN op.status IN ('CONFIRMED','PARTIALLY_CONFIRMED','PLANNED') AND op.deleted_at IS NULL THEN e.amount_cents ELSE 0 END) AS projected_amount FROM operation_entries e JOIN operations op ON op.id=e.operation_id WHERE e.account_id=a.id) e ON TRUE WHERE a.family_id=$1 AND ($2::uuid IS NULL OR a.member_id=$2) AND ($3::uuid IS NULL OR a.id=$3) ORDER BY a.created_at`, familyID, memberID, accountID)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load dashboard")
		return
	}
	defer accountRows.Close()
	for accountRows.Next() {
		var account accountSummary
		if err := accountRows.Scan(&account.ID, &account.Name, &account.MemberID, &account.Status, &account.CurrentBalanceCents, &account.ProjectedBalanceCents); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load dashboard")
			return
		}
		response.CurrentBalanceCents += account.CurrentBalanceCents
		response.ProjectedBalanceCents += account.ProjectedBalanceCents
		response.Accounts = append(response.Accounts, account)
	}
	if err := accountRows.Err(); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load dashboard")
		return
	}
	rows, err := h.db.Query(r.Context(), `SELECT COALESCE(SUM(CASE WHEN o.type='INCOME' THEN e.amount_cents ELSE 0 END),0),COALESCE(SUM(CASE WHEN o.type='EXPENSE' THEN -e.amount_cents ELSE 0 END),0) FROM operations o JOIN operation_entries e ON e.operation_id=o.id JOIN accounts a ON a.id=e.account_id WHERE o.family_id=$1 AND o.deleted_at IS NULL AND o.effective_at BETWEEN $4 AND $5 AND ($2::uuid IS NULL OR a.member_id=$2) AND ($3::uuid IS NULL OR a.id=$3) AND ($6::text IS NULL OR o.status=$6)`, familyID, memberID, accountID, from, to, status)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load dashboard")
		return
	}
	if rows.Next() {
		if err := rows.Scan(&response.IncomeCents, &response.ExpenseCents); err != nil {
			rows.Close()
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load dashboard")
			return
		}
	}
	rows.Close()
	operationRows, err := h.db.Query(r.Context(), `SELECT o.id,o.type,o.status,o.description,o.effective_at,e.amount_cents,e.account_id FROM operations o JOIN operation_entries e ON e.operation_id=o.id JOIN accounts a ON a.id=e.account_id WHERE o.family_id=$1 AND o.deleted_at IS NULL AND o.effective_at BETWEEN $4 AND $5 AND ($2::uuid IS NULL OR a.member_id=$2) AND ($3::uuid IS NULL OR a.id=$3) AND ($6::text IS NULL OR o.status=$6) ORDER BY o.effective_at DESC,o.created_at DESC LIMIT 100`, familyID, memberID, accountID, from, to, status)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load dashboard")
		return
	}
	defer operationRows.Close()
	for operationRows.Next() {
		var operation operationSummary
		if err := operationRows.Scan(&operation.ID, &operation.Type, &operation.Status, &operation.Description, &operation.EffectiveAt, &operation.AmountCents, &operation.AccountID); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load dashboard")
			return
		}
		response.Operations = append(response.Operations, operation)
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) familyID(r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, false
	}
	familyID, err := uuid.Parse(r.PathValue("familyID"))
	if err != nil {
		return uuid.Nil, false
	}
	var exists bool
	err = h.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM family_members m JOIN families f ON f.id=m.family_id WHERE m.family_id=$1 AND m.user_id=$2 AND m.archived_at IS NULL AND f.blocked_at IS NULL)`, familyID, userID).Scan(&exists)
	return familyID, err == nil && exists
}

func dateRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-time.Nanosecond)
	if value := r.URL.Query().Get("from"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("from must use YYYY-MM-DD")
		}
		from = parsed
	}
	if value := r.URL.Query().Get("to"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("to must use YYYY-MM-DD")
		}
		to = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	return from, to, nil
}

func filters(r *http.Request) (*uuid.UUID, *uuid.UUID, *string, error) {
	parse := func(name string) (*uuid.UUID, error) {
		value := r.URL.Query().Get(name)
		if value == "" {
			return nil, nil
		}
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("%s must be a UUID", name)
		}
		return &id, nil
	}
	member, err := parse("memberId")
	if err != nil {
		return nil, nil, nil, err
	}
	account, err := parse("accountId")
	if err != nil {
		return nil, nil, nil, err
	}
	statusValue := r.URL.Query().Get("status")
	var status *string
	if statusValue != "" {
		status = &statusValue
	}
	return member, account, status, nil
}
