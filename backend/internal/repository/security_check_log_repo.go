package repository

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/ent/securitychecklog"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// InsertSecurityCheckLogs writes a batch and ignores duplicate event IDs.
func (r *groupRepository) InsertSecurityCheckLogs(ctx context.Context, records []service.SecurityCheckLogRecord) error {
	if len(records) == 0 {
		return nil
	}
	builders := make([]*dbent.SecurityCheckLogCreate, 0, len(records))
	for _, record := range records {
		storedBody, originalBytes, storedBytes, truncated, err := service.PrepareSecurityCheckRequestBody(record.RequestBody)
		if err != nil {
			return err
		}
		triggered := make([]domain.SecurityCheckTriggeredRule, 0, len(record.TriggeredRules))
		for _, rule := range record.TriggeredRules {
			triggered = append(triggered, domain.SecurityCheckTriggeredRule{
				Dimension: rule.Dimension,
				Threshold: rule.Threshold,
				Action:    rule.Action,
				RiskProb:  rule.RiskProb,
			})
		}
		createdAt := record.EnqueuedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		builder := r.client.SecurityCheckLog.Create().
			SetEventID(record.Metadata.EventID).
			SetConfigVersion(record.ConfigVersion).
			SetRulesSnapshot(record.RulesSnapshot).
			SetRequestBodyOriginalBytes(originalBytes).
			SetRequestBodyStoredBytes(storedBytes).
			SetRequestBodyTruncated(truncated).
			SetCheckStatus(string(record.CheckStatus)).
			SetDecision(string(record.Decision)).
			SetIsUnsafe(record.IsUnsafe).
			SetTriggeredRules(triggered).
			SetCreatedAt(createdAt)
		if len(storedBody) > 0 {
			builder.SetRequestBody(storedBody)
		}
		if len(record.SingGuardResponse) > 0 {
			builder.SetSingguardResponse(string(record.SingGuardResponse))
		}
		setSecurityCheckOptionalStrings(builder, record)
		if record.Metadata.UserID > 0 {
			builder.SetUserID(record.Metadata.UserID)
		}
		if record.Metadata.APIKeyID > 0 {
			builder.SetAPIKeyID(record.Metadata.APIKeyID)
		}
		if record.Metadata.GroupID > 0 {
			builder.SetGroupID(record.Metadata.GroupID)
		}
		if record.Latency > 0 {
			builder.SetLatencyMs(int(record.Latency / time.Millisecond))
		}
		if record.SingGuardLatency > 0 {
			builder.SetSingguardLatencyMs(int(record.SingGuardLatency / time.Millisecond))
		}
		if record.QueueDelay > 0 {
			builder.SetQueueDelayMs(int(record.QueueDelay / time.Millisecond))
		}
		if record.ExceptionType != "" {
			builder.SetExceptionType(record.ExceptionType)
		}
		if record.ExceptionMessage != "" {
			builder.SetExceptionMessage(record.ExceptionMessage)
		}
		builders = append(builders, builder)
	}
	return r.client.SecurityCheckLog.CreateBulk(builders...).OnConflict(
		entsql.ConflictColumns(securitychecklog.FieldEventID),
		entsql.ResolveWithIgnore(),
	).Exec(ctx)
}

func setSecurityCheckOptionalStrings(builder *dbent.SecurityCheckLogCreate, record service.SecurityCheckLogRecord) {
	if record.Metadata.RequestID != "" {
		builder.SetRequestID(record.Metadata.RequestID)
	}
	if record.Metadata.ClientRequestID != "" {
		builder.SetClientRequestID(record.Metadata.ClientRequestID)
	}
	if record.Metadata.APIKeyName != "" {
		builder.SetAPIKeyName(record.Metadata.APIKeyName)
	}
	if record.Metadata.GroupName != "" {
		builder.SetGroupName(record.Metadata.GroupName)
	}
	if record.Metadata.Model != "" {
		builder.SetModel(record.Metadata.Model)
	}
	if record.Metadata.Provider != "" {
		builder.SetProvider(record.Metadata.Provider)
	}
	if record.Metadata.Protocol != "" {
		builder.SetProtocol(record.Metadata.Protocol)
	}
	if record.Metadata.Endpoint != "" {
		builder.SetEndpoint(record.Metadata.Endpoint)
	}
}

func (r *groupRepository) ListSecurityCheckLogs(ctx context.Context, filter service.SecurityCheckLogFilter) (service.SecurityCheckLogPage, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	predicates := make([]predicate.SecurityCheckLog, 0, 4)
	if !filter.From.IsZero() {
		predicates = append(predicates, securitychecklog.CreatedAtGTE(filter.From))
	}
	if !filter.To.IsZero() {
		predicates = append(predicates, securitychecklog.CreatedAtLT(filter.To))
	}
	if filter.GroupID > 0 {
		predicates = append(predicates, securitychecklog.GroupIDEQ(filter.GroupID))
	}
	if filter.Decision != "" {
		predicates = append(predicates, securitychecklog.DecisionEQ(filter.Decision))
	}
	if filter.Status != "" {
		predicates = append(predicates, securitychecklog.CheckStatusEQ(filter.Status))
	}
	query := r.client.SecurityCheckLog.Query()
	if len(predicates) > 0 {
		query = query.Where(predicates...)
	}
	total, err := query.Count(ctx)
	if err != nil {
		return service.SecurityCheckLogPage{}, err
	}
	entities, err := query.Select(
		securitychecklog.FieldID, securitychecklog.FieldEventID, securitychecklog.FieldRequestID,
		securitychecklog.FieldGroupID, securitychecklog.FieldGroupName, securitychecklog.FieldModel,
		securitychecklog.FieldProtocol, securitychecklog.FieldCheckStatus, securitychecklog.FieldDecision,
		securitychecklog.FieldIsUnsafe, securitychecklog.FieldTriggeredRules, securitychecklog.FieldLatencyMs,
		securitychecklog.FieldCreatedAt, securitychecklog.FieldRequestBodyOriginalBytes,
		securitychecklog.FieldRequestBodyStoredBytes, securitychecklog.FieldRequestBodyTruncated,
	).Order(securitychecklog.ByCreatedAt(entsql.OrderDesc())).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return service.SecurityCheckLogPage{}, err
	}
	items := make([]service.SecurityCheckLogSummary, 0, len(entities))
	for _, entity := range entities {
		items = append(items, securityLogSummary(entity))
	}
	return service.SecurityCheckLogPage{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (r *groupRepository) GetSecurityCheckLog(ctx context.Context, id int64) (service.SecurityCheckLogDetail, error) {
	entity, err := r.client.SecurityCheckLog.Get(ctx, id)
	if err != nil {
		return service.SecurityCheckLogDetail{}, err
	}
	detail := service.SecurityCheckLogDetail{SecurityCheckLogSummary: securityLogSummary(entity), ConfigVersion: entity.ConfigVersion, RulesSnapshot: entity.RulesSnapshot, ExceptionType: entity.ExceptionType, ExceptionMessage: entity.ExceptionMessage, SingGuardLatencyMs: entity.SingguardLatencyMs, QueueDelayMs: entity.QueueDelayMs}
	if entity.SingguardResponse != nil {
		detail.SingGuardResponse = *entity.SingguardResponse
	}
	if len(entity.RequestBody) > 0 {
		reader, err := gzip.NewReader(bytes.NewReader(entity.RequestBody))
		if err == nil {
			body, readErr := io.ReadAll(reader)
			detail.RequestBody = string(body)
			err = readErr
			_ = reader.Close()
		}
		if err != nil {
			return service.SecurityCheckLogDetail{}, err
		}
	}
	return detail, nil
}

func (r *groupRepository) DeleteSecurityCheckLogsBefore(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	ids, err := r.client.SecurityCheckLog.Query().Where(securitychecklog.CreatedAtLT(before)).Limit(limit).IDs(ctx)
	if err != nil || len(ids) == 0 {
		return len(ids), err
	}
	deleted, err := r.client.SecurityCheckLog.Delete().Where(securitychecklog.IDIn(ids...)).Exec(ctx)
	return deleted, err
}

func securityLogSummary(entity *dbent.SecurityCheckLog) service.SecurityCheckLogSummary {
	return service.SecurityCheckLogSummary{
		ID: entity.ID, EventID: entity.EventID, RequestID: entity.RequestID, GroupID: entity.GroupID,
		GroupName: entity.GroupName, Model: entity.Model, Protocol: entity.Protocol,
		CheckStatus: entity.CheckStatus, Decision: entity.Decision, IsUnsafe: entity.IsUnsafe,
		TriggeredRules: entity.TriggeredRules, LatencyMs: entity.LatencyMs, CreatedAt: entity.CreatedAt,
		RequestBodyOriginalBytes: entity.RequestBodyOriginalBytes, RequestBodyStoredBytes: entity.RequestBodyStoredBytes,
		RequestBodyTruncated: entity.RequestBodyTruncated,
	}
}

var _ service.SecurityCheckLogRepository = (*groupRepository)(nil)
var _ service.SecurityCheckLogStore = (*groupRepository)(nil)
