package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ConcurrencySampleKindAccount = "account"
	ConcurrencySampleKindRoute   = "route"
)

type ConcurrencySample struct {
	Kind       string
	AccountID  int64
	GroupID    int64
	RouteAlias string
	Current    int
	Limit      int
	Waiting    int
	SampledAt  time.Time
	Available  bool
}

type ConcurrencyAggregate struct {
	WindowSeconds     int
	BucketStart       time.Time
	SampleKind        string
	DimensionHash     string
	AccountID         int64
	GroupID           int64
	RouteAlias        string
	CurrentPeak       int
	CurrentSum        int64
	SampleCount       int64
	EffectiveLimit    int
	WaitingPeak       int
	PeakRatioActual   int
	PeakRatioLimit    int
	PeakRatioSampleAt time.Time
	WaitSumMS         int64
	WaitSampleCount   int64
	FirstSampledAt    time.Time
	LastSampledAt     time.Time
}

// AddWait records one completed account queue wait. It shares the account row
// with periodic concurrency samples but does not alter current-concurrency
// sample counts.
func (a *ConcurrencyAccumulator) AddWait(sampledAt time.Time, accountID int64, wait time.Duration) ([]ConcurrencyAggregate, int) {
	if a == nil || accountID <= 0 {
		return nil, 0
	}
	if sampledAt.IsZero() {
		sampledAt = time.Now().UTC()
	} else {
		sampledAt = sampledAt.UTC()
	}
	minute := sampledAt.Truncate(time.Minute)
	var flushed []ConcurrencyAggregate
	if a.minute.IsZero() {
		a.minute = minute
	} else if minute.After(a.minute) {
		flushed = a.Flush()
		a.minute = minute
	}
	sample := ConcurrencySample{Kind: ConcurrencySampleKindAccount, AccountID: accountID}
	key := concurrencySampleKey(sample)
	row := a.rows[key]
	if row == nil {
		row = &ConcurrencyAggregate{
			WindowSeconds: 60, BucketStart: minute, SampleKind: sample.Kind,
			DimensionHash: ConcurrencyDimensionHash(sample), AccountID: accountID,
			FirstSampledAt: sampledAt,
		}
		a.rows[key] = row
	}
	waitMS := wait.Milliseconds()
	if waitMS < 1 {
		waitMS = 1
	}
	row.WaitSumMS += waitMS
	row.WaitSampleCount++
	row.LastSampledAt = sampledAt
	return flushed, len(a.rows)
}

type ConcurrencyAccumulator struct {
	minute time.Time
	rows   map[string]*ConcurrencyAggregate
}

func NewConcurrencyAccumulator() *ConcurrencyAccumulator {
	return &ConcurrencyAccumulator{rows: make(map[string]*ConcurrencyAggregate)}
}

// Add aggregates one Redis snapshot. It only returns rows when crossing a UTC
// minute; snapshots inside the same minute never create database writes.
func (a *ConcurrencyAccumulator) Add(sampledAt time.Time, samples []ConcurrencySample) ([]ConcurrencyAggregate, int) {
	if a == nil {
		return nil, 0
	}
	if sampledAt.IsZero() {
		sampledAt = time.Now().UTC()
	} else {
		sampledAt = sampledAt.UTC()
	}
	minute := sampledAt.Truncate(time.Minute)
	var flushed []ConcurrencyAggregate
	if a.minute.IsZero() {
		a.minute = minute
	} else if minute.After(a.minute) {
		flushed = a.Flush()
		a.minute = minute
	}
	for _, sample := range samples {
		if !sample.Available {
			continue
		}
		sample.RouteAlias = strings.TrimSpace(sample.RouteAlias)
		key := concurrencySampleKey(sample)
		row := a.rows[key]
		if row == nil {
			row = &ConcurrencyAggregate{
				WindowSeconds:  60,
				BucketStart:    minute,
				SampleKind:     sample.Kind,
				DimensionHash:  ConcurrencyDimensionHash(sample),
				AccountID:      sample.AccountID,
				GroupID:        sample.GroupID,
				RouteAlias:     sample.RouteAlias,
				FirstSampledAt: sampledAt,
			}
			a.rows[key] = row
		}
		if sample.Current > row.CurrentPeak {
			row.CurrentPeak = sample.Current
		}
		if sample.Waiting > row.WaitingPeak {
			row.WaitingPeak = sample.Waiting
		}
		if sample.Limit > row.EffectiveLimit {
			row.EffectiveLimit = sample.Limit
		}
		if sample.Limit > 0 {
			currentRatio := float64(sample.Current) / float64(sample.Limit)
			peakRatio := 0.0
			if row.PeakRatioLimit > 0 {
				peakRatio = float64(row.PeakRatioActual) / float64(row.PeakRatioLimit)
			}
			if currentRatio > peakRatio || row.PeakRatioSampleAt.IsZero() {
				row.PeakRatioActual = sample.Current
				row.PeakRatioLimit = sample.Limit
				row.PeakRatioSampleAt = sampledAt
			}
		}
		row.CurrentSum += int64(sample.Current)
		row.SampleCount++
		row.LastSampledAt = sampledAt
	}
	return flushed, len(a.rows)
}

func (a *ConcurrencyAccumulator) Flush() []ConcurrencyAggregate {
	if a == nil || len(a.rows) == 0 {
		return nil
	}
	out := make([]ConcurrencyAggregate, 0, len(a.rows))
	for _, row := range a.rows {
		row.BucketStart = a.minute
		out = append(out, *row)
	}
	a.rows = make(map[string]*ConcurrencyAggregate)
	sort.Slice(out, func(i, j int) bool { return out[i].DimensionHash < out[j].DimensionHash })
	return out
}

func ConcurrencyDimensionHash(sample ConcurrencySample) string {
	sum := sha256.Sum256([]byte(concurrencySampleKey(sample)))
	return hex.EncodeToString(sum[:])
}

func concurrencySampleKey(sample ConcurrencySample) string {
	return sample.Kind + "|" + strconv.FormatInt(sample.AccountID, 10) + "|" +
		strconv.FormatInt(sample.GroupID, 10) + "|" + strings.TrimSpace(sample.RouteAlias)
}
