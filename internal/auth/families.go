package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/digocelos/financisto/internal/httpapi"
	"github.com/google/uuid"
)

type memberRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type memberResponse struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Email            string     `json:"email"`
	Linked           bool       `json:"linked"`
	ArchivedAt       *time.Time `json:"archivedAt,omitempty"`
	InvitationStatus string     `json:"invitationStatus,omitempty"`
}

type invitationResponse struct {
	InvitationURL string    `json:"invitationUrl"`
	ExpiresAt     time.Time `json:"expiresAt"`
	Status        string    `json:"status"`
}

func (s *Service) addFamilyRoutes(mux *http.ServeMux) {
	protected := s.RequireUser(http.HandlerFunc(s.createMember))
	mux.Handle("POST /v1/families/{familyID}/members", protected)
	mux.Handle("GET /v1/families/{familyID}/members", s.RequireUser(http.HandlerFunc(s.listMembers)))
	mux.Handle("POST /v1/families/{familyID}/members/{memberID}/invitation", s.RequireUser(http.HandlerFunc(s.inviteMember)))
}

func (s *Service) familyIDForRequest(r *http.Request) (uuid.UUID, error) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("missing user")
	}
	familyID, err := uuid.Parse(r.PathValue("familyID"))
	if err != nil {
		return uuid.Nil, errors.New("invalid family")
	}
	var exists bool
	err = s.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM family_members m JOIN families f ON f.id = m.family_id WHERE m.family_id = $1 AND m.user_id = $2 AND m.archived_at IS NULL AND f.blocked_at IS NULL)`, familyID, userID).Scan(&exists)
	if err != nil || !exists {
		return uuid.Nil, errors.New("family unavailable")
	}
	return familyID, nil
}

func (s *Service) createMember(w http.ResponseWriter, r *http.Request) {
	familyID, err := s.familyIDForRequest(r)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	var input memberRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Name) == "" || !validEmail(input.Email) {
		httpapi.WriteProblem(w, http.StatusBadRequest, "name and valid email are required")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	memberID := uuid.New()
	_, err = s.db.Exec(r.Context(), `INSERT INTO family_members (id, family_id, name, email) VALUES ($1, $2, $3, $4)`, memberID, familyID, input.Name, input.Email)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusConflict, "a member with this email already exists")
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, memberResponse{ID: memberID, Name: input.Name, Email: input.Email})
}

func (s *Service) listMembers(w http.ResponseWriter, r *http.Request) {
	familyID, err := s.familyIDForRequest(r)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT m.id, m.name, m.email, m.user_id IS NOT NULL, m.archived_at, COALESCE(i.status, '') FROM family_members m LEFT JOIN LATERAL (SELECT CASE WHEN accepted_at IS NOT NULL THEN 'accepted' WHEN revoked_at IS NOT NULL THEN 'revoked' WHEN expires_at <= NOW() THEN 'expired' ELSE 'pending' END AS status FROM invitations WHERE member_id = m.id ORDER BY created_at DESC LIMIT 1) i ON TRUE WHERE m.family_id = $1 ORDER BY m.created_at`, familyID)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load members")
		return
	}
	defer rows.Close()
	members := make([]memberResponse, 0)
	for rows.Next() {
		var member memberResponse
		if err := rows.Scan(&member.ID, &member.Name, &member.Email, &member.Linked, &member.ArchivedAt, &member.InvitationStatus); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load members")
			return
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load members")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, members)
}

func (s *Service) inviteMember(w http.ResponseWriter, r *http.Request) {
	familyID, err := s.familyIDForRequest(r)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "family not found")
		return
	}
	memberID, err := uuid.Parse(r.PathValue("memberID"))
	if err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "member not found")
		return
	}
	var memberEmail string
	if err := s.db.QueryRow(r.Context(), `SELECT email FROM family_members WHERE id = $1 AND family_id = $2 AND user_id IS NULL AND archived_at IS NULL`, memberID, familyID).Scan(&memberEmail); err != nil {
		httpapi.WriteProblem(w, http.StatusNotFound, "member is already linked or does not exist")
		return
	}
	userID := r.Context().Value(userIDKey).(uuid.UUID)
	token, err := randomToken(32)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not create invitation")
		return
	}
	expiresAt := time.Now().UTC().Add(time.Hour)
	if _, err := s.db.Exec(r.Context(), `UPDATE invitations SET revoked_at = NOW() WHERE member_id = $1 AND accepted_at IS NULL AND revoked_at IS NULL`, memberID); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not replace invitation")
		return
	}
	_, err = s.db.Exec(r.Context(), `INSERT INTO invitations (id, family_id, member_id, invited_by, email, token_hash, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`, uuid.New(), familyID, memberID, userID, memberEmail, hashToken(token), expiresAt)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not persist invitation")
		return
	}
	base := s.invitationURLBase
	if base == "" {
		base = s.frontendURL + "/invite/"
	}
	invitationURL := strings.TrimRight(base, "/") + "/" + token
	if err := s.sendInvitationEmail(memberEmail, invitationURL); err != nil {
		_, _ = s.db.Exec(r.Context(), `UPDATE invitations SET revoked_at = NOW() WHERE token_hash = $1`, hashToken(token))
		httpapi.WriteProblem(w, http.StatusBadGateway, "could not send invitation email")
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, invitationResponse{InvitationURL: invitationURL, ExpiresAt: expiresAt, Status: "pending"})
}

func (s *Service) sendInvitationEmail(recipient, invitationURL string) error {
	if s.smtpHost == "" {
		return nil
	}
	from := s.smtpFrom
	if from == "" {
		return errors.New("SMTP_FROM is required when SMTP is configured")
	}
	port := s.smtpPort
	if port == 0 {
		port = 587
	}
	address := s.smtpHost + ":" + strconv.Itoa(port)
	message := "To: " + recipient + "\r\n" +
		"Subject: Convite para o Financisto\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Você foi convidado para uma família no Financisto. Acesse: " + invitationURL + "\r\n"
	var auth smtp.Auth
	if s.smtpUsername != "" {
		auth = smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)
	}
	return smtp.SendMail(address, auth, from, []string{recipient}, []byte(message))
}

func (s *Service) upsertUserAndAcceptInvitation(ctx context.Context, identity claims, familyName, invitationToken string) (uuid.UUID, error) {
	// The invitation is consumed in the same transaction as the User upsert and
	// membership link, so a token cannot be accepted twice under concurrency.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	userID := uuid.New()
	if err := tx.QueryRow(ctx, `INSERT INTO users (id, provider_subject, email, name, avatar_url) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (provider_subject) DO UPDATE SET email = EXCLUDED.email, name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url, updated_at = NOW() RETURNING id`, userID, identity.Subject, identity.Email, identity.Name, nullableString(identity.Picture)).Scan(&userID); err != nil {
		return uuid.Nil, err
	}
	var familyID, memberID uuid.UUID
	var invitationEmail string
	if err := tx.QueryRow(ctx, `SELECT family_id, member_id, email FROM invitations WHERE token_hash = $1 AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > NOW() FOR UPDATE`, hashToken(invitationToken)).Scan(&familyID, &memberID, &invitationEmail); err != nil {
		return uuid.Nil, errors.New("invitation is invalid or expired")
	}
	if !strings.EqualFold(invitationEmail, identity.Email) {
		return uuid.Nil, errors.New("invitation email does not match authenticated email")
	}
	var linkedUser *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT user_id FROM family_members WHERE id = $1 AND family_id = $2 FOR UPDATE`, memberID, familyID).Scan(&linkedUser); err != nil || linkedUser != nil && *linkedUser != userID {
		return uuid.Nil, errors.New("member is unavailable")
	}
	if _, err := tx.Exec(ctx, `UPDATE family_members SET user_id = $1 WHERE id = $2 AND family_id = $3`, userID, memberID, familyID); err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE invitations SET accepted_at = NOW() WHERE token_hash = $1`, hashToken(invitationToken)); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func validEmail(value string) bool {
	parsed, err := mail.ParseAddress(strings.TrimSpace(value))
	return err == nil && parsed.Address == strings.TrimSpace(value)
}
