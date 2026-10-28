package goals

import "testing"

func TestMovementSignedAmount(t *testing.T) {
	if got := signedMovement("CONTRIBUTION", 1000); got != 1000 { t.Fatalf("contribution = %d", got) }
	if got := signedMovement("WITHDRAWAL", 1000); got != -1000 { t.Fatalf("withdrawal = %d", got) }
	if got := signedMovement("DIVIDEND", 1000); got != 1000 { t.Fatalf("dividend = %d", got) }
}
