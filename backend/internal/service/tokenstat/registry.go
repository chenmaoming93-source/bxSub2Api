package tokenstat

import (
	"fmt"
	"sort"
	"strings"
)

var registeredDimensions = []DimensionDefinition{
	{Code: DimensionUserID, DisplayName: "用户", ValueType: ValueTypeInt64, Order: 10, Version: 1},
	{Code: DimensionAPIKeyID, DisplayName: "API Key", ValueType: ValueTypeInt64, Order: 20, Version: 1},
	{Code: DimensionGroupID, DisplayName: "分组", ValueType: ValueTypeInt64, Order: 30, Version: 1},
	{Code: DimensionRouteAlias, DisplayName: "路由别名", ValueType: ValueTypeString, Order: 40, Version: 1},
	{Code: DimensionAccountID, DisplayName: "模型账号", ValueType: ValueTypeInt64, Order: 50, Version: 1},
	{Code: DimensionUpstreamModel, DisplayName: "上游模型", ValueType: ValueTypeString, Order: 60, Version: 1},
	{Code: DimensionDepartment, DisplayName: "部门", ValueType: ValueTypeString, Order: 70, Version: 1},
}

var registeredMetrics = []MetricDefinition{
	{Code: MetricTotalTokens, DisplayName: "总 Token", Unit: "token", AllowQuota: true, Version: 1},
}

func Dimensions() []DimensionDefinition {
	return append([]DimensionDefinition(nil), registeredDimensions...)
}

// ConfigurableDimensions lists dimensions that administrators may select for
// new projections. Department remains internally recognized for compatibility
// with existing events and projections, but new reports use current users.department
// joined to the user_id projection instead.
func ConfigurableDimensions() []DimensionDefinition {
	result := make([]DimensionDefinition, 0, len(registeredDimensions))
	for _, definition := range registeredDimensions {
		if definition.Code == DimensionDepartment {
			continue
		}
		result = append(result, definition)
	}
	return result
}

func Metrics() []MetricDefinition {
	return append([]MetricDefinition(nil), registeredMetrics...)
}

func Dimension(code DimensionCode) (DimensionDefinition, bool) {
	for _, definition := range registeredDimensions {
		if definition.Code == code {
			return definition, true
		}
	}
	return DimensionDefinition{}, false
}

func Metric(code MetricCode) (MetricDefinition, bool) {
	for _, definition := range registeredMetrics {
		if definition.Code == code {
			return definition, true
		}
	}
	return MetricDefinition{}, false
}

func CanonicalDimensionCodes(codes []DimensionCode) ([]DimensionCode, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("at least one dimension is required")
	}
	seen := make(map[DimensionCode]struct{}, len(codes))
	result := append([]DimensionCode(nil), codes...)
	for _, code := range result {
		if _, ok := Dimension(code); !ok {
			return nil, fmt.Errorf("unknown dimension %q", code)
		}
		if _, ok := seen[code]; ok {
			return nil, fmt.Errorf("duplicate dimension %q", code)
		}
		seen[code] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool {
		left, _ := Dimension(result[i])
		right, _ := Dimension(result[j])
		return left.Order < right.Order
	})
	return result, nil
}

func DimensionSignature(codes []DimensionCode) (string, error) {
	canonical, err := CanonicalDimensionCodes(codes)
	if err != nil {
		return "", err
	}
	parts := make([]string, len(canonical))
	for i, code := range canonical {
		parts[i] = string(code)
	}
	return strings.Join(parts, ","), nil
}

func ValidateProjection(definition ProjectionDefinition) error {
	if strings.TrimSpace(definition.Name) == "" {
		return fmt.Errorf("projection name is required")
	}
	if _, err := CanonicalDimensionCodes(definition.DimensionCodes); err != nil {
		return err
	}
	if len(definition.MetricCodes) == 0 {
		return fmt.Errorf("at least one metric is required")
	}
	seen := make(map[MetricCode]struct{}, len(definition.MetricCodes))
	for _, code := range definition.MetricCodes {
		if _, ok := Metric(code); !ok {
			return fmt.Errorf("unknown metric %q", code)
		}
		if _, ok := seen[code]; ok {
			return fmt.Errorf("duplicate metric %q", code)
		}
		seen[code] = struct{}{}
	}
	return nil
}
