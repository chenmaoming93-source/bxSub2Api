package repository

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/groupcandidatetokendailyusage"
	"github.com/Wei-Shaw/sub2api/ent/modeltokendailyusage"
	"github.com/Wei-Shaw/sub2api/ent/usermodeltokendailyusage"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type dailyTokenQuotaRepository struct{ client *dbent.Client }

func NewDailyTokenQuotaRepository(client *dbent.Client) service.DailyTokenQuotaRepository {
	return &dailyTokenQuotaRepository{client: client}
}

func (r *dailyTokenQuotaRepository) GetModelDailyTokenQuota(ctx context.Context, key service.ModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	day := timezone.StartOfDay(key.At)
	row, err := r.client.ModelTokenDailyUsage.Query().
		Where(modeltokendailyusage.ModelEQ(key.Model), modeltokendailyusage.UsageDateEQ(day)).Only(ctx)
	if dbent.IsNotFound(err) {
		return service.DailyTokenQuotaSnapshot{UsageDate: day}, nil
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get model daily token quota: %w", err)
	}
	return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: row.UsedTokens, DailyLimitTokens: cloneInt64(row.DailyLimitTokens)}, nil
}

func (r *dailyTokenQuotaRepository) GetUserModelDailyTokenQuota(ctx context.Context, key service.UserModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	day := timezone.StartOfDay(key.At)
	row, err := r.client.UserModelTokenDailyUsage.Query().
		Where(usermodeltokendailyusage.UserIDEQ(key.UserID), usermodeltokendailyusage.ModelEQ(key.Model), usermodeltokendailyusage.UsageDateEQ(day)).Only(ctx)
	if dbent.IsNotFound(err) {
		return service.DailyTokenQuotaSnapshot{UsageDate: day}, nil
	}
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, fmt.Errorf("get user model daily token quota: %w", err)
	}
	return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: row.UsedTokens, DailyLimitTokens: cloneInt64(row.DailyLimitTokens)}, nil
}

func (r *dailyTokenQuotaRepository) GetGroupCandidateDailyTokenQuota(ctx context.Context, key service.GroupCandidateDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	day := timezone.StartOfDay(key.At)
	row, err := r.client.GroupCandidateTokenDailyUsage.Query().Where(
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
	return service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: day, UsedTokens: row.UsedTokens, DailyLimitTokens: cloneInt64(row.DailyLimitTokens)}, nil
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

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

	modelLimit := increment.ModelDailyLimitTokens
	if modelLimit == nil {
		modelLimit, err = latestModelLimit(ctx, client, increment.ModelKey.Model, modelDay)
		if err != nil {
			return err
		}
	}
	userLimit := increment.UserModelDailyLimitTokens
	if userLimit == nil {
		userLimit, err = latestUserModelLimit(ctx, client, increment.UserModelKey.UserID, increment.UserModelKey.Model, userDay)
		if err != nil {
			return err
		}
	}

	if err := client.ModelTokenDailyUsage.Create().
		SetModel(increment.ModelKey.Model).SetUsageDate(modelDay).SetUsedTokens(increment.Tokens).
		SetNillableDailyLimitTokens(modelLimit).
		OnConflictColumns(modeltokendailyusage.FieldModel, modeltokendailyusage.FieldUsageDate).
		AddUsedTokens(increment.Tokens).Exec(ctx); err != nil {
		return fmt.Errorf("increment model daily token quota: %w", err)
	}
	if err := client.UserModelTokenDailyUsage.Create().
		SetUserID(increment.UserModelKey.UserID).SetModel(increment.UserModelKey.Model).SetUsageDate(userDay).SetUsedTokens(increment.Tokens).
		SetNillableDailyLimitTokens(userLimit).
		OnConflictColumns(usermodeltokendailyusage.FieldUserID, usermodeltokendailyusage.FieldModel, usermodeltokendailyusage.FieldUsageDate).
		AddUsedTokens(increment.Tokens).Exec(ctx); err != nil {
		return fmt.Errorf("increment user model daily token quota: %w", err)
	}
	groupUpsert := client.GroupCandidateTokenDailyUsage.Create().
		SetGroupID(increment.GroupCandidateKey.GroupID).
		SetRouteAlias(increment.GroupCandidateKey.RouteAlias).
		SetUpstreamModel(increment.GroupCandidateKey.UpstreamModel).
		SetUsageDate(groupDay).SetUsedTokens(increment.Tokens).
		SetNillableDailyLimitTokens(increment.GroupCandidateDailyLimitTokens).
		OnConflictColumns(
			groupcandidatetokendailyusage.FieldGroupID,
			groupcandidatetokendailyusage.FieldRouteAlias,
			groupcandidatetokendailyusage.FieldUpstreamModel,
			groupcandidatetokendailyusage.FieldUsageDate,
		).
		AddUsedTokens(increment.Tokens)
	if increment.GroupCandidateDailyLimitTokens == nil {
		groupUpsert = groupUpsert.ClearDailyLimitTokens()
	} else {
		groupUpsert = groupUpsert.SetDailyLimitTokens(*increment.GroupCandidateDailyLimitTokens)
	}
	if err := groupUpsert.Exec(ctx); err != nil {
		return fmt.Errorf("increment group candidate daily token quota: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("increment daily token quotas: commit: %w", err)
	}
	return nil
}

func latestModelLimit(ctx context.Context, client *dbent.Client, model string, before time.Time) (*int64, error) {
	row, err := client.ModelTokenDailyUsage.Query().
		Where(modeltokendailyusage.ModelEQ(model), modeltokendailyusage.UsageDateLT(before)).
		Order(dbent.Desc(modeltokendailyusage.FieldUsageDate)).First(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load previous model daily token limit: %w", err)
	}
	return cloneInt64(row.DailyLimitTokens), nil
}

func latestUserModelLimit(ctx context.Context, client *dbent.Client, userID int64, model string, before time.Time) (*int64, error) {
	row, err := client.UserModelTokenDailyUsage.Query().
		Where(usermodeltokendailyusage.UserIDEQ(userID), usermodeltokendailyusage.ModelEQ(model), usermodeltokendailyusage.UsageDateLT(before)).
		Order(dbent.Desc(usermodeltokendailyusage.FieldUsageDate)).First(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load previous user model daily token limit: %w", err)
	}
	return cloneInt64(row.DailyLimitTokens), nil
}
