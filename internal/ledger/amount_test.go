package ledger

import "testing"

func TestEntryAmountForOperationType(t *testing.T) {
	tests := []struct {
		typeName string
		amount   int64
		want     int64
	}{
		{typeName: "INCOME", amount: 1000, want: 1000},
		{typeName: "EXPENSE", amount: 1000, want: -1000},
		{typeName: "ADJUSTMENT", amount: 1000, want: 1000},
	}
	for _, tt := range tests {
		if got := signedAmount(tt.typeName, tt.amount); got != tt.want {
			t.Errorf("signedAmount(%q, %d) = %d, want %d", tt.typeName, tt.amount, got, tt.want)
		}
	}
}
