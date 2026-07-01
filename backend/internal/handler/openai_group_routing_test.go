package handler

import "testing"

func TestContainsAccountID(t *testing.T) {
	if !containsAccountID([]int64{3, 7}, 3) || containsAccountID([]int64{3, 7}, 4) {
		t.Fatal("containsAccountID returned an unexpected result")
	}
}
