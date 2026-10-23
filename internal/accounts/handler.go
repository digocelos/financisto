package accounts

import (
	"encoding/json"
	"net/http"
	"strings"
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
	mux.Handle("GET /v1/catalogs", h.auth.RequireUser(http.HandlerFunc(h.catalogs)))
	mux.Handle("GET /v1/families/{familyID}/accounts", h.auth.RequireUser(http.HandlerFunc(h.list)))
	mux.Handle("POST /v1/families/{familyID}/accounts", h.auth.RequireUser(http.HandlerFunc(h.create)))
	mux.Handle("GET /v1/families/{familyID}/accounts/{accountID}", h.auth.RequireUser(http.HandlerFunc(h.get)))
	mux.Handle("POST /v1/families/{familyID}/accounts/{accountID}/archive", h.auth.RequireUser(http.HandlerFunc(h.archive)))
}

type accountRequest struct {
	MemberID      uuid.UUID `json:"memberId"`
	InstitutionID uuid.UUID `json:"institutionId"`
	AccountTypeID uuid.UUID `json:"accountTypeId"`
	Name          string    `json:"name"`
	Currency      string    `json:"currency"`
	OpeningCents  int64     `json:"openingBalanceCents"`
	OpeningAt     string    `json:"openingBalanceAt"`
}

type accountResponse struct {
	ID            uuid.UUID `json:"id"`
	FamilyID      uuid.UUID `json:"familyId"`
	MemberID      uuid.UUID `json:"memberId"`
	InstitutionID uuid.UUID `json:"institutionId"`
	Institution   string    `json:"institution"`
	IconURL       string    `json:"iconUrl,omitempty"`
	AccountTypeID uuid.UUID `json:"accountTypeId"`
	AccountType   string    `json:"accountType"`
	Name          string    `json:"name"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	OpeningCents  int64     `json:"openingBalanceCents"`
	OpeningAt     time.Time `json:"openingBalanceAt"`
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
	var memberExists bool
	if err := h.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM family_members m JOIN families f ON f.id = m.family_id WHERE m.family_id = $1 AND m.user_id = $2 AND m.archived_at IS NULL AND f.blocked_at IS NULL)`, familyID, userID).Scan(&memberExists); err != nil || !memberExists {
		return uuid.Nil, false
	}
	return familyID, true
}

func (h *Handler) catalogs(w http.ResponseWriter, r *http.Request) {
	var institutions []map[string]any
	rows, err := h.db.Query(r.Context(), `SELECT id, name, COALESCE(icon_url, '') FROM financial_institutions WHERE active ORDER BY name`)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load catalogs")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var name, icon string
		if err := rows.Scan(&id, &name, &icon); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load catalogs")
			return
		}
		institutions = append(institutions, map[string]any{"id": id, "name": name, "iconUrl": icon})
	}
	var types []map[string]any
	rows, err = h.db.Query(r.Context(), `SELECT id, code, name FROM account_types WHERE active ORDER BY name`)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load catalogs")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var code, name string
		if err := rows.Scan(&id, &code, &name); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load catalogs")
			return
		}
		types = append(types, map[string]any{"id": id, "code": code, "name": name})
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"institutions": institutions, "accountTypes": types})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	var input accountRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Name) == "" || input.Currency != "BRL" || input.MemberID == uuid.Nil || input.InstitutionID == uuid.Nil || input.AccountTypeID == uuid.Nil {
		httpapi.WriteProblem(w, http.StatusBadRequest, "member, institution, account type, name and BRL currency are required")
		return
	}
	openingAt := time.Now().UTC()
	if input.OpeningAt != "" {
		parsed, err := time.Parse("2006-01-02", input.OpeningAt)
		if err != nil {
			httpapi.WriteProblem(w, http.StatusBadRequest, "openingBalanceAt must use YYYY-MM-DD")
			return
		}
		openingAt = parsed
	}
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not create account")
		return
	}
	defer tx.Rollback(r.Context())
	var institution, icon, accountType string
	if err := tx.QueryRow(r.Context(), `SELECT i.name, COALESCE(i.icon_url, ''), t.name FROM financial_institutions i, account_types t WHERE i.id = $1 AND t.id = $2 AND i.active AND t.active`, input.InstitutionID, input.AccountTypeID).Scan(&institution, &icon, &accountType); err != nil {
		httpapi.WriteProblem(w, http.StatusBadRequest, "institution or account type is unavailable")
		return
	}
	var memberExists bool
	if err := tx.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM family_members WHERE id = $1 AND family_id = $2 AND archived_at IS NULL)`, input.MemberID, familyID).Scan(&memberExists); err != nil || !memberExists {
		httpapi.WriteProblem(w, http.StatusBadRequest, "member is not part of this family")
		return
	}
	accountID := uuid.New()
	if _, err := tx.Exec(r.Context(), `INSERT INTO accounts (id, family_id, member_id, institution_id, account_type_id, name, currency) VALUES ($1, $2, $3, $4, $5, $6, 'BRL')`, accountID, familyID, input.MemberID, input.InstitutionID, input.AccountTypeID, strings.TrimSpace(input.Name)); err != nil {
		httpapi.WriteProblem(w, http.StatusConflict, "could not create account")
		return
	}
	if _, err := tx.Exec(r.Context(), `INSERT INTO account_opening_operations (id, account_id, amount_cents, effective_at, created_by) VALUES ($1, $2, $3, $4, $5)`, uuid.New(), accountID, input.OpeningCents, openingAt, userID); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not create opening operation")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not create account")
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, accountResponse{ID: accountID, FamilyID: familyID, MemberID: input.MemberID, InstitutionID: input.InstitutionID, Institution: institution, IconURL: icon, AccountTypeID: input.AccountTypeID, AccountType: accountType, Name: strings.TrimSpace(input.Name), Currency: "BRL", Status: "ACTIVE", OpeningCents: input.OpeningCents, OpeningAt: openingAt})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	rows, err := h.db.Query(r.Context(), `SELECT a.id, a.family_id, a.member_id, a.institution_id, i.name, COALESCE(i.icon_url, ''), a.account_type_id, t.name, a.name, a.currency, a.status, o.amount_cents, o.effective_at FROM accounts a JOIN financial_institutions i ON i.id = a.institution_id JOIN account_types t ON t.id = a.account_type_id JOIN account_opening_operations o ON o.account_id = a.id WHERE a.family_id = $1 ORDER BY a.created_at`, familyID)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load accounts")
		return
	}
	defer rows.Close()
	accounts := make([]accountResponse, 0)
	for rows.Next() {
		var account accountResponse
		if err := rows.Scan(&account.ID, &account.FamilyID, &account.MemberID, &account.InstitutionID, &account.Institution, &account.IconURL, &account.AccountTypeID, &account.AccountType, &account.Name, &account.Currency, &account.Status, &account.OpeningCents, &account.OpeningAt); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load accounts")
			return
		}
		accounts = append(accounts, account)
	}
	httpapi.WriteJSON(w, http.StatusOK, accounts)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	accountID, err := uuid.Parse(r.PathValue("accountID"))
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "account not found")
		return
	}
	var account accountResponse
	err = h.db.QueryRow(r.Context(), `SELECT a.id, a.family_id, a.member_id, a.institution_id, i.name, COALESCE(i.icon_url, ''), a.account_type_id, t.name, a.name, a.currency, a.status, o.amount_cents, o.effective_at FROM accounts a JOIN financial_institutions i ON i.id = a.institution_id JOIN account_types t ON t.id = a.account_type_id JOIN account_opening_operations o ON o.account_id = a.id WHERE a.id = $1 AND a.family_id = $2`, accountID, familyID).Scan(&account.ID, &account.FamilyID, &account.MemberID, &account.InstitutionID, &account.Institution, &account.IconURL, &account.AccountTypeID, &account.AccountType, &account.Name, &account.Currency, &account.Status, &account.OpeningCents, &account.OpeningAt)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "account not found")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, account)
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	accountID, err := uuid.Parse(r.PathValue("accountID"))
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "account not found")
		return
	}
	result, err := h.db.Exec(r.Context(), `UPDATE accounts SET status = 'ARCHIVED', updated_at = NOW() WHERE id = $1 AND family_id = $2`, accountID, familyID)
	if err != nil || result.RowsAffected() == 0 {
		httpapi.WriteProblem(w, http.StatusNotFound, "account not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
