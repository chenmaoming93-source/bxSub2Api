package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/groupcandidatetokendailyusage"
	"github.com/Wei-Shaw/sub2api/ent/groupcandidatetokendailylimitconfig"
	"github.com/Wei-Shaw/sub2api/ent/modeltokendailyusage"
	"github.com/Wei-Shaw/sub2api/ent/modeltokendailylimitconfig"
	"github.com/Wei-Shaw/sub2api/ent/usermodeltokendailyusage"
	"github.com/Wei-Shaw/sub2api/ent/usermodeltokendailylimitconfig"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type dailyTokenQuotaRepository struct{ client *dbent.Client }

func NewDailyTokenQuotaRepository(client *dbent.Client) service.DailyTokenQuotaRepository {
	return &dailyTokenQuotaRepository{client: client}
}

// ─── Get (quota snapshot for a single scope) ──────────────────────────

func (r *dailyTokenQuotaRepository) GetModelDailyTokenQuota(ctx context.Context, key service.ModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	day := timezone.StartOfDay(key.At)
	// Read limit from the independent config table.
	cfg, err := r.client.ModelTokenDailyLimitConfig.Query().
		Where(modeltokendailylimitconfig.ModelEQ(key.Model)).Only(ctx)
	if dbent.IsNotFound(err) {
		// No config → no limit, but usage may still exist.
		return usageOnlyModelSnapshot(ctx, r.client, key.Model, day)
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get model daily token quota: %w", err)
	}
	usage, err := r.client.ModelTokenDailyUsage.Query().
		Where(modeltokendailyusage.ModelEQ(key.Model), modeltokendailyusage.UsageDateEQ(day)).Only(ctx)
	if dbent.IsNotFound(err) {
		return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: 0, DailyLimitTokens: cloneInt64(cfg.DailyLimitTokens)}, nil
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get model daily token quota: %w", err)
	}
	return service.DailyTokenQuotaSnapshot{
		Exists: true, UsageDate: day,
		UsedTokens:       usage.UsedTokens,
		DailyLimitTokens: cloneInt64(cfg.DailyLimitTokens),
	}, nil
}

func (r *dailyTokenQuotaRepository) GetUserModelDailyTokenQuota(ctx context.Context, key service.UserModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	day := timezone.StartOfDay(key.At)
	cfg, err := r.client.UserModelTokenDailyLimitConfig.Query().
		Where(usermodeltokendailylimitconfig.UserIDEQ(key.UserID), usermodeltokendailylimitconfig.ModelEQ(key.Model)).Only(ctx)
	if dbent.IsNotFound(err) {
		return usageOnlyUserModelSnapshot(ctx, r.client, key.UserID, key.Model, day)
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get user model daily token quota: %w", err)
	}
	usage, err := r.client.UserModelTokenDailyUsage.Query().
		Where(usermodeltokendailyusage.UserIDEQ(key.UserID), usermodeltokendailyusage.ModelEQ(key.Model), usermodeltokendailyusage.UsageDateEQ(day)).Only(ctx)
	if dbent.IsNotFound(err) {
		return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: 0, DailyLimitTokens: cloneInt64(cfg.DailyLimitTokens)}, nil
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get user model daily token quota: %w", err)
	}
	return service.DailyTokenQuotaSnapshot{
		Exists: true, UsageDate: day,
		UsedTokens:       usage.UsedTokens,
		DailyLimitTokens: cloneInt64(cfg.DailyLimitTokens),
	}, nil
}

func (r *dailyTokenQuotaRepository) GetGroupCandidateDailyTokenQuota(ctx context.Context, key service.GroupCandidateDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	day := timezone.StartOfDay(key.At)
	cfg, err := r.client.GroupCandidateTokenDailyLimitConfig.Query().Where(
		groupcandidatetokendailylimitconfig.GroupIDEQ(key.GroupID),
		groupcandidatetokendailylimitconfig.RouteAliasEQ(key.RouteAlias),
		groupcandidatetokendailylimitconfig.UpstreamModelEQ(key.UpstreamModel),
	).Only(ctx)
	if dbent.IsNotFound(err) {
		return usageOnlyGroupCandidateSnapshot(ctx, r.client, key, day)
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get group candidate daily token quota: %w", err)
	}
	usage, err := r.client.GroupCandidateTokenDailyUsage.Query().Where(
		groupcandidatetokendailyusage.GroupIDEQ(key.GroupID),
		groupcandidatetokendailyusage.RouteAliasEQ(key.RouteAlias),
		groupcandidatetokendailyusage.UpstreamModelEQ(key.UpstreamModel),
		groupcandidatetokendailyusage.UsageDateEQ(day),
	).Only(ctx)
	if dbent.IsNotFound(err) {
		return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: 0, DailyLimitTokens: cloneInt64(cfg.DailyLimitTokens)}, nil
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get group candidate daily token quota: %w", err)
	}
	return service.DailyTokenQuotaSnapshot{
		Exists: true, UsageDate: day,
		UsedTokens:       usage.UsedTokens,
		DailyLimitTokens: cloneInt64(cfg.DailyLimitTokens),
	}, nil
}

// ─── Helpers for config-not-found fallback ───────────────────────────

func usageOnlyModelSnapshot(ctx context.Context, client *dbent.Client, model string, day time.Time) (service.DailyTokenQuotaSnapshot, error) {
	row, err := client.ModelTokenDailyUsage.Query().
		Where(modeltokendailyusage.ModelEQ(model), modeltokendailyusage.UsageDateEQ(day)).Only(ctx)
	if dbent.IsNotFound(err) {
		return service.DailyTokenQuotaSnapshot{UsageDate: day}, nil
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get model daily token quota: %w", err)
	}
	return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: row.UsedTokens}, nil
}

func usageOnlyUserModelSnapshot(ctx context.Context, client *dbent.Client, userID int64, model string, day time.Time) (service.DailyTokenQuotaSnapshot, error) {
	row, err := client.UserModelTokenDailyUsage.Query().
		Where(usermodeltokendailyusage.UserIDEQ(userID), usermodeltokendailyusage.ModelEQ(model), usermodeltokendailyusage.UsageDateEQ(day)).Only(ctx)
	if dbent.IsNotFound(err) {
		return service.DailyTokenQuotaSnapshot{UsageDate: day}, nil
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get user model daily token quota: %w", err)
	}
	return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: row.UsedTokens}, nil
}

func usageOnlyGroupCandidateSnapshot(ctx context.Context, client *dbent.Client, key service.GroupCandidateDailyTokenQuotaKey, day time.Time) (service.DailyTokenQuotaSnapshot, error) {
	row, err := client.GroupCandidateTokenDailyUsage.Query().Where(
		groupcandidatetokendailyusage.GroupIDEQ(key.GroupID),
		groupcandidatetokendailyusage.RouteAliasEQ(key.RouteAlias),
		groupcandidatetokendailyusage.UpstreamModelEQ(key.UpstreamModel),
		groupcandidatetokendailyusage.UsageDateEQ(day),
	).Only(ctx)
	if dbent.IsNotFound(err) {
		return service.DailyTokenQuotaSnapshot{UsageDate: day}, nil
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get group candidate daily token quota: %w", err)
	}
	return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: row.UsedTokens}, nil
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// ─── List (admin read) ───────────────────────────────────────────────

func (r *dailyTokenQuotaRepository) ListModelDailyTokenQuotas(ctx context.Context, at time.Time) ([]service.ModelDailyTokenQuotaRecord, error) {
	day := timezone.StartOfDay(at)
	// Read all config rows — they are date-independent.
	configs, err := r.client.ModelTokenDailyLimitConfig.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list model daily token quotas: %w", err)
	}
	// Read today's usage and build a model→used map.
	usages, err := r.client.ModelTokenDailyUsage.Query().
		Where(modeltokendailyusage.UsageDateEQ(day)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list model daily token quotas: %w", err)
	}
	usedMap := make(map[string]int64, len(usages))
	for _, u := range usages {
		usedMap[u.Model] = u.UsedTokens
	}
	records := make([]service.ModelDailyTokenQuotaRecord, 0, len(configs))
	for _, cfg := range configs {
		records = append(records, service.ModelDailyTokenQuotaRecord{
			Model:            cfg.Model,
			UsageDate:        day,
			UsedTokens:       usedMap[cfg.Model],
			DailyLimitTokens: cloneInt64(cfg.DailyLimitTokens),
		})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Model < records[j].Model })
	return records, nil
}

func (r *dailyTokenQuotaRepository) SetModelDailyTokenQuota(ctx context.Context, model string, at time.Time, limit *int64) (service.ModelDailyTokenQuotaRecord, error) {
	day := timezone.StartOfDay(at)
	// Upsert the config row (date-independent).
	if limit == nil {
		// Delete config → no limit.
		_, err := r.client.ModelTokenDailyLimitConfig.Delete().
			Where(modeltokendailylimitconfig.ModelEQ(model)).Exec(ctx)
		if err != nil {
			return service.ModelDailyTokenQuotaRecord{}, fmt.Errorf("set model daily token quota: %w", err)
		}
	} else {
		err := r.client.ModelTokenDailyLimitConfig.Create().
			SetModel(model).SetDailyLimitTokens(*limit).
			OnConflictColumns(modeltokendailylimitconfig.FieldModel).
			UpdateDailyLimitTokens().Exec(ctx)
		if err != nil {
			return service.ModelDailyTokenQuotaRecord{}, fmt.Errorf("set model daily token quota: %w", err)
		}
	}
	// Read back today's used_tokens.
	usedTokens := int64(0)
	usage, err := r.client.ModelTokenDailyUsage.Query().
		Where(modeltokendailyusage.ModelEQ(model), modeltokendailyusage.UsageDateEQ(day)).Only(ctx)
	if err == nil {
		usedTokens = usage.UsedTokens
	} else if !dbent.IsNotFound(err) {
		return service.ModelDailyTokenQuotaRecord{}, fmt.Errorf("set model daily token quota: %w", err)
	}
	return service.ModelDailyTokenQuotaRecord{
		Model: model, UsageDate: day, UsedTokens: usedTokens, DailyLimitTokens: limit,
	}, nil
}

func (r *dailyTokenQuotaRepository) ListUserModelDailyTokenQuotas(ctx context.Context, userID int64, at time.Time) ([]service.UserModelDailyTokenQuotaRecord, error) {
	day := timezone.StartOfDay(at)
	configs, err := r.client.UserModelTokenDailyLimitConfig.Query().
		Where(usermodeltokendailylimitconfig.UserIDEQ(userID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user model daily token quotas: %w", err)
	}
	usages, err := r.client.UserModelTokenDailyUsage.Query().
		Where(usermodeltokendailyusage.UserIDEQ(userID), usermodeltokendailyusage.UsageDateEQ(day)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user model daily token quotas: %w", err)
	}
	// key = model
	usedMap := make(map[string]int64, len(usages))
	for _, u := range usages {
		usedMap[u.Model] = u.UsedTokens
	}
	records := make([]service.UserModelDailyTokenQuotaRecord, 0, len(configs))
	for _, cfg := range configs {
		records = append(records, service.UserModelDailyTokenQuotaRecord{
			UserID:           userID,
			Model:            cfg.Model,
			UsageDate:        day,
			UsedTokens:       usedMap[cfg.Model],
			DailyLimitTokens: cloneInt64(cfg.DailyLimitTokens),
		})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Model < records[j].Model })
	return records, nil
}

func (r *dailyTokenQuotaRepository) UpsertUserModelDailyTokenQuotas(ctx context.Context, userID int64, at time.Time, inputs []service.UserModelDailyTokenQuotaInput) ([]service.UserModelDailyTokenQuotaRecord, error) {
	for _, input := range inputs {
		if input.DailyLimitTokens == nil {
			// Delete config to clear the limit.
			_, err := r.client.UserModelTokenDailyLimitConfig.Delete().
				Where(usermodeltokendailylimitconfig.UserIDEQ(userID), usermodeltokendailylimitconfig.ModelEQ(input.Model)).Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("upsert user model daily token quota: %w", err)
			}
		} else {
			err := r.client.UserModelTokenDailyLimitConfig.Create().
				SetUserID(userID).SetModel(input.Model).SetDailyLimitTokens(*input.DailyLimitTokens).
				OnConflictColumns(usermodeltokendailylimitconfig.FieldUserID, usermodeltokendailylimitconfig.FieldModel).
				UpdateDailyLimitTokens().Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("upsert user model daily token quota: %w", err)
			}
		}
	}

	// Delete configs for models not in this request.
	if len(inputs) == 0 {
		_, err := r.client.UserModelTokenDailyLimitConfig.Delete().
			Where(usermodeltokendailylimitconfig.UserIDEQ(userID)).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("delete stale user model daily token configs: %w", err)
		}
	} else {
		retained := make([]string, len(inputs))
		for i, input := range inputs {
			retained[i] = input.Model
		}
		_, err := r.client.UserModelTokenDailyLimitConfig.Delete().
			Where(usermodeltokendailylimitconfig.UserIDEQ(userID), usermodeltokendailylimitconfig.ModelNotIn(retained...)).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("delete stale user model daily token configs: %w", err)
		}
	}

	return r.ListUserModelDailyTokenQuotas(ctx, userID, at)
}

func (r *dailyTokenQuotaRepository) DeleteUserModelTokenQuotaByModel(ctx context.Context, userID int64, model string) error {
	_, err := r.client.UserModelTokenDailyLimitConfig.Delete().
		Where(usermodeltokendailylimitconfig.UserIDEQ(userID), usermodeltokendailylimitconfig.ModelEQ(model)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete user model daily token config by model: %w", err)
	}
	return nil
}

// ─── Increment (atomic usage write) ──────────────────────────────────

func (r *dailyTokenQuotaRepository) IncrementDailyTokenQuotas(ctx context.Context, increment service.DailyTokenQuotaIncrement) error {
	if increment.Tokens <= 0 {
		return fmt.Errorf("increment daily token quotas: tokens must be positive")
	}
	modelDay := timezone.StartOfDay(increment.ModelKey.At)
	userDay := timezone.StartOfDay(increment.UserModelKey.At)
	groupDay := timezone.StartOfDay(increment.GroupCandidateKey.At)
	if increment.ModelKey.Model == "" || increment.UserModelKey.UserID <= 0 || increment.UserModelKey.Model == "" ||
		increment.GroupCandidateKey.GroupID <= 0 || increment.GroupCandidateKey.RouteAlias == "" || increment.GroupCandidateKey.UpstreamModel == "" {
		return fmt.Errorf("increment daily token quotas: incomplete quota key")
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("increment daily token quotas: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()

	// Model scope — write only used_tokens.
	if err := client.ModelTokenDailyUsage.Create().
		SetModel(increment.ModelKey.Model).SetUsageDate(modelDay).SetUsedTokens(increment.Tokens).
		OnConflictColumns(modeltokendailyusage.FieldModel, modeltokendailyusage.FieldUsageDate).
		AddUsedTokens(increment.Tokens).Exec(ctx); err != nil {
		return fmt.Errorf("increment model daily token quota: %w", err)
	}
	// User+Model scope — write only used_tokens.
	if err := client.UserModelTokenDailyUsage.Create().
		SetUserID(increment.UserModelKey.UserID).SetModel(increment.UserModelKey.Model).SetUsageDate(userDay).SetUsedTokens(increment.Tokens).
		OnConflictColumns(usermodeltokendailyusage.FieldUserID, usermodeltokendailyusage.FieldModel, usermodeltokendailyusage.FieldUsageDate).
		AddUsedTokens(increment.Tokens).Exec(ctx); err != nil {
		return fmt.Errorf("increment user model daily token quota: %w", err)
	}
	// Group candidate scope — write only used_tokens.
	if err := client.GroupCandidateTokenDailyUsage.Create().
		SetGroupID(increment.GroupCandidateKey.GroupID).
		SetRouteAlias(increment.GroupCandidateKey.RouteAlias).
		SetUpstreamModel(increment.GroupCandidateKey.UpstreamModel).
		SetUsageDate(groupDay).SetUsedTokens(increment.Tokens).
		OnConflictColumns(
			groupcandidatetokendailyusage.FieldGroupID,
			groupcandidatetokendailyusage.FieldRouteAlias,
			groupcandidatetokendailyusage.FieldUpstreamModel,
			groupcandidatetokendailyusage.FieldUsageDate,
		).
		AddUsedTokens(increment.Tokens).Exec(ctx); err != nil {
		return fmt.Errorf("increment group candidate daily token quota: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("increment daily token quotas: commit: %w", err)
	}
	return nil
}
