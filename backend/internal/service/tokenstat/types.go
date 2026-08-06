package tokenstat

import (
	"fmt"
	"strings"
	"time"
)

type DimensionCode string

const (
	DimensionUserID        DimensionCode = "user_id"
	DimensionAPIKeyID      DimensionCode = "api_key_id"
	DimensionGroupID       DimensionCode = "group_id"
	DimensionRouteAlias    DimensionCode = "route_alias"
	DimensionAccountID     DimensionCode = "account_id"
	DimensionUpstreamModel DimensionCode = "upstream_model"
)

type MetricCode string

const MetricTotalTokens MetricCode = "total_tokens"

type ValueType string

const (
	ValueTypeInt64    ValueType = "int64"
	ValueTypeString   ValueType = "string"
	ValueTypeWildcard ValueType = "wildcard"
)

type DimensionDefinition struct {
	Code        DimensionCode `json:"code"`
	DisplayName string        `json:"display_name"`
	ValueType   ValueType     `json:"value_type"`
	Order       int           `json:"order"`
	Version     uint32        `json:"version"`
}

type MetricDefinition struct {
	Code        MetricCode `json:"code"`
	DisplayName string     `json:"display_name"`
	Unit        string     `json:"unit"`
	AllowQuota  bool       `json:"allow_quota"`
	Version     uint32     `json:"version"`
}

type DimensionValue struct {
	Type   ValueType `json:"type"`
	Int64  int64     `json:"int64,omitempty"`
	String string    `json:"string,omitempty"`
}

func Int64Value(value int64) DimensionValue {
	return DimensionValue{Type: ValueTypeInt64, Int64: value}
}

func StringValue(value string) DimensionValue {
	return DimensionValue{Type: ValueTypeString, String: strings.TrimSpace(value)}
}

func WildcardValue() DimensionValue { return DimensionValue{Type: ValueTypeWildcard} }

func validateQuotaDimensionValue(definition DimensionDefinition, value DimensionValue) error {
	if value.Type == ValueTypeWildcard {
		return nil
	}
	return validateDimensionValue(definition, value)
}

type ProjectionDefinition struct {
	ID             uint64
	Name           string
	DimensionCodes []DimensionCode
	MetricCodes    []MetricCode
}

type UsageEvent struct {
	OccurredAt  time.Time
	RequestType string
	Dimensions  map[DimensionCode]DimensionValue
	Metrics     map[MetricCode]int64
}

func (e UsageEvent) Validate() error {
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("occurred_at is required")
	}
	for code, value := range e.Dimensions {
		definition, ok := Dimension(code)
		if !ok {
			return fmt.Errorf("unknown dimension %q", code)
		}
		if err := validateDimensionValue(definition, value); err != nil {
			return err
		}
	}
	for code, value := range e.Metrics {
		if _, ok := Metric(code); !ok {
			return fmt.Errorf("unknown metric %q", code)
		}
		if value < 0 {
			return fmt.Errorf("metric %q must be non-negative", code)
		}
	}
	return nil
}

func validateDimensionValue(definition DimensionDefinition, value DimensionValue) error {
	if value.Type != definition.ValueType {
		return fmt.Errorf("dimension %q expects %s, got %s", definition.Code, definition.ValueType, value.Type)
	}
	switch value.Type {
	case ValueTypeInt64:
		if value.Int64 <= 0 {
			return fmt.Errorf("dimension %q must be positive", definition.Code)
		}
	case ValueTypeString:
		if strings.TrimSpace(value.String) == "" {
			return fmt.Errorf("dimension %q must not be empty", definition.Code)
		}
	default:
		return fmt.Errorf("dimension %q has unsupported value type %q", definition.Code, value.Type)
	}
	return nil
}
