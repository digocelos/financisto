package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type OAuthState struct {
	Nonce           string `json:"nonce"`
	FamilyName      string `json:"family_name"`
	InvitationToken string `json:"invitation_token,omitempty"`
}

func NewOAuthState(familyName, invitationToken string) (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	payload, err := json.Marshal(OAuthState{
		Nonce:           base64.RawURLEncoding.EncodeToString(nonce),
		FamilyName:      familyName,
		InvitationToken: invitationToken,
	})
	if err != nil {
		return "", fmt.Errorf("encode oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func ParseOAuthState(value string) (OAuthState, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return OAuthState{}, fmt.Errorf("decode oauth state: %w", err)
	}
	var state OAuthState
	if err := json.Unmarshal(payload, &state); err != nil {
		return OAuthState{}, fmt.Errorf("parse oauth state: %w", err)
	}
	if state.Nonce == "" {
		return OAuthState{}, fmt.Errorf("oauth state nonce is empty")
	}
	return state, nil
}
