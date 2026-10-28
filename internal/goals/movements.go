package goals

func signedMovement(kind string, amount int64) int64 {
	if kind == "WITHDRAWAL" { return -amount }
	return amount
}
