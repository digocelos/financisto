package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/digocelos/financisto/internal/auth"
	"github.com/digocelos/financisto/internal/httpapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	mux.Handle("GET /v1/families/{familyID}/operations", h.auth.RequireUser(http.HandlerFunc(h.list)))
	mux.Handle("POST /v1/families/{familyID}/operations", h.auth.RequireUser(http.HandlerFunc(h.create)))
	mux.Handle("PATCH /v1/families/{familyID}/operations/{operationID}", h.auth.RequireUser(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /v1/families/{familyID}/operations/{operationID}", h.auth.RequireUser(http.HandlerFunc(h.delete)))
}

type operationRequest struct {
	Type                 string     `json:"type"`
	Status               string     `json:"status"`
	AccountID            uuid.UUID  `json:"accountId"`
	DestinationAccountID uuid.UUID  `json:"destinationAccountId"`
	AmountCents          int64      `json:"amountCents"`
	Description          string     `json:"description"`
	EffectiveAt          string     `json:"effectiveAt"`
	DueAt                string     `json:"dueAt"`
	CategoryID           *uuid.UUID `json:"categoryId"`
	PlanningGroupID      *uuid.UUID `json:"planningGroupId"`
}

type operationResponse struct {
	ID          uuid.UUID       `json:"id"`
	Type        string          `json:"type"`
	Status      string          `json:"status"`
	Description string          `json:"description"`
	EffectiveAt time.Time       `json:"effectiveAt"`
	DueAt       *time.Time      `json:"dueAt,omitempty"`
	Version     int             `json:"version"`
	DeletedAt   *time.Time      `json:"deletedAt,omitempty"`
	Entries     []entryResponse `json:"entries"`
}

type entryResponse struct {
	ID                 uuid.UUID `json:"id"`
	AccountID          uuid.UUID `json:"accountId"`
	AmountCents        int64     `json:"amountCents"`
	BalanceBeforeCents int64     `json:"balanceBeforeCents"`
	BalanceAfterCents  int64     `json:"balanceAfterCents"`
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
	err = h.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM family_members m JOIN families f ON f.id = m.family_id WHERE m.family_id = $1 AND m.user_id = $2 AND m.archived_at IS NULL AND f.blocked_at IS NULL)`, familyID, userID).Scan(&exists)
	return familyID, err == nil && exists
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	var input operationRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpapi.WriteProblem(w, http.StatusBadRequest, "invalid operation")
		return
	}
	if err := validateOperation(input); err != nil {
		httpapi.WriteProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	effectiveAt, dueAt, err := parseDates(input.EffectiveAt, input.DueAt)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not create operation")
		return
	}
	defer tx.Rollback(r.Context())
	accountIDs := []uuid.UUID{input.AccountID}
	if input.Type == "TRANSFER" {
		accountIDs = append(accountIDs, input.DestinationAccountID)
	}
	if err := h.validateAccounts(r.Context(), tx, familyID, accountIDs); err != nil {
		httpapi.WriteProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.CategoryID != nil {
		var nature string
		if err := tx.QueryRow(r.Context(), `SELECT nature FROM categories WHERE id=$1 AND family_id=$2 AND archived_at IS NULL`, *input.CategoryID, familyID).Scan(&nature); err != nil || (nature != "BOTH" && nature != input.Type) {
			httpapi.WriteProblem(w, http.StatusBadRequest, "category is incompatible with operation type")
			return
		}
	}
	opID := uuid.New()
	status := input.Status
	if status == "" {
		status = "CONFIRMED"
	}
	var confirmedAt *time.Time
	if status == "CONFIRMED" || status == "PARTIALLY_CONFIRMED" {
		now := time.Now().UTC()
		confirmedAt = &now
	}
	if _, err := tx.Exec(r.Context(), `INSERT INTO operations (id, family_id, type, status, description, effective_at, due_at, confirmed_at, category_id, planning_group_id, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, opID, familyID, input.Type, status, strings.TrimSpace(input.Description), effectiveAt, dueAt, confirmedAt, input.CategoryID, input.PlanningGroupID, userID); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not create operation")
		return
	}
	entries := []struct {
		accountID uuid.UUID
		amount    int64
	}{{input.AccountID, signedAmount(input.Type, input.AmountCents)}}
	if input.Type == "TRANSFER" {
		entries = append(entries, struct {
			accountID uuid.UUID
			amount    int64
		}{input.DestinationAccountID, input.AmountCents})
	}
	for _, entry := range entries {
		if _, err := tx.Exec(r.Context(), `INSERT INTO operation_entries (id, operation_id, account_id, amount_cents, balance_before_cents, balance_after_cents) VALUES ($1,$2,$3,$4,0,0)`, uuid.New(), opID, entry.accountID, entry.amount); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not create entries")
			return
		}
	}
	if err := h.rebuildBalances(r.Context(), tx, entriesAccountIDs(entries)); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not rebuild balances")
		return
	}
	if err := h.audit(r.Context(), tx, familyID, userID, opID, "created", nil, input); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not audit operation")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not create operation")
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, map[string]any{"id": opID, "version": 1})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	query := `SELECT o.id,o.type,o.status,o.description,o.effective_at,o.due_at,o.version,o.deleted_at,e.id,e.account_id,e.amount_cents,e.balance_before_cents,e.balance_after_cents FROM operations o LEFT JOIN operation_entries e ON e.operation_id=o.id WHERE o.family_id=$1`
	if r.URL.Query().Get("includeDeleted") != "true" {
		query += ` AND o.deleted_at IS NULL`
	}
	query += ` ORDER BY o.effective_at DESC,o.created_at DESC,o.id,e.created_at,e.id`
	rows, err := h.db.Query(r.Context(), query, familyID)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load operations")
		return
	}
	defer rows.Close()
	operations := make([]operationResponse, 0)
	byID := make(map[uuid.UUID]int)
	for rows.Next() {
		var operation operationResponse
		var dueAt, deletedAt *time.Time
		var entryID, accountID *uuid.UUID
		var amount, before, after *int64
		if err := rows.Scan(&operation.ID, &operation.Type, &operation.Status, &operation.Description, &operation.EffectiveAt, &dueAt, &operation.Version, &deletedAt, &entryID, &accountID, &amount, &before, &after); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load operations")
			return
		}
		operation.DueAt, operation.DeletedAt = dueAt, deletedAt
		index, exists := byID[operation.ID]
		if !exists {
			operations = append(operations, operation)
			index = len(operations) - 1
			byID[operation.ID] = index
		}
		if entryID != nil {
			operations[index].Entries = append(operations[index].Entries, entryResponse{ID: *entryID, AccountID: *accountID, AmountCents: *amount, BalanceBeforeCents: *before, BalanceAfterCents: *after})
		}
	}
	httpapi.WriteJSON(w, http.StatusOK, operations)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	operationID, err := uuid.Parse(r.PathValue("operationID"))
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "operation not found")
		return
	}
	var input struct {
		Version     int     `json:"version"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		AmountCents *int64  `json:"amountCents"`
		EffectiveAt *string `json:"effectiveAt"`
		DueAt       *string `json:"dueAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Version < 1 {
		httpapi.WriteProblem(w, http.StatusBadRequest, "version and update are required")
		return
	}
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not update operation")
		return
	}
	defer tx.Rollback(r.Context())
	var typeName, status, description string
	var effectiveAt time.Time
	var dueAt *time.Time
	var currentVersion int
	if err := tx.QueryRow(r.Context(), `SELECT type,status,description,effective_at,due_at,version FROM operations WHERE id=$1 AND family_id=$2 AND deleted_at IS NULL FOR UPDATE`, operationID, familyID).Scan(&typeName, &status, &description, &effectiveAt, &dueAt, &currentVersion); err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "operation not found")
		return
	}
	if currentVersion != input.Version {
		httpapi.WriteProblem(w, http.StatusConflict, "operation was changed by another request")
		return
	}
	before := map[string]any{"description": description, "status": status, "effectiveAt": effectiveAt.Format("2006-01-02")}
	if input.Description != nil {
		description = strings.TrimSpace(*input.Description)
	}
	if input.Status != nil {
		if !validStatus(*input.Status) {
			httpapi.WriteProblem(w, http.StatusBadRequest, "unsupported operation status")
			return
		}
		status = *input.Status
	}
	if input.EffectiveAt != nil {
		parsed, parseErr := time.Parse("2006-01-02", *input.EffectiveAt)
		if parseErr != nil {
			httpapi.WriteProblem(w, http.StatusBadRequest, "effectiveAt must use YYYY-MM-DD")
			return
		}
		effectiveAt = parsed
	}
	if input.DueAt != nil {
		if *input.DueAt == "" {
			dueAt = nil
		} else {
			parsed, parseErr := time.Parse("2006-01-02", *input.DueAt)
			if parseErr != nil {
				httpapi.WriteProblem(w, http.StatusBadRequest, "dueAt must use YYYY-MM-DD")
				return
			}
			dueAt = &parsed
		}
	}
	rows, err := tx.Query(r.Context(), `SELECT id,account_id,amount_cents FROM operation_entries WHERE operation_id=$1 ORDER BY created_at,id`, operationID)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load entries")
		return
	}
	type entry struct {
		id, accountID uuid.UUID
		amount        int64
	}
	entries := make([]entry, 0)
	for rows.Next() {
		var item entry
		if err := rows.Scan(&item.id, &item.accountID, &item.amount); err != nil {
			rows.Close()
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load entries")
			return
		}
		entries = append(entries, item)
	}
	rows.Close()
	if input.AmountCents != nil {
		if *input.AmountCents <= 0 {
			httpapi.WriteProblem(w, http.StatusBadRequest, "amountCents must be positive")
			return
		}
		for index := range entries {
			if typeName == "EXPENSE" || (typeName == "TRANSFER" && index == 0) {
				entries[index].amount = -*input.AmountCents
			} else {
				entries[index].amount = *input.AmountCents
			}
		}
	}
	if _, err := tx.Exec(r.Context(), `UPDATE operations SET description=$1,status=$2,effective_at=$3,due_at=$4,confirmed_at=CASE WHEN $2 IN ('CONFIRMED','PARTIALLY_CONFIRMED') THEN COALESCE(confirmed_at,NOW()) ELSE confirmed_at END,version=version+1,updated_at=NOW() WHERE id=$5 AND family_id=$6 AND version=$7`, description, status, effectiveAt, dueAt, operationID, familyID, input.Version); err != nil {
		httpapi.WriteProblem(w, http.StatusConflict, "operation was changed by another request")
		return
	}
	for _, entry := range entries {
		if _, err := tx.Exec(r.Context(), `UPDATE operation_entries SET amount_cents=$1 WHERE id=$2`, entry.amount, entry.id); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not update entries")
			return
		}
	}
	accountIDs := make([]uuid.UUID, 0, len(entries))
	for _, entry := range entries {
		accountIDs = append(accountIDs, entry.accountID)
	}
	if err := h.rebuildBalances(r.Context(), tx, accountIDs); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not rebuild balances")
		return
	}
	if err := h.audit(r.Context(), tx, familyID, userID, operationID, "updated", before, map[string]any{"description": description, "status": status, "effectiveAt": effectiveAt.Format("2006-01-02")}); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not audit operation")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpapi.WriteProblem(w, http.StatusConflict, "operation was changed by another request")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"id": operationID, "version": input.Version + 1})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	operationID, err := uuid.Parse(r.PathValue("operationID"))
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "operation not found")
		return
	}
	var version int
	if _, err := fmt.Sscanf(strings.TrimSpace(r.Header.Get("If-Match")), "%d", &version); err != nil || version < 1 {
		httpapi.WriteProblem(w, http.StatusBadRequest, "If-Match version is required")
		return
	}
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not delete operation")
		return
	}
	defer tx.Rollback(r.Context())
	rows, err := tx.Query(r.Context(), `SELECT account_id FROM operation_entries WHERE operation_id=$1`, operationID)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load operation")
		return
	}
	var accountIDs []uuid.UUID
	for rows.Next() {
		var accountID uuid.UUID
		if err := rows.Scan(&accountID); err != nil {
			rows.Close()
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load operation")
			return
		}
		accountIDs = append(accountIDs, accountID)
	}
	rows.Close()
	result, err := tx.Exec(r.Context(), `UPDATE operations SET deleted_at=NOW(), version=version+1, updated_at=NOW() WHERE id=$1 AND family_id=$2 AND deleted_at IS NULL AND version=$3`, operationID, familyID, version)
	if err != nil || result.RowsAffected() == 0 {
		httpapi.WriteProblem(w, http.StatusConflict, "operation was changed or already deleted")
		return
	}
	if err := h.rebuildBalances(r.Context(), tx, accountIDs); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not rebuild balances")
		return
	}
	if err := h.audit(r.Context(), tx, familyID, userID, operationID, "deleted", nil, map[string]any{"deleted": true}); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not audit operation")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not delete operation")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true, "actorId": userID})
}

func (h *Handler) validateAccounts(ctx context.Context, tx pgx.Tx, familyID uuid.UUID, ids []uuid.UUID) error {
	for _, id := range ids {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM accounts WHERE id=$1 AND family_id=$2)`, id, familyID).Scan(&exists); err != nil || !exists {
			return errors.New("account is not part of this family")
		}
	}
	return nil
}

func (h *Handler) rebuildBalances(ctx context.Context, tx pgx.Tx, accountIDs []uuid.UUID) error {
	for _, accountID := range accountIDs {
		var balance int64
		if err := tx.QueryRow(ctx, `SELECT amount_cents FROM account_opening_operations WHERE account_id=$1`, accountID).Scan(&balance); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT e.id,e.amount_cents FROM operation_entries e JOIN operations o ON o.id=e.operation_id WHERE e.account_id=$1 AND o.deleted_at IS NULL AND o.status IN ('CONFIRMED','PARTIALLY_CONFIRMED') ORDER BY o.effective_at,o.created_at,o.id,e.created_at,e.id FOR UPDATE`, accountID)
		if err != nil {
			return err
		}
		type balanceEntry struct {
			id     uuid.UUID
			amount int64
		}
		entries := make([]balanceEntry, 0)
		for rows.Next() {
			var id uuid.UUID
			var amount int64
			if err := rows.Scan(&id, &amount); err != nil {
				rows.Close()
				return err
			}
			entries = append(entries, balanceEntry{id: id, amount: amount})
		}
		rows.Close()
		for _, entry := range entries {
			before := balance
			balance += entry.amount
			if _, err := tx.Exec(ctx, `UPDATE operation_entries SET balance_before_cents=$1,balance_after_cents=$2 WHERE id=$3`, before, balance, entry.id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) audit(ctx context.Context, tx pgx.Tx, familyID, actorID, entityID uuid.UUID, action string, before, after any) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	_, err := tx.Exec(ctx, `INSERT INTO audit_events (id,family_id,actor_id,entity_type,entity_id,action,before_data,after_data) VALUES ($1,$2,$3,'operation',$4,$5,$6,$7)`, uuid.New(), familyID, actorID, entityID, action, beforeJSON, afterJSON)
	return err
}

func validateOperation(input operationRequest) error {
	if input.AmountCents <= 0 {
		return errors.New("amountCents must be positive")
	}
	switch input.Type {
	case "INCOME", "EXPENSE", "TRANSFER", "ADJUSTMENT":
	default:
		return errors.New("unsupported operation type")
	}
	if input.Type == "TRANSFER" && (input.DestinationAccountID == uuid.Nil || input.DestinationAccountID == input.AccountID) {
		return errors.New("transfer requires a different destination account")
	}
	return nil
}

func parseDates(effective, due string) (time.Time, *time.Time, error) {
	if effective == "" {
		return time.Time{}, nil, errors.New("effectiveAt is required")
	}
	effectiveAt, err := time.Parse("2006-01-02", effective)
	if err != nil {
		return time.Time{}, nil, errors.New("effectiveAt must use YYYY-MM-DD")
	}
	if due == "" {
		return effectiveAt, nil, nil
	}
	dueAt, err := time.Parse("2006-01-02", due)
	if err != nil {
		return time.Time{}, nil, errors.New("dueAt must use YYYY-MM-DD")
	}
	return effectiveAt, &dueAt, nil
}

func signedAmount(operationType string, amount int64) int64 {
	if operationType == "EXPENSE" {
		return -amount
	}
	return amount
}

func validStatus(status string) bool {
	switch status {
	case "PLANNED", "CONFIRMED", "PARTIALLY_CONFIRMED", "CANCELLED":
		return true
	default:
		return false
	}
}

func entriesAccountIDs(entries []struct {
	accountID uuid.UUID
	amount    int64
}) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.accountID)
	}
	return ids
}
