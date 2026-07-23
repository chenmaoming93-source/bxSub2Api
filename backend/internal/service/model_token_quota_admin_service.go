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
	reader    CurrentTokenUsageReader
	repairer  CurrentTokenUsageRepairer
}

func (s *ModelTokenQuotaAdminService) ConfigureCurrentTokenUsage(reader CurrentTokenUsageReader, repairer CurrentTokenUsageRepairer) *ModelTokenQuotaAdminService {
	s.reader, s.repairer = reader, repairer
	return s
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
	at := s.now()
	records, err := s.repo.ListModelDailyTokenQuotas(ctx, at)
	if err != nil {
		return nil, err
	}
	return s.realtime(ctx, at, records)
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
	records, err := s.realtime(ctx, at, []ModelDailyTokenQuotaRecord{record})
	if err != nil {
		return ModelDailyTokenQuotaRecord{}, err
	}
	return records[0], nil
}

func (s *ModelTokenQuotaAdminService) realtime(ctx context.Context, at time.Time, records []ModelDailyTokenQuotaRecord) ([]ModelDailyTokenQuotaRecord, error) {
	if s.reader == nil || len(records) == 0 {
		return records, nil
	}
	models := make([]string, 0, len(records))
	mysql := make([]ModelTokenUsageRow, 0, len(records))
	for _, record := range records {
		models = append(models, record.Model)
		mysql = append(mysql, ModelTokenUsageRow{UsageDate: record.UsageDate, Model: record.Model, UsedTokens: record.UsedTokens, DailyLimitTokens: record.DailyLimitTokens})
	}
	result, err := s.reader.ReadModelUsage(ctx, at, models)
	if err != nil {
		return records, nil
	}
	merged, repair := MergeModelTokenUsage(mysql, result.Rows)
	byKey := indexRows(merged, modelUsageKey)
	for i := range records {
		if row, ok := byKey[modelUsageKey(ModelTokenUsageRow{UsageDate: records[i].UsageDate, Model: records[i].Model})]; ok {
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
		_ = s.repairer.RepairModelUsage(ctx, at, positive)
	}
	return records, nil
}
