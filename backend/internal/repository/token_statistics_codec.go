package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const tokenStatisticsKeyPrefix = "sub2api:token_stats"

type TokenStatisticsType string

const (
	TokenStatisticsModel          TokenStatisticsType = "model"
	TokenStatisticsUserModel      TokenStatisticsType = "user_model"
	TokenStatisticsGroupCandidate TokenStatisticsType = "group_candidate"
)

var tokenStatisticsLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic("load token statistics timezone: " + err.Error())
	}
	return location
}()

func TokenStatisticsKey(statisticsType TokenStatisticsType, usageDate time.Time) (string, error) {
	if err := validateTokenStatisticsType(statisticsType); err != nil {
		return "", fmt.Errorf("build token statistics key: %w", err)
	}
	if usageDate.IsZero() {
		return "", fmt.Errorf("build token statistics key type=%s: usage date must not be zero", statisticsType)
	}
	return fmt.Sprintf("%s:%s:%s", tokenStatisticsKeyPrefix, statisticsType, usageDate.In(tokenStatisticsLocation).Format(time.DateOnly)), nil
}

// TokenStatisticsExpireAt returns midnight after the configured number of
// complete retention days following the business date.
func TokenStatisticsExpireAt(usageDate time.Time, retentionDays int) (time.Time, error) {
	if usageDate.IsZero() {
		return time.Time{}, fmt.Errorf("calculate token statistics expiry: usage date must not be zero")
	}
	if retentionDays <= 0 {
		return time.Time{}, fmt.Errorf("calculate token statistics expiry: retention_days=%d, expected positive integer", retentionDays)
	}
	localDate := usageDate.In(tokenStatisticsLocation)
	startOfDate := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, tokenStatisticsLocation)
	return startOfDate.AddDate(0, 0, retentionDays+1), nil
}

func EncodeModelTokenStatisticsField(model string) (string, error) {
	return encodeTokenStatisticsField(TokenStatisticsModel, []string{model})
}

func DecodeModelTokenStatisticsField(field string) (string, error) {
	parts, err := decodeTokenStatisticsField(TokenStatisticsModel, field, 1)
	if err != nil {
		return "", err
	}
	return parts[0], nil
}

func EncodeUserModelTokenStatisticsField(userID int64, model string) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("encode token statistics field type=%s: user_id=%d, expected positive integer", TokenStatisticsUserModel, userID)
	}
	return encodeTokenStatisticsField(TokenStatisticsUserModel, []string{fmt.Sprintf("%d", userID), model})
}

func DecodeUserModelTokenStatisticsField(field string) (int64, string, error) {
	parts, err := decodeTokenStatisticsField(TokenStatisticsUserModel, field, 2)
	if err != nil {
		return 0, "", err
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || userID <= 0 {
		return 0, "", fmt.Errorf("decode token statistics field type=%s: invalid user_id=%q", TokenStatisticsUserModel, parts[0])
	}
	return userID, parts[1], nil
}

func EncodeGroupCandidateTokenStatisticsField(groupID int64, routeAlias, upstreamModel string) (string, error) {
	if groupID <= 0 {
		return "", fmt.Errorf("encode token statistics field type=%s: group_id=%d, expected positive integer", TokenStatisticsGroupCandidate, groupID)
	}
	return encodeTokenStatisticsField(TokenStatisticsGroupCandidate, []string{fmt.Sprintf("%d", groupID), routeAlias, upstreamModel})
}

func DecodeGroupCandidateTokenStatisticsField(field string) (int64, string, string, error) {
	parts, err := decodeTokenStatisticsField(TokenStatisticsGroupCandidate, field, 3)
	if err != nil {
		return 0, "", "", err
	}
	groupID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || groupID <= 0 {
		return 0, "", "", fmt.Errorf("decode token statistics field type=%s: invalid group_id=%q", TokenStatisticsGroupCandidate, parts[0])
	}
	return groupID, parts[1], parts[2], nil
}

func encodeTokenStatisticsField(statisticsType TokenStatisticsType, parts []string) (string, error) {
	if err := validateTokenStatisticsType(statisticsType); err != nil {
		return "", fmt.Errorf("encode token statistics field: %w", err)
	}
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", fmt.Errorf("encode token statistics field type=%s: component[%d] must not be empty", statisticsType, i)
		}
	}
	payload, err := json.Marshal(parts)
	if err != nil {
		return "", fmt.Errorf("encode token statistics field type=%s: marshal components: %w", statisticsType, err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeTokenStatisticsField(statisticsType TokenStatisticsType, field string, expectedParts int) ([]string, error) {
	if strings.TrimSpace(field) == "" {
		return nil, fmt.Errorf("decode token statistics field type=%s: field must not be empty", statisticsType)
	}
	payload, err := base64.RawURLEncoding.DecodeString(field)
	if err != nil {
		return nil, fmt.Errorf("decode token statistics field type=%s: invalid base64: %w", statisticsType, err)
	}
	var parts []string
	if err := json.Unmarshal(payload, &parts); err != nil {
		return nil, fmt.Errorf("decode token statistics field type=%s: invalid payload: %w", statisticsType, err)
	}
	if len(parts) != expectedParts {
		return nil, fmt.Errorf("decode token statistics field type=%s: component count=%d, expected=%d", statisticsType, len(parts), expectedParts)
	}
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf("decode token statistics field type=%s: component[%d] must not be empty", statisticsType, i)
		}
	}
	return parts, nil
}

func validateTokenStatisticsType(statisticsType TokenStatisticsType) error {
	switch statisticsType {
	case TokenStatisticsModel, TokenStatisticsUserModel, TokenStatisticsGroupCandidate:
		return nil
	default:
		return fmt.Errorf("invalid statistics type=%q", statisticsType)
	}
}
