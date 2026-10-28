package cards

import "time"

type installment struct {
	Number       int
	AmountCents  int64
	InvoiceMonth string
}

func buildInstallments(principal int64, count int, installmentAmount int64, purchasedAt time.Time, closingDay int) ([]installment, int64, int64) {
	installments := make([]installment, count)
	total := installmentAmount * int64(count)
	for index := range installments {
		date := purchasedAt
		if purchasedAt.Day() > closingDay {
			date = date.AddDate(0, 1, 0)
		}
		date = date.AddDate(0, index, 0)
		installments[index] = installment{Number: index + 1, AmountCents: installmentAmount, InvoiceMonth: date.Format("2006-01")}
	}
	return installments, total, total - principal
}
