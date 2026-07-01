package handler

func containsAccountID(ids []int64, accountID int64) bool {
	for _, id := range ids {
		if id == accountID {
			return true
		}
	}
	return false
}
