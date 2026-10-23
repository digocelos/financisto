package planning

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
	mux.Handle("GET /v1/families/{familyID}/planning", h.auth.RequireUser(http.HandlerFunc(h.listPlanning)))
	mux.Handle("POST /v1/families/{familyID}/planning/groups", h.auth.RequireUser(http.HandlerFunc(h.createGroup)))
	mux.Handle("PATCH /v1/families/{familyID}/planning/groups/{groupID}/archive", h.auth.RequireUser(http.HandlerFunc(h.archiveGroup)))
	mux.Handle("POST /v1/families/{familyID}/planning/categories", h.auth.RequireUser(http.HandlerFunc(h.createCategory)))
	mux.Handle("PATCH /v1/families/{familyID}/planning/categories/{categoryID}/archive", h.auth.RequireUser(http.HandlerFunc(h.archiveCategory)))
	mux.Handle("POST /v1/families/{familyID}/recurrences", h.auth.RequireUser(http.HandlerFunc(h.createRecurrence)))
	mux.Handle("PATCH /v1/families/{familyID}/recurrences/{ruleID}/occurrences/{operationID}", h.auth.RequireUser(http.HandlerFunc(h.overrideOccurrence)))
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

type groupInput struct {
	Name string `json:"name"`
}
type categoryInput struct {
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parentId"`
	Nature   string     `json:"nature"`
}

func (h *Handler) listPlanning(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	groups := make([]map[string]any, 0)
	rows, err := h.db.Query(r.Context(), `SELECT id,name,archived_at FROM planning_groups WHERE family_id=$1 ORDER BY created_at`, familyID)
	if err != nil {
		httpapi.WriteProblem(w, 500, "could not load planning")
		return
	}
	for rows.Next() {
		var id uuid.UUID
		var name string
		var archived *time.Time
		_ = rows.Scan(&id, &name, &archived)
		groups = append(groups, map[string]any{"id": id, "name": name, "archivedAt": archived})
	}
	rows.Close()
	categories := make([]map[string]any, 0)
	rows, err = h.db.Query(r.Context(), `SELECT id,parent_id,name,nature,archived_at FROM categories WHERE family_id=$1 ORDER BY created_at`, familyID)
	if err != nil {
		httpapi.WriteProblem(w, 500, "could not load planning")
		return
	}
	for rows.Next() {
		var id uuid.UUID
		var parent *uuid.UUID
		var name, nature string
		var archived *time.Time
		_ = rows.Scan(&id, &parent, &name, &nature, &archived)
		categories = append(categories, map[string]any{"id": id, "parentId": parent, "name": name, "nature": nature, "archivedAt": archived})
	}
	rows.Close()
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"groups": groups, "categories": categories})
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, 404, "family not found")
		return
	}
	var input groupInput
	if json.NewDecoder(r.Body).Decode(&input) != nil || strings.TrimSpace(input.Name) == "" {
		httpapi.WriteProblem(w, 400, "name is required")
		return
	}
	var id = uuid.New()
	if _, err := h.db.Exec(r.Context(), `INSERT INTO planning_groups(id,family_id,name) VALUES($1,$2,$3)`, id, familyID, strings.TrimSpace(input.Name)); err != nil {
		httpapi.WriteProblem(w, 409, "group already exists")
		return
	}
	httpapi.WriteJSON(w, 201, map[string]any{"id": id, "name": strings.TrimSpace(input.Name)})
}
func (h *Handler) archiveGroup(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, 404, "family not found")
		return
	}
	id, err := uuid.Parse(r.PathValue("groupID"))
	if err != nil {
		httpapi.WriteProblem(w, 404, "group not found")
		return
	}
	result, err := h.db.Exec(r.Context(), `UPDATE planning_groups SET archived_at=NOW() WHERE id=$1 AND family_id=$2 AND archived_at IS NULL`, id, familyID)
	if err != nil || result.RowsAffected() == 0 {
		httpapi.WriteProblem(w, 404, "group not found")
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, 404, "family not found")
		return
	}
	var input categoryInput
	if json.NewDecoder(r.Body).Decode(&input) != nil || strings.TrimSpace(input.Name) == "" || !validNature(input.Nature) {
		httpapi.WriteProblem(w, 400, "name and valid nature are required")
		return
	}
	if input.ParentID != nil {
		var depth int
		if err := h.db.QueryRow(r.Context(), `WITH RECURSIVE tree AS (SELECT id,parent_id,1 AS depth FROM categories WHERE id=$1 AND family_id=$2 UNION ALL SELECT c.id,c.parent_id,tree.depth+1 FROM categories c JOIN tree ON c.id=tree.parent_id) SELECT COALESCE(MAX(depth),0) FROM tree`, *input.ParentID, familyID).Scan(&depth); err != nil || depth >= 3 {
			httpapi.WriteProblem(w, 400, "categories support up to three levels")
			return
		}
	}
	id := uuid.New()
	if _, err := h.db.Exec(r.Context(), `INSERT INTO categories(id,family_id,parent_id,name,nature) VALUES($1,$2,$3,$4,$5)`, id, familyID, input.ParentID, strings.TrimSpace(input.Name), input.Nature); err != nil {
		httpapi.WriteProblem(w, 409, "category already exists")
		return
	}
	httpapi.WriteJSON(w, 201, map[string]any{"id": id, "name": strings.TrimSpace(input.Name), "nature": input.Nature})
}
func (h *Handler) archiveCategory(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, 404, "family not found")
		return
	}
	id, err := uuid.Parse(r.PathValue("categoryID"))
	if err != nil {
		httpapi.WriteProblem(w, 404, "category not found")
		return
	}
	result, err := h.db.Exec(r.Context(), `UPDATE categories SET archived_at=NOW() WHERE id=$1 AND family_id=$2 AND archived_at IS NULL`, id, familyID)
	if err != nil || result.RowsAffected() == 0 {
		httpapi.WriteProblem(w, 404, "category not found")
		return
	}
	w.WriteHeader(204)
}

type recurrenceInput struct {
	AccountID       uuid.UUID  `json:"accountId"`
	CategoryID      *uuid.UUID `json:"categoryId"`
	PlanningGroupID *uuid.UUID `json:"planningGroupId"`
	Type            string     `json:"type"`
	Description     string     `json:"description"`
	AmountCents     int64      `json:"amountCents"`
	Frequency       string     `json:"frequency"`
	DayOfMonth      int        `json:"dayOfMonth"`
	StartsOn        string     `json:"startsOn"`
	EndsOn          string     `json:"endsOn"`
}

func (h *Handler) createRecurrence(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, 404, "family not found")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	var input recurrenceInput
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		httpapi.WriteProblem(w, 400, "invalid recurrence")
		return
	}
	if input.Type != "INCOME" && input.Type != "EXPENSE" || input.AmountCents <= 0 || input.Frequency != "MONTHLY" || input.DayOfMonth < 1 || input.DayOfMonth > 31 {
		httpapi.WriteProblem(w, 400, "invalid recurrence fields")
		return
	}
	start, err := time.Parse("2006-01-02", input.StartsOn)
	if err != nil {
		httpapi.WriteProblem(w, 400, "startsOn must use YYYY-MM-DD")
		return
	}
	var end *time.Time
	if input.EndsOn != "" {
		parsed, e := time.Parse("2006-01-02", input.EndsOn)
		if e != nil {
			httpapi.WriteProblem(w, 400, "endsOn must use YYYY-MM-DD")
			return
		}
		end = &parsed
	}
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		httpapi.WriteProblem(w, 500, "could not create recurrence")
		return
	}
	defer tx.Rollback(r.Context())
	if err := validateRuleReferences(r.Context(), tx, familyID, input); err != nil {
		httpapi.WriteProblem(w, 400, err.Error())
		return
	}
	ruleID := uuid.New()
	if _, err := tx.Exec(r.Context(), `INSERT INTO recurring_rules(id,family_id,account_id,category_id,planning_group_id,type,description,amount_cents,frequency,day_of_month,starts_on,ends_on,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, ruleID, familyID, input.AccountID, input.CategoryID, input.PlanningGroupID, input.Type, strings.TrimSpace(input.Description), input.AmountCents, input.Frequency, input.DayOfMonth, start, end, userID); err != nil {
		httpapi.WriteProblem(w, 500, "could not create recurrence")
		return
	}
	for date := start; date.Before(time.Now().UTC().AddDate(0, 12, 0)); date = date.AddDate(0, 1, 0) {
		if end != nil && date.After(*end) {
			break
		}
		if err := insertOccurrence(r.Context(), tx, ruleID, familyID, userID, input, date); err != nil {
			httpapi.WriteProblem(w, 500, "could not generate recurrence")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpapi.WriteProblem(w, 500, "could not create recurrence")
		return
	}
	httpapi.WriteJSON(w, 201, map[string]any{"id": ruleID, "generatedMonths": 12})
}

func (h *Handler) overrideOccurrence(w http.ResponseWriter, r *http.Request) {
	familyID, ok := h.familyID(r)
	if !ok {
		httpapi.WriteProblem(w, 404, "family not found")
		return
	}
	operationID, err := uuid.Parse(r.PathValue("operationID"))
	if err != nil {
		httpapi.WriteProblem(w, 404, "occurrence not found")
		return
	}
	var input struct {
		AmountCents *int64  `json:"amountCents"`
		Description *string `json:"description"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		httpapi.WriteProblem(w, 400, "invalid override")
		return
	}
	if input.AmountCents == nil && input.Description == nil {
		httpapi.WriteProblem(w, 400, "override is empty")
		return
	}
	set := []string{}
	args := []any{}
	if input.AmountCents != nil {
		set = append(set, "amount_cents=$"+fmt.Sprint(len(args)+1))
		args = append(args, *input.AmountCents)
	}
	if input.Description != nil {
		set = append(set, "description=$"+fmt.Sprint(len(args)+1))
		args = append(args, strings.TrimSpace(*input.Description))
	}
	args = append(args, operationID, familyID)
	result, err := h.db.Exec(r.Context(), `UPDATE operations SET `+strings.Join(set, ",")+`,updated_at=NOW() WHERE id=$`+fmt.Sprint(len(args)-1)+` AND family_id=$`+fmt.Sprint(len(args))+` AND recurring_rule_id IS NOT NULL AND deleted_at IS NULL`, args...)
	if err != nil || result.RowsAffected() == 0 {
		httpapi.WriteProblem(w, 404, "occurrence not found")
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"overridden": true})
}

func validateRuleReferences(ctx context.Context, tx pgx.Tx, familyID uuid.UUID, input recurrenceInput) error {
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND family_id=$2)`, input.AccountID, familyID).Scan(&ok); err != nil || !ok {
		return errors.New("account is not part of this family")
	}
	if input.CategoryID != nil {
		var nature string
		if err := tx.QueryRow(ctx, `SELECT nature FROM categories WHERE id=$1 AND family_id=$2 AND archived_at IS NULL`, *input.CategoryID, familyID).Scan(&nature); err != nil || (nature != "BOTH" && nature != input.Type) {
			return errors.New("category is not available")
		}
	}
	return nil
}
func insertOccurrence(ctx context.Context, tx pgx.Tx, ruleID, familyID, userID uuid.UUID, input recurrenceInput, date time.Time) error {
	operationID := uuid.New()
	amount := input.AmountCents
	if input.Type == "EXPENSE" {
		amount = -amount
	}
	if _, err := tx.Exec(ctx, `INSERT INTO operations(id,family_id,type,status,description,effective_at,category_id,planning_group_id,recurring_rule_id,created_by) VALUES($1,$2,$3,'PLANNED',$4,$5,$6,$7,$8,$9)`, operationID, familyID, input.Type, input.Description, date, input.CategoryID, input.PlanningGroupID, ruleID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO operation_entries(id,operation_id,account_id,amount_cents,balance_before_cents,balance_after_cents) VALUES($1,$2,$3,$4,0,0)`, uuid.New(), operationID, input.AccountID, amount); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO recurring_occurrences(id,recurring_rule_id,operation_id,effective_at) VALUES($1,$2,$3,$4)`, uuid.New(), ruleID, operationID, date)
	return err
}
func validNature(nature string) bool {
	return nature == "INCOME" || nature == "EXPENSE" || nature == "BOTH"
}
