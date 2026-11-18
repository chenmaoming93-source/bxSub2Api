package growth

import (
	"context"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthbudget"
	"github.com/Wei-Shaw/sub2api/ent/growthbudgetentry"
	"github.com/Wei-Shaw/sub2api/ent/growthtask"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	BudgetEntryTopUp    = "TOPUP"
	BudgetEntryReduce   = "REDUCE"
	BudgetEntryAllocate = "ALLOCATE"
	BudgetEntryRevoke   = "REVOKE"
)

var (
	ErrBudgetAllocatedLocked = infraerrors.Conflict("BUDGET_ALLOCATED_LOCKED", "budget allocated amount is locked by tasks")
	ErrBudgetInsufficient    = infraerrors.Conflict("BUDGET_INSUFFICIENT", "unallocated budget is insufficient")
	ErrTaskQuotaInvalid      = infraerrors.Conflict("TASK_QUOTA_INVALID", "task quota cannot be revoked")
	ErrGrowthTaskNotFound    = infraerrors.NotFound("TASK_NOT_FOUND", "growth task not found")
)

type BudgetAccount struct {
	TotalAmount       int64     `json:"total_amount"`
	AllocatedAmount   int64     `json:"allocated_amount"`
	DistributedAmount int64     `json:"distributed_amount"`
	UnallocatedAmount int64     `json:"unallocated_amount"`
	DistributedRate   float64   `json:"distributed_rate"`
	Version           int64     `json:"version"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type BudgetEntry struct {
	ID               int64     `json:"id"`
	EntryType        string    `json:"entry_type"`
	Points           int64     `json:"points"`
	BudgetAfter      int64     `json:"budget_after"`
	AllocatedAfter   int64     `json:"allocated_after"`
	DistributedAfter int64     `json:"distributed_after"`
	TaskID           *int64    `json:"task_id,omitempty"`
	TaskTitle        string    `json:"task_title,omitempty"`
	OperatorID       int64     `json:"operator_id"`
	Reason           string    `json:"reason"`
	RequestID        string    `json:"request_id"`
	CreatedAt        time.Time `json:"created_at"`
}

type TaskAllocation struct {
	TaskID            int64     `json:"task_id"`
	TaskCode          string    `json:"task_code"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	AllocatedPoints   int64     `json:"allocated_points"`
	DistributedPoints int64     `json:"distributed_points"`
	RemainingPoints   int64     `json:"remaining_points"`
	UsageRate         float64   `json:"usage_rate"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type BudgetListFilter struct {
	EntryType string
	TaskID    int64
	Status    string
	Search    string
	Page      int
	PageSize  int
}

type BudgetMutation struct {
	OperatorID int64
	Points     int64
	Reason     string
	RequestID  string
}

type BudgetService struct {
	client *ent.Client
}

func NewBudgetService(client *ent.Client) *BudgetService {
	return &BudgetService{client: client}
}

func (s *BudgetService) GetAccount(ctx context.Context) (BudgetAccount, error) {
	if s == nil || s.client == nil {
		return BudgetAccount{}, infraerrors.InternalServer("GROWTH_UNAVAILABLE", "growth service unavailable")
	}
	entity, err := s.client.GrowthBudget.Query().
		Where(growthbudget.ID(1)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return accountFromValues(0, 0, 0, 1, time.Time{}), nil
	}
	if err != nil {
		return BudgetAccount{}, err
	}
	return accountFromEntity(entity), nil
}

func (s *BudgetService) TopUp(ctx context.Context, mutation BudgetMutation) (BudgetAccount, error) {
	return s.changeTotal(ctx, BudgetEntryTopUp, mutation)
}

func (s *BudgetService) Reduce(ctx context.Context, mutation BudgetMutation) (BudgetAccount, error) {
	return s.changeTotal(ctx, BudgetEntryReduce, mutation)
}

func (s *BudgetService) changeTotal(ctx context.Context, entryType string, mutation BudgetMutation) (BudgetAccount, error) {
	if err := validateBudgetMutation(mutation); err != nil {
		return BudgetAccount{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return BudgetAccount{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	budget, err := ensureBudgetForUpdate(ctx, tx)
	if err != nil {
		return BudgetAccount{}, err
	}
	before := accountFromEntity(budget)
	if mutation.Points > 0 && budget.TotalAmount > math.MaxInt64-mutation.Points {
		return BudgetAccount{}, infraerrors.BadRequest("BUDGET_OVERFLOW", "budget amount is too large")
	}
	nextTotal := budget.TotalAmount + mutation.Points
	if entryType == BudgetEntryReduce {
		if mutation.Points > budget.TotalAmount {
			return BudgetAccount{}, ErrBudgetAllocatedLocked
		}
		nextTotal = budget.TotalAmount - mutation.Points
		if nextTotal < budget.AllocatedAmount {
			return BudgetAccount{}, ErrBudgetAllocatedLocked
		}
	}

	updated, err := tx.GrowthBudget.UpdateOneID(1).
		SetTotalAmount(nextTotal).
		SetVersion(budget.Version + 1).
		Save(ctx)
	if err != nil {
		return BudgetAccount{}, err
	}
	if err := appendBudgetEntry(ctx, tx, entryType, mutation, updated, nil); err != nil {
		return BudgetAccount{}, err
	}
	if err := appendBudgetAudit(ctx, tx, mutation, before, updated, nil, "budget."+strings.ToLower(entryType)); err != nil {
		return BudgetAccount{}, err
	}
	if err := tx.Commit(); err != nil {
		return BudgetAccount{}, err
	}
	committed = true
	return accountFromEntity(updated), nil
}

func (s *BudgetService) Allocate(ctx context.Context, taskID int64, mutation BudgetMutation) (BudgetAccount, error) {
	if taskID <= 0 {
		return BudgetAccount{}, infraerrors.BadRequest("INVALID_TASK_ID", "task_id must be positive")
	}
	if err := validateBudgetMutation(mutation); err != nil {
		return BudgetAccount{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return BudgetAccount{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	budget, err := ensureBudgetForUpdate(ctx, tx)
	if err != nil {
		return BudgetAccount{}, err
	}
	task, err := tx.GrowthTask.Query().
		Where(growthtask.ID(taskID)).
		ForUpdate().
		Only(ctx)
	if ent.IsNotFound(err) {
		return BudgetAccount{}, ErrGrowthTaskNotFound
	}
	if err != nil {
		return BudgetAccount{}, err
	}
	unallocated := budget.TotalAmount - budget.AllocatedAmount
	if mutation.Points > unallocated {
		return BudgetAccount{}, ErrBudgetInsufficient
	}
	if task.AllocatedPoints > math.MaxInt64-mutation.Points {
		return BudgetAccount{}, infraerrors.BadRequest("BUDGET_OVERFLOW", "task quota is too large")
	}
	beforeBudget := accountFromEntity(budget)
	beforeTask := taskAllocationFromEntity(task)
	if _, err := tx.GrowthTask.UpdateOneID(taskID).
		AddAllocatedPoints(mutation.Points).
		Save(ctx); err != nil {
		return BudgetAccount{}, err
	}
	updated, err := tx.GrowthBudget.UpdateOneID(1).
		SetAllocatedAmount(budget.AllocatedAmount + mutation.Points).
		SetVersion(budget.Version + 1).
		Save(ctx)
	if err != nil {
		return BudgetAccount{}, err
	}
	task.AllocatedPoints += mutation.Points
	if err := appendBudgetEntry(ctx, tx, BudgetEntryAllocate, mutation, updated, &taskID); err != nil {
		return BudgetAccount{}, err
	}
	if err := appendBudgetAudit(ctx, tx, mutation, beforeBudget, updated, &beforeTask, "budget.allocate"); err != nil {
		return BudgetAccount{}, err
	}
	if err := tx.Commit(); err != nil {
		return BudgetAccount{}, err
	}
	committed = true
	return accountFromEntity(updated), nil
}

func (s *BudgetService) Revoke(ctx context.Context, taskID int64, mutation BudgetMutation) (BudgetAccount, error) {
	if taskID <= 0 {
		return BudgetAccount{}, infraerrors.BadRequest("INVALID_TASK_ID", "task_id must be positive")
	}
	if err := validateBudgetMutation(mutation); err != nil {
		return BudgetAccount{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return BudgetAccount{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	budget, err := ensureBudgetForUpdate(ctx, tx)
	if err != nil {
		return BudgetAccount{}, err
	}
	task, err := tx.GrowthTask.Query().
		Where(growthtask.ID(taskID)).
		ForUpdate().
		Only(ctx)
	if ent.IsNotFound(err) {
		return BudgetAccount{}, ErrGrowthTaskNotFound
	}
	if err != nil {
		return BudgetAccount{}, err
	}
	remaining := task.AllocatedPoints - task.DistributedPoints
	if task.Status != "OFFLINE" || mutation.Points > remaining {
		return BudgetAccount{}, ErrTaskQuotaInvalid
	}
	if mutation.Points > budget.AllocatedAmount {
		return BudgetAccount{}, ErrTaskQuotaInvalid
	}
	beforeBudget := accountFromEntity(budget)
	beforeTask := taskAllocationFromEntity(task)
	if _, err := tx.GrowthTask.UpdateOneID(taskID).
		SetAllocatedPoints(task.AllocatedPoints - mutation.Points).
		Save(ctx); err != nil {
		return BudgetAccount{}, err
	}
	updated, err := tx.GrowthBudget.UpdateOneID(1).
		SetAllocatedAmount(budget.AllocatedAmount - mutation.Points).
		SetVersion(budget.Version + 1).
		Save(ctx)
	if err != nil {
		return BudgetAccount{}, err
	}
	if err := appendBudgetEntry(ctx, tx, BudgetEntryRevoke, mutation, updated, &taskID); err != nil {
		return BudgetAccount{}, err
	}
	if err := appendBudgetAudit(ctx, tx, mutation, beforeBudget, updated, &beforeTask, "budget.revoke"); err != nil {
		return BudgetAccount{}, err
	}
	if err := tx.Commit(); err != nil {
		return BudgetAccount{}, err
	}
	committed = true
	return accountFromEntity(updated), nil
}

func (s *BudgetService) ListEntries(ctx context.Context, filter BudgetListFilter) ([]BudgetEntry, int64, error) {
	filter = normalizeBudgetFilter(filter)
	query := s.client.GrowthBudgetEntry.Query()
	if filter.EntryType != "" {
		query = query.Where(growthbudgetentry.EntryTypeEQ(filter.EntryType))
	}
	if filter.TaskID > 0 {
		query = query.Where(growthbudgetentry.TaskIDEQ(filter.TaskID))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	entities, err := query.
		Order(ent.Desc(growthbudgetentry.FieldID)).
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	taskTitles, err := s.taskTitleMap(ctx, entities)
	if err != nil {
		return nil, 0, err
	}
	result := make([]BudgetEntry, 0, len(entities))
	for _, entity := range entities {
		item := BudgetEntry{
			ID:               entity.ID,
			EntryType:        entity.EntryType,
			Points:           entity.Points,
			BudgetAfter:      entity.BudgetAfter,
			AllocatedAfter:   entity.AllocatedAfter,
			DistributedAfter: entity.DistributedAfter,
			OperatorID:       entity.OperatorID,
			Reason:           entity.Reason,
			RequestID:        entity.RequestID,
			CreatedAt:        entity.CreatedAt,
		}
		if entity.TaskID != nil {
			value := *entity.TaskID
			item.TaskID = &value
			item.TaskTitle = taskTitles[value]
		}
		result = append(result, item)
	}
	return result, int64(total), nil
}

func (s *BudgetService) ListAllocations(ctx context.Context, filter BudgetListFilter) ([]TaskAllocation, int64, error) {
	filter = normalizeBudgetFilter(filter)
	query := s.client.GrowthTask.Query()
	if filter.Status != "" {
		query = query.Where(growthtask.StatusEQ(filter.Status))
	}
	if filter.Search != "" {
		query = query.Where(growthtask.Or(
			growthtask.TaskCodeContainsFold(filter.Search),
			growthtask.TitleContainsFold(filter.Search),
		))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	entities, err := query.
		Order(ent.Asc(growthtask.FieldSortOrder), ent.Desc(growthtask.FieldID)).
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]TaskAllocation, 0, len(entities))
	for _, entity := range entities {
		result = append(result, taskAllocationFromEntity(entity))
	}
	return result, int64(total), nil
}

func ensureBudgetForUpdate(ctx context.Context, tx *ent.Tx) (*ent.GrowthBudget, error) {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO growth_budgets (id, total_amount, allocated_amount, distributed_amount, version, created_at, updated_at)
VALUES (1, 0, 0, 0, 1, CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE id = id`); err != nil {
		return nil, err
	}
	return tx.GrowthBudget.Query().Where(growthbudget.ID(1)).ForUpdate().Only(ctx)
}

func appendBudgetEntry(
	ctx context.Context,
	tx *ent.Tx,
	entryType string,
	mutation BudgetMutation,
	budget *ent.GrowthBudget,
	taskID *int64,
) error {
	create := tx.GrowthBudgetEntry.Create().
		SetBudgetID(1).
		SetEntryType(entryType).
		SetPoints(mutation.Points).
		SetBudgetAfter(budget.TotalAmount).
		SetAllocatedAfter(budget.AllocatedAmount).
		SetDistributedAfter(budget.DistributedAmount).
		SetOperatorID(mutation.OperatorID).
		SetReason(strings.TrimSpace(mutation.Reason)).
		SetRequestID(strings.TrimSpace(mutation.RequestID))
	if taskID != nil {
		create.SetTaskID(*taskID)
	}
	_, err := create.Save(ctx)
	return err
}

func appendBudgetAudit(
	ctx context.Context,
	tx *ent.Tx,
	mutation BudgetMutation,
	before BudgetAccount,
	after *ent.GrowthBudget,
	task *TaskAllocation,
	action string,
) error {
	resourceID := int64(1)
	resourceType := "growth_budget"
	if task != nil {
		resourceID = task.TaskID
		resourceType = "growth_task"
	}
	beforeJSON := map[string]any{
		"total_amount":       before.TotalAmount,
		"allocated_amount":   before.AllocatedAmount,
		"distributed_amount": before.DistributedAmount,
	}
	afterJSON := map[string]any{
		"total_amount":       after.TotalAmount,
		"allocated_amount":   after.AllocatedAmount,
		"distributed_amount": after.DistributedAmount,
	}
	if task != nil {
		beforeJSON["task"] = task
	}
	_, err := tx.GrowthAdminAudit.Create().
		SetOperatorID(mutation.OperatorID).
		SetAction(action).
		SetResourceType(resourceType).
		SetResourceID(resourceID).
		SetBeforeJSON(beforeJSON).
		SetAfterJSON(afterJSON).
		SetReason(strings.TrimSpace(mutation.Reason)).
		SetRequestID(strings.TrimSpace(mutation.RequestID)).
		Save(ctx)
	return err
}

func validateBudgetMutation(mutation BudgetMutation) error {
	if mutation.OperatorID <= 0 {
		return infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if mutation.Points <= 0 {
		return infraerrors.BadRequest("INVALID_POINTS", "points must be positive")
	}
	if utf8.RuneCountInString(strings.TrimSpace(mutation.Reason)) < 5 {
		return infraerrors.BadRequest("REASON_REQUIRED", "reason must contain at least 5 characters")
	}
	if len(mutation.RequestID) > 128 {
		return infraerrors.BadRequest("INVALID_REQUEST_ID", "request_id is too long")
	}
	return nil
}

func accountFromEntity(entity *ent.GrowthBudget) BudgetAccount {
	return accountFromValues(
		entity.TotalAmount,
		entity.AllocatedAmount,
		entity.DistributedAmount,
		entity.Version,
		entity.UpdatedAt,
	)
}

func accountFromValues(total, allocated, distributed, version int64, updatedAt time.Time) BudgetAccount {
	account := BudgetAccount{
		TotalAmount:       total,
		AllocatedAmount:   allocated,
		DistributedAmount: distributed,
		UnallocatedAmount: total - allocated,
		Version:           version,
		UpdatedAt:         updatedAt,
	}
	if total > 0 {
		account.DistributedRate = float64(distributed) / float64(total)
	}
	return account
}

func taskAllocationFromEntity(entity *ent.GrowthTask) TaskAllocation {
	remaining := entity.AllocatedPoints - entity.DistributedPoints
	result := TaskAllocation{
		TaskID:            entity.ID,
		TaskCode:          entity.TaskCode,
		Title:             entity.Title,
		Status:            entity.Status,
		AllocatedPoints:   entity.AllocatedPoints,
		DistributedPoints: entity.DistributedPoints,
		RemainingPoints:   remaining,
		UpdatedAt:         entity.UpdatedAt,
	}
	if entity.AllocatedPoints > 0 {
		result.UsageRate = float64(entity.DistributedPoints) / float64(entity.AllocatedPoints)
	}
	return result
}

func normalizeBudgetFilter(filter BudgetListFilter) BudgetListFilter {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	filter.EntryType = strings.TrimSpace(filter.EntryType)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Search = strings.TrimSpace(filter.Search)
	return filter
}

func (s *BudgetService) taskTitleMap(ctx context.Context, entries []*ent.GrowthBudgetEntry) (map[int64]string, error) {
	ids := make([]int64, 0, len(entries))
	seen := make(map[int64]bool, len(entries))
	for _, entry := range entries {
		if entry.TaskID != nil && !seen[*entry.TaskID] {
			seen[*entry.TaskID] = true
			ids = append(ids, *entry.TaskID)
		}
	}
	result := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	tasks, err := s.client.GrowthTask.Query().
		Where(growthtask.IDIn(ids...)).
		Select(growthtask.FieldID, growthtask.FieldTitle).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		result[task.ID] = task.Title
	}
	return result, nil
}
