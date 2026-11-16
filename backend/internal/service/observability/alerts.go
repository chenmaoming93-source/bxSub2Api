package observability

import (
	"math"
	"time"
)

func RequiredAlertEvaluations(sustainedMinutes int, interval time.Duration, requiredMinimum int) int {
	if requiredMinimum < 1 {
		requiredMinimum = 1
	}
	if sustainedMinutes <= 0 {
		return requiredMinimum
	}
	required := 1
	if interval > 0 {
		required = int(math.Ceil(float64(sustainedMinutes*60) / interval.Seconds()))
	} else {
		required = sustainedMinutes
	}
	if required < requiredMinimum {
		required = requiredMinimum
	}
	return required
}
