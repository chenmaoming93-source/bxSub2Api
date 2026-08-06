package tokenstat

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/tokenstatprojection"
	"github.com/Wei-Shaw/sub2api/ent/tokenstatquotarule"
	"github.com/redis/go-redis/v9"
)

const (
	QuotaStatusPending  = "PENDING"
	QuotaStatusEnabled  = "ENABLED"
	QuotaStatusDisabled = "DISABLED"
	quotaConfigChannel  = "sub2api:dynamic_token_stats_quota:v1:changed"
)

type QuotaInput struct {
	Name            string
	DimensionCodes  []DimensionCode
	DimensionValues map[DimensionCode]DimensionValue
	MetricCode      MetricCode
	PeriodType      PeriodType
	LimitValue      int64
	Mode            QuotaMode
	CreatedBy       uint64
}

func (s *ProjectionAdminService) ListQuotas(ctx context.Context) ([]*ent.TokenStatQuotaRule, error) {
	return s.client.TokenStatQuotaRule.Query().Order(ent.Asc(tokenstatquotarule.FieldID)).All(ctx)
}

func (s *ProjectionAdminService) CreateQuota(ctx context.Context, input QuotaInput) (*ent.TokenStatQuotaRule, error) {
	if strings.TrimSpace(input.Name) == "" || input.LimitValue <= 0 {
		return nil, fmt.Errorf("quota name and positive limit_value are required")
	}
	metric, ok := Metric(input.MetricCode)
	if !ok || !metric.AllowQuota {
		return nil, fmt.Errorf("metric %q does not support quotas", input.MetricCode)
	}
	if input.PeriodType != PeriodDay && input.PeriodType != PeriodWeek && input.PeriodType != PeriodMonth {
		return nil, fmt.Errorf("invalid period type")
	}
	if input.Mode != QuotaModeObserve && input.Mode != QuotaModeEnforce {
		return nil, fmt.Errorf("invalid quota mode")
	}
	canonical, err := CanonicalDimensionCodes(input.DimensionCodes)
	if err != nil {
		return nil, err
	}
	for _, code := range canonical {
		definition, _ := Dimension(code)
		value, exists := input.DimensionValues[code]
		if !exists {
			return nil, fmt.Errorf("missing dimension %q", code)
		}
		if err := validateQuotaDimensionValue(definition, value); err != nil {
			return nil, err
		}
	}
	identityHash, err := quotaRuleDimensionHash(canonical, input.DimensionValues)
	if err != nil {
		return nil, err
	}
	signature, _ := DimensionSignature(canonical)
	projection, err := s.client.TokenStatProjection.Query().
		Where(tokenstatprojection.DimensionSignatureEQ(signature)).Only(ctx)
	status := QuotaStatusEnabled
	if ent.IsNotFound(err) {
		projection, err = s.Create(ctx, ProjectionInput{
			Name: "Auto: " + signature, DimensionCodes: canonical,
			MetricCodes: []MetricCode{input.MetricCode}, CreatedBy: input.CreatedBy,
		})
		status = QuotaStatusPending
	}
	if err != nil {
		return nil, err
	}
	if projection.Status != ProjectionStatusActive {
		status = QuotaStatusPending
	}
	raw := make(map[string]any, len(input.DimensionValues))
	for code, value := range input.DimensionValues {
		if value.Type == ValueTypeWildcard {
			raw[string(code)] = map[string]any{"type": string(ValueTypeWildcard)}
		} else if value.Type == ValueTypeInt64 {
			raw[string(code)] = value.Int64
		} else {
			raw[string(code)] = value.String
		}
	}
	// New rules apply to the current natural period immediately. The counter
	// already accumulated in Redis is therefore included in the first check.
	effectiveFrom := time.Now()
	rule, err := s.client.TokenStatQuotaRule.Create().
		SetName(strings.TrimSpace(input.Name)).SetProjectionID(projection.ID).
		SetDimensionHash(identityHash[:]).SetDimensionValues(raw).
		SetMetricCode(string(input.MetricCode)).SetPeriodType(string(input.PeriodType)).
		SetLimitValue(input.LimitValue).SetEnforcementMode(string(input.Mode)).
		SetStatus(status).SetEffectiveFrom(effectiveFrom).SetCreatedBy(input.CreatedBy).Save(ctx)
	if err != nil {
		return nil, err
	}
	if status == QuotaStatusEnabled {
		_ = s.RefreshQuotaRules(ctx)
	}
	return rule, nil
}

func (s *ProjectionAdminService) enablePendingQuotas(ctx context.Context, projectionID int64) error {
	_, err := s.client.TokenStatQuotaRule.Update().
		Where(
			tokenstatquotarule.ProjectionIDEQ(projectionID),
			tokenstatquotarule.StatusEQ(QuotaStatusPending),
		).
		SetStatus(QuotaStatusEnabled).
		Save(ctx)
	if err != nil {
		return err
	}
	return s.RefreshQuotaRules(ctx)
}

func (s *ProjectionAdminService) SetQuotaStatus(ctx context.Context, id int64, enabled bool) (*ent.TokenStatQuotaRule, error) {
	status := QuotaStatusDisabled
	if enabled {
		rule, err := s.client.TokenStatQuotaRule.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		projection, err := s.client.TokenStatProjection.Get(ctx, rule.ProjectionID)
		if err != nil || projection.Status != ProjectionStatusActive {
			return nil, fmt.Errorf("quota projection is not active")
		}
		status = QuotaStatusEnabled
	}
	update := s.client.TokenStatQuotaRule.UpdateOneID(id).SetStatus(status)
	if enabled {
		// Re-enabling a rule is an explicit request to enforce it now, including
		// usage already accumulated in the current natural period.
		update.SetEffectiveFrom(time.Now())
	}
	rule, err := update.Save(ctx)
	if err == nil {
		err = s.RefreshQuotaRules(ctx)
	}
	return rule, err
}

func (s *ProjectionAdminService) UpdateQuota(ctx context.Context, id int64, input QuotaInput) (*ent.TokenStatQuotaRule, error) {
	if strings.TrimSpace(input.Name) == "" || input.LimitValue <= 0 {
		return nil, fmt.Errorf("quota name and positive limit_value are required")
	}
	if input.Mode != QuotaModeObserve && input.Mode != QuotaModeEnforce {
		return nil, fmt.Errorf("invalid quota mode")
	}
	rule, err := s.client.TokenStatQuotaRule.UpdateOneID(id).
		SetName(strings.TrimSpace(input.Name)).SetLimitValue(input.LimitValue).
		SetEnforcementMode(string(input.Mode)).Save(ctx)
	if err == nil {
		err = s.RefreshQuotaRules(ctx)
	}
	return rule, err
}

// DeleteQuota removes only the quota rule. The associated projection, Redis
// usage counters, and persisted aggregate history remain intact.
func (s *ProjectionAdminService) DeleteQuota(ctx context.Context, id int64) error {
	if err := s.client.TokenStatQuotaRule.DeleteOneID(id).Exec(ctx); err != nil {
		return err
	}
	return s.RefreshQuotaRules(ctx)
}

func (s *ProjectionAdminService) RefreshQuotaRules(ctx context.Context) error {
	if s.quotaChecker != nil {
		if err := s.LoadQuotaRules(ctx, s.quotaChecker); err != nil {
			return err
		}
	}
	if s.redis == nil {
		return nil
	}
	rules, err := s.client.TokenStatQuotaRule.Query().Where(tokenstatquotarule.StatusEQ(QuotaStatusEnabled)).All(ctx)
	if err != nil {
		return err
	}
	return s.redis.Publish(ctx, quotaConfigChannel, len(rules)).Err()
}

func (s *ProjectionAdminService) LoadQuotaRules(ctx context.Context, checker *QuotaChecker) error {
	rows, err := s.client.TokenStatQuotaRule.Query().Where(tokenstatquotarule.StatusEQ(QuotaStatusEnabled)).All(ctx)
	if err != nil {
		return err
	}
	rules := make([]QuotaRule, 0, len(rows))
	for _, row := range rows {
		projection, err := s.client.TokenStatProjection.Get(ctx, row.ProjectionID)
		if err != nil {
			return err
		}
		codes := make([]DimensionCode, len(projection.DimensionCodes))
		values := make(map[DimensionCode]DimensionValue, len(codes))
		for i, rawCode := range projection.DimensionCodes {
			code := DimensionCode(rawCode)
			codes[i] = code
			definition, ok := Dimension(code)
			if !ok {
				continue
			}
			raw := row.DimensionValues[rawCode]
			if wildcard, ok := raw.(map[string]any); ok && wildcard["type"] == string(ValueTypeWildcard) {
				values[code] = WildcardValue()
			} else if definition.ValueType == ValueTypeInt64 {
				if number, ok := raw.(float64); ok {
					values[code] = Int64Value(int64(number))
				} else if number, ok := raw.(int64); ok {
					values[code] = Int64Value(number)
				}
			} else if text, ok := raw.(string); ok {
				values[code] = StringValue(text)
			}
		}
		rules = append(rules, QuotaRule{
			ID: row.ID, ProjectionID: row.ProjectionID, DimensionCodes: codes, DimensionValues: values,
			MetricCode: MetricCode(row.MetricCode), PeriodType: PeriodType(row.PeriodType),
			LimitValue: row.LimitValue, Mode: QuotaMode(row.EnforcementMode),
			EffectiveFrom: row.EffectiveFrom, EffectiveUntil: row.EffectiveUntil,
		})
	}
	checker.ReplaceRules(rules)
	return nil
}

func quotaRuleDimensionHash(codes []DimensionCode, values map[DimensionCode]DimensionValue) ([16]byte, error) {
	payload := make([]struct {
		Code  DimensionCode  `json:"code"`
		Value DimensionValue `json:"value"`
	}, 0, len(codes))
	for _, code := range codes {
		payload = append(payload, struct {
			Code  DimensionCode  `json:"code"`
			Value DimensionValue `json:"value"`
		}{code, values[code]})
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return [16]byte{}, err
	}
	sum := sha256.Sum256(encoded)
	var result [16]byte
	copy(result[:], sum[:16])
	return result, nil
}

func (s *ProjectionAdminService) CanDisableProjection(ctx context.Context, projectionID int64) error {
	exists, err := s.client.TokenStatQuotaRule.Query().
		Where(tokenstatquotarule.ProjectionIDEQ(projectionID), tokenstatquotarule.StatusEQ(QuotaStatusEnabled)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("projection is referenced by an enabled quota")
	}
	return nil
}

func nextNaturalPeriodStart(now time.Time, periodType PeriodType) time.Time {
	periods := NaturalPeriods(now, now.Location())
	for _, period := range periods {
		if period.Type == periodType {
			return period.End
		}
	}
	return now
}

type RedisQuotaCounterReader struct{ client *redis.Client }

func NewRedisQuotaCounterReader(client *redis.Client) *RedisQuotaCounterReader {
	return &RedisQuotaCounterReader{client: client}
}

func (r *RedisQuotaCounterReader) Read(ctx context.Context, key, field string) (int64, error) {
	value, err := r.client.HGet(ctx, key, field).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return value, err
}
