package cards

import (
	"testing"
	"time"
)

func TestBuildInstallmentsAssignsPurchaseToInvoices(t *testing.T) {
	installments, total, interest := buildInstallments(10000, 3, 3600, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), 10)

	if total != 10800 || interest != 800 {
		t.Fatalf("total=%d interest=%d, want 10800 and 800", total, interest)
	}
	if len(installments) != 3 {
		t.Fatalf("installments=%d, want 3", len(installments))
	}
	if installments[0].InvoiceMonth != "2026-10" || installments[1].InvoiceMonth != "2026-11" || installments[2].InvoiceMonth != "2026-12" {
		t.Fatalf("invoice months = %#v", installments)
	}
}
