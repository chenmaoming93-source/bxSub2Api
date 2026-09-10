package service

import "testing"

func allocationPtr(value int) *int { return &value }

func allocationValues(values []*int) []int {
	result := make([]int, len(values))
	for i, value := range values {
		if value != nil {
			result[i] = *value
		}
	}
	return result
}

func TestScaleModelRouteConcurrencyAllocationsScalesWithLargestRemainder(t *testing.T) {
	got := ScaleModelRouteConcurrencyAllocations(10, 7, []*int{allocationPtr(3), allocationPtr(3), allocationPtr(4)})
	if want := []int{2, 2, 3}; !equalInts(allocationValues(got), want) {
		t.Fatalf("got %v, want %v", allocationValues(got), want)
	}
}

func TestScaleModelRouteConcurrencyAllocationsKeepsUnlimitedCandidates(t *testing.T) {
	got := ScaleModelRouteConcurrencyAllocations(10, 7, []*int{allocationPtr(3), nil, allocationPtr(3)})
	if got[1] != nil || !equalInts(allocationValues(got), []int{2, 0, 2}) {
		t.Fatalf("got %v with unlimited candidate %v", allocationValues(got), got[1])
	}
}

func TestScaleModelRouteConcurrencyAllocationsFromUnlimited(t *testing.T) {
	got := ScaleModelRouteConcurrencyAllocations(0, 10, []*int{allocationPtr(4), allocationPtr(5), nil})
	if !equalInts(allocationValues(got), []int{4, 5, 0}) || got[2] != nil {
		t.Fatalf("got %v with unlimited candidate %v", allocationValues(got), got[2])
	}

	got = ScaleModelRouteConcurrencyAllocations(0, 6, []*int{allocationPtr(4), allocationPtr(5), nil})
	if !equalInts(allocationValues(got), []int{3, 3, 0}) || got[2] != nil {
		t.Fatalf("got %v with unlimited candidate %v", allocationValues(got), got[2])
	}
}

func TestScaleModelRouteConcurrencyAllocationsToUnlimitedKeepsValues(t *testing.T) {
	got := ScaleModelRouteConcurrencyAllocations(10, 0, []*int{allocationPtr(3), allocationPtr(4), nil})
	if !equalInts(allocationValues(got), []int{3, 4, 0}) || got[2] != nil {
		t.Fatalf("got %v with unlimited candidate %v", allocationValues(got), got[2])
	}
}

func TestCandidateConcurrencyShare(t *testing.T) {
	if share, ok := CandidateConcurrencyShare(10, allocationPtr(3)); !ok || share != 30 {
		t.Fatalf("share = %v, ok = %v", share, ok)
	}
	if _, ok := CandidateConcurrencyShare(0, allocationPtr(3)); ok {
		t.Fatal("unlimited account must not have a percentage")
	}
	if _, ok := CandidateConcurrencyShare(10, nil); ok {
		t.Fatal("unlimited candidate must not have a percentage")
	}
}

func TestScaleModelRouteConcurrencyValue(t *testing.T) {
	if got := ScaleModelRouteConcurrencyValue(100, 200, allocationPtr(10)); got == nil || *got != 20 {
		t.Fatalf("got %v, want 20", got)
	}
	if got := ScaleModelRouteConcurrencyValue(100, 50, allocationPtr(50)); got == nil || *got != 25 {
		t.Fatalf("got %v, want 25", got)
	}
	if got := ScaleModelRouteConcurrencyValue(100, 1, allocationPtr(1)); got == nil || *got != 1 {
		t.Fatalf("got %v, want positive minimum 1", got)
	}
	if got := ScaleModelRouteConcurrencyValue(100, 200, nil); got != nil {
		t.Fatalf("unlimited value changed to %v", got)
	}
	if got := ScaleModelRouteConcurrencyValue(0, 200, allocationPtr(10)); got == nil || *got != 10 {
		t.Fatalf("old unlimited value changed to %v", got)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
