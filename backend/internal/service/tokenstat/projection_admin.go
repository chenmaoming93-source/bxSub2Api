package tokenstat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/tokenstatprojection"
	"github.com/redis/go-redis/v9"
)

const (
	ProjectionStatusDraft     = "DRAFT"
	ProjectionStatusPublished = "PUBLISHED"
	ProjectionStatusActive    = "ACTIVE"
	ProjectionStatusDisabled  = "DISABLED"

	configVersionKey = "sub2api:dynamic_token_stats_config:v1:version"
	configActiveKey  = "sub2api:dynamic_token_stats_config:v1:active"
	configChannel    = "sub2api:dynamic_token_stats_config:v1:changed"
)

var ErrInvalidProjectionTransition = errors.New("invalid projection state transition")

type ProjectionAdminService struct {
	client        *ent.Client
	redis         *redis.Client
	localVersion  atomic.Uint64
	active        atomic.Value
	quotaChecker  *QuotaChecker
	subscribeOnce sync.Once
}

func (s *ProjectionAdminService) AttachQuotaChecker(checker *QuotaChecker) {
	s.quotaChecker = checker
	if s.redis != nil && checker != nil {
		s.subscribeOnce.Do(func() { go s.subscribeConfigurationChanges() })
	}
}

func (s *ProjectionAdminService) subscribeConfigurationChanges() {
	pubsub := s.redis.Subscribe(context.Background(), configChannel, quotaConfigChannel)
	defer pubsub.Close()
	for message := range pubsub.Channel() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var err error
		if message.Channel == configChannel {
			err = s.RefreshActive(ctx)
		} else {
			err = s.LoadQuotaRules(ctx, s.quotaChecker)
		}
		cancel()
		if err != nil {
			slog.Error("dynamic token statistics configuration refresh failed", "channel", message.Channel, "error", err)
		}
	}
}

type ProjectionInput struct {
	Name           string
	DimensionCodes []DimensionCode
	MetricCodes    []MetricCode
	CreatedBy      uint64
}

func NewProjectionAdminService(client *ent.Client, redisClient *redis.Client) *ProjectionAdminService {
	service := &ProjectionAdminService{client: client, redis: redisClient}
	service.active.Store([]ProjectionDefinition{})
	return service
}

func (s *ProjectionAdminService) ActiveProjections() []ProjectionDefinition {
	current, _ := s.active.Load().([]ProjectionDefinition)
	return append([]ProjectionDefinition(nil), current...)
}

func (s *ProjectionAdminService) RefreshActive(ctx context.Context) error {
	active, err := s.client.TokenStatProjection.Query().
		Where(tokenstatprojection.StatusEQ(ProjectionStatusActive)).All(ctx)
	if err != nil {
		return err
	}
	definitions := make([]ProjectionDefinition, 0, len(active))
	for _, projection := range active {
		codes := make([]DimensionCode, len(projection.DimensionCodes))
		for i, code := range projection.DimensionCodes {
			codes[i] = DimensionCode(code)
		}
		definitions = append(definitions, ProjectionDefinition{
			ID: uint64(projection.ID), Name: projection.Name, DimensionCodes: codes,
			MetricCodes: []MetricCode{MetricTotalTokens},
		})
	}
	s.active.Store(definitions)
	return nil
}

func (s *ProjectionAdminService) List(ctx context.Context) ([]*ent.TokenStatProjection, error) {
	return s.client.TokenStatProjection.Query().Order(ent.Asc(tokenstatprojection.FieldID)).All(ctx)
}

func (s *ProjectionAdminService) Get(ctx context.Context, id int64) (*ent.TokenStatProjection, error) {
	return s.client.TokenStatProjection.Get(ctx, id)
}

func (s *ProjectionAdminService) Create(ctx context.Context, input ProjectionInput) (*ent.TokenStatProjection, error) {
	definition := ProjectionDefinition{Name: input.Name, DimensionCodes: input.DimensionCodes, MetricCodes: input.MetricCodes}
	if err := ValidateProjection(definition); err != nil {
		return nil, err
	}
	canonical, _ := CanonicalDimensionCodes(input.DimensionCodes)
	signature, _ := DimensionSignature(canonical)
	codes := dimensionStrings(canonical)
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	projection, err := tx.TokenStatProjection.Create().
		SetName(strings.TrimSpace(input.Name)).
		SetDimensionCodes(codes).
		SetDimensionSignature(signature).
		SetCreatedBy(input.CreatedBy).
		Save(ctx)
	if err == nil {
		for _, code := range input.MetricCodes {
			_, err = tx.TokenStatProjectionMetric.Create().
				SetProjectionID(projection.ID).
				SetMetricCode(string(code)).
				SetStatus(ProjectionStatusActive).
				Save(ctx)
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return projection, nil
}

func (s *ProjectionAdminService) UpdateDraft(ctx context.Context, id int64, input ProjectionInput) (*ent.TokenStatProjection, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status != ProjectionStatusDraft {
		return nil, ErrInvalidProjectionTransition
	}
	if err := ValidateProjection(ProjectionDefinition{Name: input.Name, DimensionCodes: input.DimensionCodes, MetricCodes: input.MetricCodes}); err != nil {
		return nil, err
	}
	canonical, _ := CanonicalDimensionCodes(input.DimensionCodes)
	signature, _ := DimensionSignature(canonical)
	return s.client.TokenStatProjection.UpdateOneID(id).
		SetName(strings.TrimSpace(input.Name)).
		SetDimensionCodes(dimensionStrings(canonical)).
		SetDimensionSignature(signature).
		Save(ctx)
}

func (s *ProjectionAdminService) Publish(ctx context.Context, id int64) (*ent.TokenStatProjection, error) {
	return s.transitionAndPublish(ctx, id, ProjectionStatusDraft, ProjectionStatusPublished)
}

func (s *ProjectionAdminService) Activate(ctx context.Context, id int64) (*ent.TokenStatProjection, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status != ProjectionStatusPublished && current.Status != ProjectionStatusDisabled {
		return nil, ErrInvalidProjectionTransition
	}
	now := time.Now()
	projection, err := s.client.TokenStatProjection.UpdateOneID(id).
		SetStatus(ProjectionStatusActive).
		SetEnabledAt(now).
		ClearDisabledAt().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.publishConfig(ctx); err != nil {
		return nil, err
	}
	if err := s.enablePendingQuotas(ctx, id); err != nil {
		return nil, err
	}
	return projection, nil
}

func (s *ProjectionAdminService) Disable(ctx context.Context, id int64) (*ent.TokenStatProjection, error) {
	if err := s.CanDisableProjection(ctx, id); err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status != ProjectionStatusPublished && current.Status != ProjectionStatusActive {
		return nil, ErrInvalidProjectionTransition
	}
	now := time.Now()
	updated, err := s.client.TokenStatProjection.UpdateOneID(id).
		SetStatus(ProjectionStatusDisabled).SetDisabledAt(now).Save(ctx)
	if err != nil {
		return nil, err
	}
	return updated, s.publishConfig(ctx)
}

func (s *ProjectionAdminService) transitionAndPublish(ctx context.Context, id int64, from, to string) (*ent.TokenStatProjection, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status != from {
		return nil, ErrInvalidProjectionTransition
	}
	now := time.Now()
	update := s.client.TokenStatProjection.UpdateOneID(id).SetStatus(to)
	if to == ProjectionStatusPublished {
		update.SetPublishedAt(now)
	} else {
		update.SetEnabledAt(now)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.publishConfig(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *ProjectionAdminService) publishConfig(ctx context.Context) error {
	active, err := s.client.TokenStatProjection.Query().
		Where(tokenstatprojection.StatusEQ(ProjectionStatusActive)).
		All(ctx)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(active)
	if err != nil {
		return err
	}
	if s.redis == nil {
		observability.configVersion.Store(s.localVersion.Add(1))
		return s.RefreshActive(ctx)
	}
	version, err := s.redis.Incr(ctx, configVersionKey).Uint64()
	if err != nil {
		return fmt.Errorf("increment dynamic token statistics config version: %w", err)
	}
	observability.configVersion.Store(version)
	if err := s.redis.Set(ctx, configActiveKey, payload, 0).Err(); err != nil {
		return err
	}
	s.localVersion.Store(version)
	if err := s.redis.Publish(ctx, configChannel, version).Err(); err != nil {
		return err
	}
	return s.RefreshActive(ctx)
}

func dimensionStrings(codes []DimensionCode) []string {
	result := make([]string, len(codes))
	for i, code := range codes {
		result[i] = string(code)
	}
	return result
}
