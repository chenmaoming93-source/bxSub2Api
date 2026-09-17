package tokenstat

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/redis/go-redis/v9"
)

// CurrentDirtyIdentityReader performs one complete cursor traversal of the
// current dirty set. It deliberately never inspects processing sets or locks
// the synchronizer.
type CurrentDirtyIdentityReader struct{ redis *redis.Client }

func NewCurrentDirtyIdentityReader(client *redis.Client) *CurrentDirtyIdentityReader {
	return &CurrentDirtyIdentityReader{redis: client}
}

func (r *CurrentDirtyIdentityReader) ScanCurrent(ctx context.Context, visit func(domain.StatisticIdentity) error) error {
	if r == nil || r.redis == nil {
		return fmt.Errorf("dynamic token statistics redis client is required")
	}
	var cursor uint64
	for {
		members, next, err := r.redis.SScan(ctx, dynamicDirtyKey, cursor, "", 256).Result()
		if err != nil {
			return err
		}
		for _, member := range members {
			identity, decodeErr := decodeStatisticIdentity([]byte(member))
			if decodeErr != nil {
				slog.WarnContext(ctx, "skip invalid dynamic token dirty identity during quota reset discovery", "error", decodeErr)
				continue
			}
			if err := visit(identity); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

type dirtyIdentityEnvelope struct {
	PeriodType      domain.PeriodType `json:"period_type"`
	PeriodStart     time.Time         `json:"period_start"`
	PeriodEnd       time.Time         `json:"period_end"`
	ProjectionID    int64             `json:"projection_id"`
	DimensionHash   string            `json:"dimension_hash"`
	DimensionValues map[string]any    `json:"dimension_values"`
	MetricCode      domain.MetricCode `json:"metric_code"`
}

func decodeStatisticIdentity(encoded []byte) (domain.StatisticIdentity, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var envelope dirtyIdentityEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return domain.StatisticIdentity{}, err
	}
	hashBytes, err := hex.DecodeString(envelope.DimensionHash)
	if err != nil || len(hashBytes) != 16 {
		return domain.StatisticIdentity{}, fmt.Errorf("invalid dimension hash")
	}
	values, err := normalizeDimensionValues(envelope.DimensionValues)
	if err != nil {
		return domain.StatisticIdentity{}, err
	}
	var hash [16]byte
	copy(hash[:], hashBytes)
	return domain.StatisticIdentity{
		Period:       domain.Period{Type: envelope.PeriodType, Start: envelope.PeriodStart, End: envelope.PeriodEnd},
		ProjectionID: envelope.ProjectionID, DimensionHash: hash, DimensionValues: values, MetricCode: envelope.MetricCode,
	}, nil
}

func normalizeDimensionValues(raw map[string]any) (map[domain.DimensionCode]domain.DimensionValue, error) {
	result := make(map[domain.DimensionCode]domain.DimensionValue, len(raw))
	for rawCode, rawValue := range raw {
		code := domain.DimensionCode(rawCode)
		definition, ok := domain.Dimension(code)
		if !ok {
			return nil, fmt.Errorf("unknown dimension %q", rawCode)
		}
		switch definition.ValueType {
		case domain.ValueTypeInt64:
			var value int64
			switch typed := rawValue.(type) {
			case json.Number:
				parsed, err := typed.Int64()
				if err != nil {
					return nil, fmt.Errorf("invalid int64 dimension %q: %w", code, err)
				}
				value = parsed
			case int64:
				value = typed
			case int:
				value = int64(typed)
			case float64:
				value = int64(typed)
				if float64(value) != typed {
					return nil, fmt.Errorf("invalid int64 dimension %q", code)
				}
			default:
				return nil, fmt.Errorf("invalid int64 dimension %q", code)
			}
			result[code] = domain.Int64Value(value)
		case domain.ValueTypeString:
			value, ok := rawValue.(string)
			if !ok {
				return nil, fmt.Errorf("invalid string dimension %q", code)
			}
			result[code] = domain.StringValue(value)
		default:
			return nil, fmt.Errorf("unsupported dimension %q", code)
		}
	}
	return result, nil
}

// ListAggregateIdentities returns one keyset page of current-period aggregate
// identities. The caller owns the pagination loop so discovery invokes this
// source only after its single dirty-set traversal.
func (r *Repository) ListAggregateIdentities(ctx context.Context, query domain.AggregateIdentityQuery) ([]domain.StatisticIdentity, *domain.AggregateIdentityCursor, error) {
	if r == nil || r.db == nil {
		return nil, nil, fmt.Errorf("tokenstat repository database is required")
	}
	if len(query.ProjectionIDs) == 0 || query.Limit <= 0 {
		return nil, nil, nil
	}

	projectionIDs := append([]int64(nil), query.ProjectionIDs...)
	sort.Slice(projectionIDs, func(i, j int) bool { return projectionIDs[i] < projectionIDs[j] })
	placeholders := make([]string, len(projectionIDs))
	args := make([]any, 0, 5+len(projectionIDs)+len(query.Filters)*2)
	args = append(args, string(query.Period.Type), query.Period.Start, string(query.MetricCode))
	for i, projectionID := range projectionIDs {
		placeholders[i] = "?"
		args = append(args, projectionID)
	}

	builder := strings.Builder{}
	builder.WriteString(`SELECT period_type, period_start, period_end, projection_id, dimension_hash, dimension_values, metric_code
FROM token_stat_aggregates
WHERE period_type = ? AND period_start = ? AND metric_code = ? AND projection_id IN (`)
	builder.WriteString(strings.Join(placeholders, ","))
	builder.WriteString(")")

	codes := make([]domain.DimensionCode, 0, len(query.Filters))
	for code := range query.Filters {
		codes = append(codes, code)
	}
	canonical, err := domain.CanonicalDimensionCodes(codes)
	if err != nil {
		return nil, nil, err
	}
	for _, code := range canonical {
		column, ok := aggregateDimensionColumn(code)
		if !ok {
			return nil, nil, fmt.Errorf("dimension %q is not queryable", code)
		}
		builder.WriteString(" AND ")
		builder.WriteString(column)
		builder.WriteString(" = ?")
		value := query.Filters[code]
		if value.Type == domain.ValueTypeInt64 {
			args = append(args, value.Int64)
		} else {
			args = append(args, value.String)
		}
	}
	if query.After != nil {
		builder.WriteString(" AND (projection_id > ? OR (projection_id = ? AND dimension_hash > ?))")
		args = append(args, query.After.ProjectionID, query.After.ProjectionID, query.After.DimensionHash[:])
	}
	builder.WriteString(" ORDER BY projection_id ASC, dimension_hash ASC LIMIT ?")
	args = append(args, query.Limit)

	rows, err := r.db.QueryContext(ctx, builder.String(), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	identities := make([]domain.StatisticIdentity, 0, query.Limit)
	for rows.Next() {
		var periodType string
		var periodStart, periodEnd time.Time
		var projectionID int64
		var hashBytes, encodedValues []byte
		var metricCode string
		if err := rows.Scan(&periodType, &periodStart, &periodEnd, &projectionID, &hashBytes, &encodedValues, &metricCode); err != nil {
			return nil, nil, err
		}
		if len(hashBytes) != 16 {
			return nil, nil, fmt.Errorf("invalid aggregate dimension hash")
		}
		decoder := json.NewDecoder(bytes.NewReader(encodedValues))
		decoder.UseNumber()
		var rawValues map[string]any
		if err := decoder.Decode(&rawValues); err != nil {
			return nil, nil, err
		}
		values, err := normalizeDimensionValues(rawValues)
		if err != nil {
			return nil, nil, err
		}
		var hash [16]byte
		copy(hash[:], hashBytes)
		identities = append(identities, domain.StatisticIdentity{
			Period:       domain.Period{Type: domain.PeriodType(periodType), Start: periodStart, End: periodEnd},
			ProjectionID: projectionID, DimensionHash: hash, DimensionValues: values, MetricCode: domain.MetricCode(metricCode),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(identities) < query.Limit {
		return identities, nil, nil
	}
	last := identities[len(identities)-1]
	return identities, &domain.AggregateIdentityCursor{ProjectionID: last.ProjectionID, DimensionHash: last.DimensionHash}, nil
}

func aggregateDimensionColumn(code domain.DimensionCode) (string, bool) {
	switch code {
	case domain.DimensionUserID:
		return "user_id", true
	case domain.DimensionAPIKeyID:
		return "api_key_id", true
	case domain.DimensionGroupID:
		return "group_id", true
	case domain.DimensionRouteAlias:
		return "route_alias", true
	case domain.DimensionAccountID:
		return "account_id", true
	case domain.DimensionUpstreamModel:
		return "upstream_model", true
	case domain.DimensionDepartment:
		return "department", true
	default:
		return "", false
	}
}
