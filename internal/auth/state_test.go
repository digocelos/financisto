package auth

import "testing"

func TestOAuthStateRoundTripPreservesFamilyName(t *testing.T) {
	state, err := NewOAuthState("Família Vasconcelos")
	if err != nil {
		t.Fatalf("NewOAuthState() error = %v", err)
	}

	decoded, err := ParseOAuthState(state)
	if err != nil {
		t.Fatalf("ParseOAuthState() error = %v", err)
	}
	if decoded.FamilyName != "Família Vasconcelos" {
		t.Fatalf("FamilyName = %q", decoded.FamilyName)
	}
	if decoded.Nonce == "" {
		t.Fatal("Nonce should be generated")
	}
}

func TestParseOAuthStateRejectsMalformedState(t *testing.T) {
	if _, err := ParseOAuthState("not-a-state"); err == nil {
		t.Fatal("ParseOAuthState() error = nil")
	}
}
