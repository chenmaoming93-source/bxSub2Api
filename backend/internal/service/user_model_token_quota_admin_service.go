package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type UserModelTokenQuotaAdminService struct {
	repo     UserModelTokenQuotaAdminRepository
	cache    UserModelDailyTokenQuotaCacheInvalidator
	now      func() time.Time
	reader   CurrentTokenUsageReader
	repairer CurrentTokenUsageRepairer
}

func (s *UserModelTokenQuotaAdminService) ConfigureCurrentTokenUsage(reader CurrentTokenUsageReader, repairer CurrentTokenUsageRepairer) *UserModelTokenQuotaAdminService {
	s.reader, s.repairer = reader, repairer
	return s
}

func NewUserModelTokenQuotaAdminService(repo UserModelTokenQuotaAdminRepository, cache UserModelDailyTokenQuotaCacheInvalidator) *UserModelTokenQuotaAdminService {
	return &UserModelTokenQuotaAdminService{repo: repo, cache: cache, now: time.Now}
}

func (s *UserModelTokenQuotaAdminService) List(ctx context.Context, userID int64) ([]UserModelDailyTokenQuotaRecord, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("user model token quota service unavailable")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("user_id must be positive")
	}
	at := s.now()
	records, err := s.repo.ListUserModelDailyTokenQuotas(ctx, userID, at)
	if err != nil {
		return nil, err
	}
	return s.realtime(ctx, userID, at, records)
}

func (s *UserModelTokenQuotaAdminService) Upsert(ctx context.Context, userID int64, inputs []UserModelDailyTokenQuotaInput) ([]UserModelDailyTokenQuotaRecord, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("user model token quota service unavailable")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("user_id must be positive")
	}
	seen := make(map[string]struct{}, len(inputs))
	clean := make([]UserModelDailyTokenQuotaInput, 0, len(inputs))
	for _, input := range inputs {
		model := strings.TrimSpace(input.Model)
		if model == "" {
			return nil, fmt.Errorf("model is required")
		}
		if len(model) > 255 {
			return nil, fmt.Errorf("model length must be <= 255")
		}
		if _, ok := seen[model]; ok {
			return nil, fmt.Errorf("duplicate model: %s", model)
		}
		seen[model] = struct{}{}
		if input.DailyLimitTokens != nil && *input.DailyLimitTokens < 0 {
			return nil, fmt.Errorf("daily_limit_tokens must be >= 0 or null")
		}
		clean = append(clean, UserModelDailyTokenQuotaInput{Model: model, DailyLimitTokens: input.DailyLimitTokens})
	}
	at := s.now()
	records, err := s.repo.UpsertUserModelDailyTokenQuotas(ctx, userID, at, clean)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		for _, input := range clean {
			if err := s.cache.InvalidateUserModelDailyTokenQuota(ctx, UserModelDailyTokenQuotaKey{UserID: userID, Model: input.Model, At: at}); err != nil {
				return nil, err
			}
		}
	}
	return s.realtime(ctx, userID, at, records)
}

func (s *UserModelTokenQuotaAdminService) realtime(ctx context.Context, userID int64, at time.Time, records []UserModelDailyTokenQuotaRecord) ([]UserModelDailyTokenQuotaRecord, error) {
	if s.reader == nil || len(records) == 0 {
		return records, nil
	}
	filters := make([]UserTokenUsageRow, 0, len(records))
	mysql := make([]UserTokenUsageRow, 0, len(records))
	for _, record := range records {
		row := UserTokenUsageRow{UsageDate: record.UsageDate, UserID: userID, Model: record.Model, UsedTokens: record.UsedTokens, DailyLimitTokens: record.DailyLimitTokens}
		filters = append(filters, row)
		mysql = append(mysql, row)
	}
	result, err := s.reader.ReadUserModelUsage(ctx, at, filters)
	if err != nil {
		return records, nil
	}
	merged, repair := MergeUserTokenUsage(mysql, result.Rows)
	byKey := indexRows(merged, userUsageKey)
	for i := range records {
		key := userUsageKey(UserTokenUsageRow{UsageDate: records[i].UsageDate, UserID: userID, Model: records[i].Model})
		if row, ok := byKey[key]; ok {
			records[i].UsedTokens = row.UsedTokens
		}
	}
	positive := repair[:0]
	for _, row := range repair {
		if row.UsedTokens > 0 {
			positive = append(positive, row)
		}
	}
	if len(positive) > 0 && s.repairer != nil {
		_ = s.repairer.RepairUserModelUsage(ctx, at, positive)
	}
	return records, nil
}
