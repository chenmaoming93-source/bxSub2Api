package service

import "math"

// ScaleModelRouteConcurrencyAllocations rescales concrete candidate
// allocations when an account concurrency changes. Nil entries represent
// unlimited candidates and are deliberately left untouched.
func ScaleModelRouteConcurrencyAllocations(oldConcurrency, newConcurrency int, allocations []*int) []*int {
	result := make([]*int, len(allocations))
	for i, value := range allocations {
		if value == nil {
			continue
		}
		v := *value
		if v < 0 {
			v = 0
		}
		result[i] = &v
	}

	if newConcurrency <= 0 {
		return result
	}
	if oldConcurrency <= 0 {
		if sumConcreteAllocations(result) <= newConcurrency {
			return result
		}
		return scaleToTarget(result, newConcurrency)
	}

	oldSum := sumConcreteAllocations(result)
	if oldSum == 0 {
		return result
	}
	target := int(math.Round(float64(oldSum) * float64(newConcurrency) / float64(oldConcurrency)))
	if target > newConcurrency {
		target = newConcurrency
	}
	return scaleToTarget(result, target)
}

// CandidateConcurrencyShare returns the display percentage for a concrete
// candidate. The boolean is false when the account is unlimited or the
// candidate is unlimited/unconfigured.
func CandidateConcurrencyShare(accountConcurrency int, candidateConcurrency *int) (float64, bool) {
	if accountConcurrency <= 0 || candidateConcurrency == nil || *candidateConcurrency < 0 {
		return 0, false
	}
	return float64(*candidateConcurrency) * 100 / float64(accountConcurrency), true
}

func sumConcreteAllocations(allocations []*int) int {
	total := 0
	for _, value := range allocations {
		if value != nil && *value > 0 {
			total += *value
		}
	}
	return total
}

func scaleToTarget(allocations []*int, target int) []*int {
	if target <= 0 {
		for i, value := range allocations {
			if value != nil {
				zero := 0
				allocations[i] = &zero
			}
		}
		return allocations
	}

	total := sumConcreteAllocations(allocations)
	if total == 0 {
		return allocations
	}
	type remainder struct {
		index int
		value float64
	}
	remainders := make([]remainder, 0, len(allocations))
	current := 0
	for i, value := range allocations {
		if value == nil {
			continue
		}
		exact := float64(*value) * float64(target) / float64(total)
		base := int(math.Floor(exact))
		allocations[i] = &base
		current += base
		remainders = append(remainders, remainder{index: i, value: exact - float64(base)})
	}

	for i := 1; i < len(remainders); i++ {
		for j := i; j > 0 && remainders[j].value > remainders[j-1].value; j-- {
			remainders[j], remainders[j-1] = remainders[j-1], remainders[j]
		}
	}
	for i := 0; current < target && i < len(remainders); i++ {
		index := remainders[i].index
		value := *allocations[index] + 1
		allocations[index] = &value
		current++
	}
	return allocations
}
