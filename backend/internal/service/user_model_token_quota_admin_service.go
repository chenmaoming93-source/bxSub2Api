package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type UserModelTokenQuotaAdminService struct {
	repo  UserModelTokenQuotaAdminRepository
	cache UserModelDailyTokenQuotaCacheInvalidator
	now   func() time.Time
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
	return s.repo.ListUserModelDailyTokenQuotas(ctx, userID, s.now())
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
	return records, nil
}
