package observability

type Histogram struct {
	SampleCount int64    `json:"sample_count"`
	SumMS       int64    `json:"sum_ms"`
	MinMS       int64    `json:"min_ms"`
	MaxMS       int64    `json:"max_ms"`
	Buckets     [8]int64 `json:"buckets"`
}

// Quantiles returns interpolated P50/P95 from fixed upper bounds. Bucket
// indexes are 100,250,500,1000,2000,5000,10000 and +Inf milliseconds.
func (h Histogram) Quantiles(quantiles ...float64) map[float64]int64 {
	out := make(map[float64]int64, len(quantiles))
	if h.SampleCount <= 0 {
		return out
	}
	upper := [...]int64{100, 250, 500, 1000, 2000, 5000, 10000}
	for _, quantile := range quantiles {
		if quantile <= 0 {
			out[quantile] = 0
			continue
		}
		if quantile > 1 {
			quantile = 1
		}
		rank := int64(float64(h.SampleCount)*quantile + 0.999999)
		if rank < 1 {
			rank = 1
		}
		var cumulative int64
		value := h.MaxMS
		lower := int64(0)
		for index, count := range h.Buckets {
			if count <= 0 || cumulative+count < rank {
				cumulative += count
				if index < len(upper) {
					lower = upper[index]
				}
				continue
			}
			if index < len(upper) {
				fraction := float64(rank-cumulative) / float64(count)
				value = lower + int64(float64(upper[index]-lower)*fraction)
				if value < lower {
					value = lower
				}
				if value > upper[index] {
					value = upper[index]
				}
			} else {
				value = h.MaxMS
			}
			break
		}
		if value < h.MinMS && h.MinMS > 0 {
			value = h.MinMS
		}
		out[quantile] = value
	}
	return out
}
