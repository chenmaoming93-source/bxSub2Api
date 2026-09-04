package tokenstat

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

var defaultQuotaChecker atomic.Pointer[QuotaChecker]
var defaultQuotaTimeoutNanos atomic.Int64
var defaultQuotaSingleTimeoutNanos atomic.Int64

func SetDefaultQuotaChecker(checker *QuotaChecker) { defaultQuotaChecker.Store(checker) }
func SetDefaultQuotaTimeout(timeout time.Duration) { defaultQuotaTimeoutNanos.Store(int64(timeout)) }

// SetDefaultQuotaSingleTimeout configures the per-rule timeout for a single
// quota counter read. A value <= 0 disables the per-rule limit and leaves every
// rule read bounded only by the shared total timeout.
func SetDefaultQuotaSingleTimeout(timeout time.Duration) {
	defaultQuotaSingleTimeoutNanos.Store(int64(timeout))
}

func CheckDefaultQuota(ctx context.Context, at time.Time, available map[DimensionCode]DimensionValue) []QuotaDecision {
	if !RuntimeEnabled() {
		return nil
	}
	checker := defaultQuotaChecker.Load()
	if checker == nil {
		return nil
	}
	timeout := time.Duration(defaultQuotaTimeoutNanos.Load())
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	readCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return checker.Check(readCtx, at, available)
}

func HasEnforcedDecision(decisions []QuotaDecision) bool {
	for _, decision := range decisions {
		if decision.Enforced {
			return true
		}
	}
	return false
}

type QuotaMode string

const (
	QuotaModeObserve QuotaMode = "OBSERVE"
	QuotaModeEnforce QuotaMode = "ENFORCE"
)

type QuotaRule struct {
	ID              int64
	ProjectionID    int64
	DimensionCodes  []DimensionCode
	DimensionValues map[DimensionCode]DimensionValue
	MetricCode      MetricCode
	PeriodType      PeriodType
	LimitValue      int64
	Mode            QuotaMode
	EffectiveFrom   *time.Time
	EffectiveUntil  *time.Time
}

type QuotaCounterReader interface {
	Read(ctx context.Context, key, field string) (int64, error)
}

type QuotaDecision struct {
	RuleID   int64
	Used     int64
	Limit    int64
	Exceeded bool
	Enforced bool
}

type QuotaChecker struct {
	reader     QuotaCounterReader
	shardCount int
	rules      atomic.Value
}

func NewQuotaChecker(reader QuotaCounterReader, shardCount int) *QuotaChecker {
	checker := &QuotaChecker{reader: reader, shardCount: shardCount}
	checker.rules.Store([]QuotaRule{})
	return checker
}

func (c *QuotaChecker) ReplaceRules(rules []QuotaRule) {
	c.rules.Store(append([]QuotaRule(nil), rules...))
}

// ActiveRules returns a snapshot of the currently enabled quota rules.
// Callers must treat the returned rules as read-only.
func (c *QuotaChecker) ActiveRules() []QuotaRule {
	if c == nil {
		return nil
	}
	rules, _ := c.rules.Load().([]QuotaRule)
	return append([]QuotaRule(nil), rules...)
}

// MinEnforcedQuotaLimit returns the smallest enabled ENFORCE limit matching
// the supplied request dimensions and natural period. Current usage is not
// considered; the result is the threshold at which the request is blocked.
func MinEnforcedQuotaLimit(at time.Time, periodType PeriodType, metric MetricCode, rules []QuotaRule, available map[DimensionCode]DimensionValue) (int64, bool) {
	var minimum int64
	found := false
	for _, rule := range rules {
		if rule.Mode != QuotaModeEnforce || rule.MetricCode != metric || rule.PeriodType != periodType ||
			!ruleEffective(rule, at) || !ruleMatches(rule, available) {
			continue
		}
		if !found || rule.LimitValue < minimum {
			minimum = rule.LimitValue
			found = true
		}
	}
	return minimum, found
}

// Check is fail-open: only a successful Redis read that proves an ENFORCE
// rule is exceeded returns Enforced=true.
func (c *QuotaChecker) Check(ctx context.Context, at time.Time, available map[DimensionCode]DimensionValue) []QuotaDecision {
	if c == nil || c.reader == nil || c.shardCount <= 0 {
		return nil
	}
	rules, _ := c.rules.Load().([]QuotaRule)
	observability.quotaChecks.Add(1)
	periods := NaturalPeriods(at, at.Location())
	periodByType := make(map[PeriodType]Period, len(periods))
	for _, period := range periods {
		periodByType[period.Type] = period
	}
	decisions := make([]QuotaDecision, 0, len(rules))
	type cachedRead struct {
		used int64
		err  error
	}
	reads := make(map[string]cachedRead)
	for _, rule := range rules {
		if !ruleEffective(rule, at) || !ruleMatches(rule, available) {
			continue
		}
		lookupValues := make(map[DimensionCode]DimensionValue, len(rule.DimensionCodes))
		for _, code := range rule.DimensionCodes {
			lookupValues[code] = available[code]
		}
		identity, err := BuildDimensionIdentity(rule.DimensionCodes, lookupValues)
		if err != nil {
			continue
		}
		period, ok := periodByType[rule.PeriodType]
		if !ok {
			continue
		}
		shard := int(binary.BigEndian.Uint16(identity.Hash[:2])) % c.shardCount
		key := fmt.Sprintf("sub2api:dynamic_token_stats:v1:%s:%s:%d:%d",
			period.Type, period.Start.Format("20060102T150405-0700"), rule.ProjectionID, shard)
		field := hex.EncodeToString(identity.Hash[:]) + ":" + string(rule.MetricCode)
		readStartedAt := time.Now()
		readKey := key + "\x00" + field
		cached, exists := reads[readKey]
		if !exists {
			// Each rule read gets its own per-rule deadline derived from the
			// shared total check context, so a single rule is bounded by
			// min(single_quota_check_timeout, remaining total budget).
			readCtx := ctx
			if singleTimeout := time.Duration(defaultQuotaSingleTimeoutNanos.Load()); singleTimeout > 0 {
				var cancel context.CancelFunc
				readCtx, cancel = context.WithTimeout(ctx, singleTimeout)
				cached.used, cached.err = c.reader.Read(readCtx, key, field)
				cancel()
			} else {
				cached.used, cached.err = c.reader.Read(ctx, key, field)
			}
			reads[readKey] = cached
		}
		used, err := cached.used, cached.err
		if err != nil {
			observability.quotaFailOpen.Add(1)
			slog.WarnContext(ctx, "dynamic token quota redis read failed; request allowed by fail-open policy",
				"rule_id", rule.ID,
				"projection_id", rule.ProjectionID,
				"period_type", rule.PeriodType,
				"redis_key", key,
				"redis_field", field,
				"elapsed_ms", time.Since(readStartedAt).Milliseconds(),
				"error", err,
				"context_error", ctx.Err(),
				"action", "fail_open",
			)
			continue
		}
		exceeded := used >= rule.LimitValue
		if exceeded {
			observability.quotaExceeded.Add(1)
		}
		decisions = append(decisions, QuotaDecision{
			RuleID: rule.ID, Used: used, Limit: rule.LimitValue, Exceeded: exceeded,
			Enforced: exceeded && rule.Mode == QuotaModeEnforce,
		})
	}
	return decisions
}

func ruleEffective(rule QuotaRule, at time.Time) bool {
	return (rule.EffectiveFrom == nil || !at.Before(*rule.EffectiveFrom)) &&
		(rule.EffectiveUntil == nil || at.Before(*rule.EffectiveUntil))
}

func ruleMatches(rule QuotaRule, available map[DimensionCode]DimensionValue) bool {
	for _, code := range rule.DimensionCodes {
		expected, ok := rule.DimensionValues[code]
		if !ok {
			return false
		}
		actual, ok := available[code]
		if !ok {
			return false
		}
		if expected.Type == ValueTypeWildcard {
			continue
		}
		if actual != expected {
			return false
		}
	}
	return true
}
