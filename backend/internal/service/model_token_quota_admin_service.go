package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ModelTokenQuotaAdminService struct {
	repo      ModelTokenQuotaAdminRepository
	cache     ModelDailyTokenQuotaCacheInvalidator
	now       func() time.Time
	maxModels int
}

func NewModelTokenQuotaAdminService(repo ModelTokenQuotaAdminRepository, cache ModelDailyTokenQuotaCacheInvalidator) *ModelTokenQuotaAdminService {
	return &ModelTokenQuotaAdminService{
		repo:      repo,
		cache:     cache,
		now:       time.Now,
		maxModels: 100,
	}
}

func (s *ModelTokenQuotaAdminService) List(ctx context.Context) ([]ModelDailyTokenQuotaRecord, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model token quota service unavailable")
	}
	return s.repo.ListModelDailyTokenQuotas(ctx, s.now())
}

func (s *ModelTokenQuotaAdminService) Set(ctx context.Context, model string, limit *int64) (ModelDailyTokenQuotaRecord, error) {
	if s == nil || s.repo == nil {
		return ModelDailyTokenQuotaRecord{}, fmt.Errorf("model token quota service unavailable")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return ModelDailyTokenQuotaRecord{}, fmt.Errorf("model is required")
	}
	if len(model) > 255 {
		return ModelDailyTokenQuotaRecord{}, fmt.Errorf("model length must be <= 255")
	}
	if limit != nil && *limit < 0 {
		return ModelDailyTokenQuotaRecord{}, fmt.Errorf("daily_limit_tokens must be >= 0 or null")
	}
	at := s.now()
	record, err := s.repo.SetModelDailyTokenQuota(ctx, model, at, limit)
	if err != nil {
		return ModelDailyTokenQuotaRecord{}, err
	}
	if s.cache != nil {
		if err := s.cache.InvalidateModelDailyTokenQuota(ctx, ModelDailyTokenQuotaKey{Model: model, At: at}); err != nil {
			return ModelDailyTokenQuotaRecord{}, err
		}
	}
	return record, nil
}
