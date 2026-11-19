package growth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthtask"
	"github.com/Wei-Shaw/sub2api/ent/growthtaskrule"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

var (
	ErrTaskCodeDuplicated = infraerrors.Conflict("TASK_CODE_DUPLICATED", "task_code already exists")
	ErrTaskStateInvalid   = infraerrors.Conflict("TASK_STATE_INVALID", "task state does not allow this operation")
	ErrTaskRuleMissing    = infraerrors.NotFound("TASK_RULE_MISSING", "task rule is missing")
)

var taskCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

type TaskRuleInput struct {
	TriggerEvent  string         `json:"trigger_event"`
	Conditions    map[string]any `json:"conditions"`
	Target        int64          `json:"target"`
	RewardPoints  int64          `json:"reward_points"`
	PeriodType    string         `json:"period_type"`
	DailyLimit    int            `json:"daily_limit"`
	PerUserLimit  int            `json:"per_user_limit"`
	EffectiveFrom *time.Time     `json:"effective_from,omitempty"`
}

type TaskCreateInput struct {
	OperatorID        int64         `json:"-"`
	TaskCode          string        `json:"task_code"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Category          string        `json:"category"`
	TaskType          string        `json:"task_type"`
	Icon              string        `json:"icon"`
	SortOrder         int           `json:"sort_order"`
	InitialAllocation int64         `json:"initial_allocation"`
	Rule              TaskRuleInput `json:"rule"`
	RequestID         string        `json:"request_id"`
}

type TaskUpdateInput struct {
	OperatorID  int64         `json:"-"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	TaskType    string        `json:"task_type"`
	Icon        string        `json:"icon"`
	SortOrder   int           `json:"sort_order"`
	Rule        TaskRuleInput `json:"rule"`
	RequestID   string        `json:"request_id"`
}

type TaskListView struct {
	TaskID             int64      `json:"task_id"`
	TaskCode           string     `json:"task_code"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Category           string     `json:"category"`
	TaskType           string     `json:"task_type"`
	Icon               string     `json:"icon"`
	SortOrder          int        `json:"sort_order"`
	Status             string     `json:"status"`
	AllocatedPoints    int64      `json:"allocated_points"`
	DistributedPoints  int64      `json:"distributed_points"`
	RemainingPoints    int64      `json:"remaining_points"`
	UsageRate          float64    `json:"usage_rate"`
	RuleID             *int64     `json:"rule_id,omitempty"`
	RuleVersion        *int       `json:"rule_version,omitempty"`
	PendingRuleID      *int64     `json:"pending_rule_id,omitempty"`
	PendingRuleVersion *int       `json:"pending_rule_version,omitempty"`
	RewardPoints       int64      `json:"reward_points"`
	PeriodType         string     `json:"period_type"`
	DailyLimit         int        `json:"daily_limit"`
	PerUserLimit       int        `json:"per_user_limit"`
	EffectiveFrom      *time.Time `json:"effective_from,omitempty"`
	EffectiveTo        *time.Time `json:"effective_to,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type TaskRuleView struct {
	ID            int64          `json:"id"`
	Version       int            `json:"version"`
	TriggerEvent  string         `json:"trigger_event"`
	Conditions    map[string]any `json:"conditions,omitempty"`
	Target        int64          `json:"target"`
	RewardPoints  int64          `json:"reward_points"`
	PeriodType    string         `json:"period_type"`
	DailyLimit    int            `json:"daily_limit"`
	PerUserLimit  int            `json:"per_user_limit"`
	CreatedBy     int64          `json:"created_by"`
	EffectiveFrom time.Time      `json:"effective_from"`
	EffectiveTo   *time.Time     `json:"effective_to,omitempty"`
	PublishedAt   *time.Time     `json:"published_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type TaskService struct {
	client *ent.Client
}

func NewTaskService(client *ent.Client) *TaskService {
	return &TaskService{client: client}
}

func (s *TaskService) Create(ctx context.Context, input TaskCreateInput) (TaskListView, error) {
	if err := validateTaskCreate(input); err != nil {
		return TaskListView{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	task, err := tx.GrowthTask.Create().
		SetTaskCode(strings.TrimSpace(input.TaskCode)).
		SetTitle(strings.TrimSpace(input.Title)).
		SetRemark(strings.TrimSpace(input.Description)).
		SetCategory(input.Category).
		SetTaskType(input.TaskType).
		SetIcon(strings.TrimSpace(input.Icon)).
		SetSortOrder(input.SortOrder).
		SetTaskStatus("DRAFT").
		Save(ctx)
	if err != nil {
		if isDuplicateKey(err) {
			return TaskListView{}, ErrTaskCodeDuplicated
		}
		return TaskListView{}, err
	}

	effectiveFrom := input.Rule.EffectiveFrom
	if effectiveFrom == nil {
		now := time.Now()
		effectiveFrom = &now
	}
	rule, err := createTaskRule(ctx, tx, task.ID, 1, input.OperatorID, input.Rule, effectiveFrom)
	if err != nil {
		return TaskListView{}, err
	}

	budget, err := ensureBudgetForUpdate(ctx, tx)
	if err != nil {
		return TaskListView{}, err
	}
	if input.InitialAllocation > budget.TotalAmount-budget.AllocatedAmount {
		return TaskListView{}, ErrBudgetInsufficient
	}
	beforeBudget := accountFromEntity(budget)
	task, err = tx.GrowthTask.UpdateOneID(task.ID).
		SetAllocatedPoints(input.InitialAllocation).
		Save(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	updatedBudget, err := tx.GrowthBudget.UpdateOneID(1).
		SetAllocatedAmount(budget.AllocatedAmount + input.InitialAllocation).
		SetVersion(budget.Version + 1).
		Save(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	mutation := BudgetMutation{
		OperatorID: input.OperatorID,
		Points:     input.InitialAllocation,
		Reason:     "创建任务初始额度",
		RequestID:  input.RequestID,
	}
	if err := appendBudgetEntry(ctx, tx, BudgetEntryAllocate, mutation, updatedBudget, &task.ID); err != nil {
		return TaskListView{}, err
	}
	if err := appendBudgetAudit(ctx, tx, mutation, beforeBudget, updatedBudget, nil, "task.initial-allocation"); err != nil {
		return TaskListView{}, err
	}
	if err := appendTaskAudit(ctx, tx, input.OperatorID, "task.create", "创建任务", task.ID, nil, task, input.RequestID); err != nil {
		return TaskListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return TaskListView{}, err
	}
	committed = true
	return buildTaskView(task, nil, rule), nil
}

func (s *TaskService) Update(ctx context.Context, taskID int64, input TaskUpdateInput) (TaskListView, error) {
	if taskID <= 0 {
		return TaskListView{}, infraerrors.BadRequest("INVALID_TASK_ID", "task_id must be positive")
	}
	if input.OperatorID <= 0 {
		return TaskListView{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if err := validateTaskRule(input.Rule); err != nil {
		return TaskListView{}, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return TaskListView{}, infraerrors.BadRequest("TITLE_REQUIRED", "title is required")
	}
	if !isTaskCategory(input.Category) || !isTaskCategory(input.TaskType) {
		return TaskListView{}, infraerrors.BadRequest("INVALID_TASK", "category and task_type are invalid")
	}
	if err := validateTaskTextFields("", input.Title, input.Description, input.Icon); err != nil {
		return TaskListView{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	task, err := tx.GrowthTask.Query().Where(growthtask.ID(taskID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return TaskListView{}, ErrGrowthTaskNotFound
	}
	if err != nil {
		return TaskListView{}, err
	}
	before := *task
	task, err = tx.GrowthTask.UpdateOneID(taskID).
		SetTitle(strings.TrimSpace(input.Title)).
		SetRemark(strings.TrimSpace(input.Description)).
		SetCategory(input.Category).
		SetTaskType(input.TaskType).
		SetIcon(strings.TrimSpace(input.Icon)).
		SetSortOrder(input.SortOrder).
		Save(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	draft, err := latestDraftRuleTx(ctx, tx, taskID)
	if ent.IsNotFound(err) {
		nextVersion := 1
		latestVersion, versionErr := latestRuleVersion(ctx, tx, taskID)
		if versionErr != nil {
			return TaskListView{}, versionErr
		}
		if latestVersion > 0 {
			nextVersion = latestVersion + 1
		}
		effectiveFrom := input.Rule.EffectiveFrom
		if effectiveFrom == nil {
			now := time.Now()
			effectiveFrom = &now
		}
		draft, err = createTaskRule(ctx, tx, taskID, nextVersion, input.OperatorID, input.Rule, effectiveFrom)
	} else if err == nil {
		update := tx.GrowthTaskRule.UpdateOneID(draft.ID).
			SetTriggerEvent(defaultTriggerEvent(input.Rule.TriggerEvent)).
			SetTarget(input.Rule.Target).
			SetRewardPoints(input.Rule.RewardPoints).
			SetPeriodType(input.Rule.PeriodType).
			SetDailyLimit(input.Rule.DailyLimit).
			SetPerUserLimit(input.Rule.PerUserLimit)
		update.SetConditions(encodeJSONText(input.Rule.Conditions))
		if input.Rule.EffectiveFrom != nil {
			update.SetEffectiveFrom(*input.Rule.EffectiveFrom)
		}
		draft, err = update.Save(ctx)
	}
	if err != nil {
		return TaskListView{}, err
	}
	if err := appendTaskAudit(ctx, tx, input.OperatorID, "task.update", "更新任务规则", taskID, &before, task, input.RequestID); err != nil {
		return TaskListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return TaskListView{}, err
	}
	committed = true
	current, _ := currentPublishedRule(ctx, s.client, taskID)
	return buildTaskView(task, current, draft), nil
}

func (s *TaskService) Publish(ctx context.Context, taskID int64, operatorID int64, effectiveFrom *time.Time, requestID string) (TaskListView, error) {
	if operatorID <= 0 {
		return TaskListView{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	task, err := tx.GrowthTask.Query().Where(growthtask.ID(taskID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return TaskListView{}, ErrGrowthTaskNotFound
	}
	if err != nil {
		return TaskListView{}, err
	}
	before := *task
	draft, err := latestDraftRuleTx(ctx, tx, taskID)
	if ent.IsNotFound(err) {
		if task.TaskStatus == "ACTIVE" {
			current, currentErr := currentPublishedRuleTx(ctx, tx, taskID)
			if currentErr == nil {
				_ = tx.Rollback()
				committed = true
				return buildTaskView(task, current, nil), nil
			}
		}
		return TaskListView{}, ErrTaskRuleMissing
	}
	if err != nil {
		return TaskListView{}, err
	}
	now := time.Now()
	start := now
	if effectiveFrom != nil && effectiveFrom.After(now) {
		start = *effectiveFrom
	}
	if _, err := tx.GrowthTaskRule.Update().
		Where(
			growthtaskrule.TaskIDEQ(taskID),
			growthtaskrule.PublishedAtNotNil(),
			growthtaskrule.EffectiveToIsNil(),
		).
		SetEffectiveTo(now).
		Save(ctx); err != nil {
		return TaskListView{}, err
	}
	published, err := tx.GrowthTaskRule.UpdateOneID(draft.ID).
		SetPublishedAt(now).
		SetEffectiveFrom(start).
		ClearEffectiveTo().
		Save(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	nextStatus := "ACTIVE"
	if before.TaskStatus == "OFFLINE" {
		// Publishing a replacement rule keeps an offline task hidden until the
		// operator re-onlines it with a fresh quota allocation.
		nextStatus = "OFFLINE"
	}
	task, err = tx.GrowthTask.UpdateOneID(taskID).
		SetTaskStatus(nextStatus).
		SetEffectiveFrom(start).
		ClearEffectiveTo().
		Save(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	if err := appendTaskAudit(ctx, tx, operatorID, "task.publish", "发布任务规则", taskID, &before, task, requestID); err != nil {
		return TaskListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return TaskListView{}, err
	}
	committed = true
	return buildTaskView(task, published, nil), nil
}

func (s *TaskService) Offline(ctx context.Context, taskID int64, operatorID int64, reason string, revokeRemaining bool, requestID string) (TaskListView, error) {
	if operatorID <= 0 {
		return TaskListView{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if err := validateReason(reason); err != nil {
		return TaskListView{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	task, err := tx.GrowthTask.Query().Where(growthtask.ID(taskID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return TaskListView{}, ErrGrowthTaskNotFound
	}
	if err != nil {
		return TaskListView{}, err
	}
	if task.TaskStatus != "ACTIVE" {
		return TaskListView{}, ErrTaskStateInvalid
	}
	before := *task
	now := time.Now()
	if _, err := tx.GrowthTaskRule.Update().
		Where(
			growthtaskrule.TaskIDEQ(taskID),
			growthtaskrule.PublishedAtNotNil(),
			growthtaskrule.EffectiveToIsNil(),
		).
		SetEffectiveTo(now).
		Save(ctx); err != nil {
		return TaskListView{}, err
	}
	if revokeRemaining {
		remaining := task.AllocatedPoints - task.DistributedPoints
		if remaining > 0 {
			budget, err := ensureBudgetForUpdate(ctx, tx)
			if err != nil {
				return TaskListView{}, err
			}
			beforeBudget := accountFromEntity(budget)
			_, err = tx.GrowthTask.UpdateOneID(taskID).SetAllocatedPoints(task.DistributedPoints).Save(ctx)
			if err != nil {
				return TaskListView{}, err
			}
			updatedBudget, err := tx.GrowthBudget.UpdateOneID(1).
				SetAllocatedAmount(budget.AllocatedAmount - remaining).
				SetVersion(budget.Version + 1).
				Save(ctx)
			if err != nil {
				return TaskListView{}, err
			}
			mutation := BudgetMutation{OperatorID: operatorID, Points: remaining, Reason: strings.TrimSpace(reason), RequestID: requestID}
			if err := appendBudgetEntry(ctx, tx, BudgetEntryRevoke, mutation, updatedBudget, &taskID); err != nil {
				return TaskListView{}, err
			}
			if err := appendBudgetAudit(ctx, tx, mutation, beforeBudget, updatedBudget, nil, "task.offline.revoke"); err != nil {
				return TaskListView{}, err
			}
		}
	}
	task, err = tx.GrowthTask.UpdateOneID(taskID).SetTaskStatus("OFFLINE").SetEffectiveTo(now).Save(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	if err := appendTaskAudit(ctx, tx, operatorID, "task.offline", reason, taskID, &before, task, requestID); err != nil {
		return TaskListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return TaskListView{}, err
	}
	committed = true
	current, _ := currentPublishedRule(ctx, s.client, taskID)
	return buildTaskView(task, current, nil), nil
}

func (s *TaskService) Online(ctx context.Context, taskID int64, operatorID int64, initialAllocation int64, requestID string) (TaskListView, error) {
	if operatorID <= 0 {
		return TaskListView{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if initialAllocation <= 0 {
		return TaskListView{}, infraerrors.BadRequest("INITIAL_ALLOCATION_REQUIRED", "initial_allocation must be positive")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	task, err := tx.GrowthTask.Query().Where(growthtask.ID(taskID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return TaskListView{}, ErrGrowthTaskNotFound
	}
	if err != nil {
		return TaskListView{}, err
	}
	if task.TaskStatus != "OFFLINE" {
		return TaskListView{}, ErrTaskStateInvalid
	}
	now := time.Now()
	ruleCount, err := tx.GrowthTaskRule.Query().
		Where(
			growthtaskrule.TaskIDEQ(taskID),
			growthtaskrule.PublishedAtNotNil(),
			growthtaskrule.Or(
				growthtaskrule.EffectiveToIsNil(),
				growthtaskrule.EffectiveToGT(now),
			),
		).
		Count(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	if ruleCount == 0 {
		return TaskListView{}, ErrTaskRuleMissing
	}
	if initialAllocation < task.DistributedPoints {
		return TaskListView{}, infraerrors.BadRequest(
			"INITIAL_ALLOCATION_BELOW_DISTRIBUTED",
			"initial_allocation cannot be lower than points already distributed",
		)
	}
	budget, err := ensureBudgetForUpdate(ctx, tx)
	if err != nil {
		return TaskListView{}, err
	}
	allocationDelta := initialAllocation - task.AllocatedPoints
	if allocationDelta > budget.TotalAmount-budget.AllocatedAmount {
		return TaskListView{}, ErrBudgetInsufficient
	}
	before := *task
	beforeBudget := accountFromEntity(budget)
	task, err = tx.GrowthTask.UpdateOneID(taskID).
		SetAllocatedPoints(initialAllocation).
		SetTaskStatus("ACTIVE").
		SetEffectiveFrom(now).
		ClearEffectiveTo().
		Save(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	updatedBudget, err := tx.GrowthBudget.UpdateOneID(1).
		SetAllocatedAmount(budget.AllocatedAmount + allocationDelta).
		SetVersion(budget.Version + 1).
		Save(ctx)
	if err != nil {
		return TaskListView{}, err
	}
	if allocationDelta != 0 {
		entryType := BudgetEntryAllocate
		points := allocationDelta
		action := "task.online.allocate"
		if points < 0 {
			entryType = BudgetEntryRevoke
			points = -points
			action = "task.online.adjust"
		}
		mutation := BudgetMutation{
			OperatorID: operatorID,
			Points:     points,
			Reason:     "任务重新上线额度",
			RequestID:  requestID,
		}
		if err := appendBudgetEntry(ctx, tx, entryType, mutation, updatedBudget, &taskID); err != nil {
			return TaskListView{}, err
		}
		if err := appendBudgetAudit(ctx, tx, mutation, beforeBudget, updatedBudget, nil, action); err != nil {
			return TaskListView{}, err
		}
	}
	if err := appendTaskAudit(ctx, tx, operatorID, "task.online", "重新上线任务", taskID, &before, task, requestID); err != nil {
		return TaskListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return TaskListView{}, err
	}
	committed = true
	current, _ := currentPublishedRule(ctx, s.client, taskID)
	return buildTaskView(task, current, nil), nil
}

func (s *TaskService) List(ctx context.Context, filter BudgetListFilter, category string) ([]TaskListView, int64, error) {
	filter = normalizeBudgetFilter(filter)
	query := s.client.GrowthTask.Query()
	if filter.Status != "" {
		query = query.Where(growthtask.TaskStatusEQ(filter.Status))
	}
	if category != "" {
		query = query.Where(growthtask.CategoryEQ(category))
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
	tasks, err := query.
		Order(ent.Asc(growthtask.FieldSortOrder), ent.Desc(growthtask.FieldID)).
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	published, pending, err := s.ruleMaps(ctx, tasks)
	if err != nil {
		return nil, 0, err
	}
	result := make([]TaskListView, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, buildTaskView(task, published[task.ID], pending[task.ID]))
	}
	return result, int64(total), nil
}

func (s *TaskService) Get(ctx context.Context, taskID int64) (TaskListView, error) {
	task, err := s.client.GrowthTask.Get(ctx, taskID)
	if ent.IsNotFound(err) {
		return TaskListView{}, ErrGrowthTaskNotFound
	}
	if err != nil {
		return TaskListView{}, err
	}
	current, _ := currentPublishedRule(ctx, s.client, taskID)
	draft, _ := latestDraftRuleClient(ctx, s.client, taskID)
	return buildTaskView(task, current, draft), nil
}

func (s *TaskService) RuleVersions(ctx context.Context, taskID int64) ([]TaskRuleView, error) {
	rules, err := s.client.GrowthTaskRule.Query().
		Where(growthtaskrule.TaskIDEQ(taskID)).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]TaskRuleView, 0, len(rules))
	for _, rule := range rules {
		result = append(result, ruleView(rule))
	}
	return result, nil
}

func createTaskRule(ctx context.Context, tx *ent.Tx, taskID int64, version int, operatorID int64, input TaskRuleInput, effectiveFrom *time.Time) (*ent.GrowthTaskRule, error) {
	create := tx.GrowthTaskRule.Create().
		SetTaskID(taskID).
		SetVersion(version).
		SetTriggerEvent(defaultTriggerEvent(input.TriggerEvent)).
		SetTarget(input.Target).
		SetRewardPoints(input.RewardPoints).
		SetPeriodType(input.PeriodType).
		SetDailyLimit(input.DailyLimit).
		SetPerUserLimit(input.PerUserLimit).
		SetCreatedBy(operatorID).
		SetEffectiveFrom(*effectiveFrom)
	create.SetConditions(encodeJSONText(input.Conditions))
	return create.Save(ctx)
}

func latestDraftRuleTx(ctx context.Context, tx *ent.Tx, taskID int64) (*ent.GrowthTaskRule, error) {
	return tx.GrowthTaskRule.Query().
		Where(growthtaskrule.TaskIDEQ(taskID), growthtaskrule.PublishedAtIsNil()).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		First(ctx)
}

func latestDraftRuleClient(ctx context.Context, client *ent.Client, taskID int64) (*ent.GrowthTaskRule, error) {
	return client.GrowthTaskRule.Query().
		Where(growthtaskrule.TaskIDEQ(taskID), growthtaskrule.PublishedAtIsNil()).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		First(ctx)
}

func currentPublishedRule(ctx context.Context, client *ent.Client, taskID int64) (*ent.GrowthTaskRule, error) {
	return client.GrowthTaskRule.Query().
		Where(growthtaskrule.TaskIDEQ(taskID), growthtaskrule.PublishedAtNotNil()).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		First(ctx)
}

func currentPublishedRuleTx(ctx context.Context, tx *ent.Tx, taskID int64) (*ent.GrowthTaskRule, error) {
	return tx.GrowthTaskRule.Query().
		Where(growthtaskrule.TaskIDEQ(taskID), growthtaskrule.PublishedAtNotNil()).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		First(ctx)
}
func latestRuleVersion(ctx context.Context, tx *ent.Tx, taskID int64) (int, error) {
	rule, err := tx.GrowthTaskRule.Query().
		Where(growthtaskrule.TaskIDEQ(taskID)).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		First(ctx)
	if ent.IsNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return rule.Version, nil
}

func (s *TaskService) ruleMaps(ctx context.Context, tasks []*ent.GrowthTask) (map[int64]*ent.GrowthTaskRule, map[int64]*ent.GrowthTaskRule, error) {
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	published := make(map[int64]*ent.GrowthTaskRule)
	pending := make(map[int64]*ent.GrowthTaskRule)
	if len(ids) == 0 {
		return published, pending, nil
	}
	rules, err := s.client.GrowthTaskRule.Query().
		Where(growthtaskrule.TaskIDIn(ids...)).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, rule := range rules {
		if rule.PublishedAt != nil {
			if _, exists := published[rule.TaskID]; !exists {
				published[rule.TaskID] = rule
			}
		} else if _, exists := pending[rule.TaskID]; !exists {
			pending[rule.TaskID] = rule
		}
	}
	return published, pending, nil
}

func appendTaskAudit(ctx context.Context, tx *ent.Tx, operatorID int64, action, reason string, taskID int64, before, after *ent.GrowthTask, requestID string) error {
	beforeJSON := map[string]any{}
	if before != nil {
		beforeJSON = taskAuditSnapshot(before)
	}
	afterJSON := taskAuditSnapshot(after)
	_, err := tx.GrowthAdminAudit.Create().
		SetOperatorID(operatorID).
		SetActionType(action).
		SetResourceType("growth_task").
		SetResourceID(taskID).
		SetBeforeJSON(encodeJSONText(beforeJSON)).
		SetAfterJSON(encodeJSONText(afterJSON)).
		SetReason(strings.TrimSpace(reason)).
		SetRequestID(requestID).
		Save(ctx)
	return err
}

func taskAuditSnapshot(task *ent.GrowthTask) map[string]any {
	if task == nil {
		return map[string]any{}
	}
	return map[string]any{
		"task_id":            task.ID,
		"task_code":          task.TaskCode,
		"title":              task.Title,
		"status":             task.TaskStatus,
		"allocated_points":   task.AllocatedPoints,
		"distributed_points": task.DistributedPoints,
	}
}

func buildTaskView(task *ent.GrowthTask, current, pending *ent.GrowthTaskRule) TaskListView {
	view := TaskListView{
		TaskID:            task.ID,
		TaskCode:          task.TaskCode,
		Title:             task.Title,
		Description:       task.Remark,
		Category:          task.Category,
		TaskType:          task.TaskType,
		Icon:              task.Icon,
		SortOrder:         task.SortOrder,
		Status:            task.TaskStatus,
		AllocatedPoints:   task.AllocatedPoints,
		DistributedPoints: task.DistributedPoints,
		RemainingPoints:   task.AllocatedPoints - task.DistributedPoints,
		EffectiveFrom:     task.EffectiveFrom,
		EffectiveTo:       task.EffectiveTo,
		UpdatedAt:         task.UpdatedAt,
	}
	if task.AllocatedPoints > 0 {
		view.UsageRate = float64(task.DistributedPoints) / float64(task.AllocatedPoints)
	}
	if current != nil {
		id, version := current.ID, current.Version
		view.RuleID = &id
		view.RuleVersion = &version
		view.RewardPoints = current.RewardPoints
		view.PeriodType = current.PeriodType
		view.DailyLimit = current.DailyLimit
		view.PerUserLimit = current.PerUserLimit
	}
	if pending != nil {
		id, version := pending.ID, pending.Version
		view.PendingRuleID = &id
		view.PendingRuleVersion = &version
	}
	return view
}

func ruleView(rule *ent.GrowthTaskRule) TaskRuleView {
	return TaskRuleView{
		ID:            rule.ID,
		Version:       rule.Version,
		TriggerEvent:  rule.TriggerEvent,
		Conditions:    decodeJSONText(rule.Conditions),
		Target:        rule.Target,
		RewardPoints:  rule.RewardPoints,
		PeriodType:    rule.PeriodType,
		DailyLimit:    rule.DailyLimit,
		PerUserLimit:  rule.PerUserLimit,
		CreatedBy:     rule.CreatedBy,
		EffectiveFrom: rule.EffectiveFrom,
		EffectiveTo:   rule.EffectiveTo,
		PublishedAt:   rule.PublishedAt,
		CreatedAt:     rule.CreatedAt,
	}
}

func validateTaskCreate(input TaskCreateInput) error {
	if input.OperatorID <= 0 {
		return infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if strings.TrimSpace(input.TaskCode) == "" || strings.TrimSpace(input.Title) == "" {
		return infraerrors.BadRequest("INVALID_TASK", "task_code and title are required")
	}
	if err := validateTaskTextFields(input.TaskCode, input.Title, input.Description, input.Icon); err != nil {
		return err
	}
	if !isTaskCategory(input.Category) || !isTaskCategory(input.TaskType) {
		return infraerrors.BadRequest("INVALID_TASK", "category and task_type are invalid")
	}
	if input.InitialAllocation <= 0 {
		return infraerrors.BadRequest("INITIAL_ALLOCATION_REQUIRED", "initial_allocation must be positive")
	}
	return validateTaskRule(input.Rule)
}

func validateTaskRule(rule TaskRuleInput) error {
	if rule.RewardPoints <= 0 || rule.Target <= 0 {
		return infraerrors.BadRequest("INVALID_RULE", "reward_points and target must be positive")
	}
	if rule.DailyLimit < 0 || rule.PerUserLimit < 0 {
		return infraerrors.BadRequest("INVALID_RULE", "limits must be non-negative")
	}
	if rule.PeriodType != "DAILY" && rule.PeriodType != "ONCE" && rule.PeriodType != "WEEKLY" {
		return infraerrors.BadRequest("INVALID_RULE", "period_type is invalid")
	}
	if utf8.RuneCountInString(strings.TrimSpace(rule.TriggerEvent)) > 64 {
		return infraerrors.BadRequest("INVALID_RULE", "trigger_event is too long")
	}
	return nil
}

func validateTaskTextFields(code, title, description, icon string) error {
	if code != "" {
		code = strings.TrimSpace(code)
		if utf8.RuneCountInString(code) > 64 || !taskCodePattern.MatchString(code) {
			return infraerrors.BadRequest("INVALID_TASK_CODE", "task_code must be 1-64 characters using letters, numbers, underscore or hyphen")
		}
	}
	if count := utf8.RuneCountInString(strings.TrimSpace(title)); count == 0 || count > 255 {
		return infraerrors.BadRequest("INVALID_TITLE", "title must contain 1-255 characters")
	}
	if utf8.RuneCountInString(description) > 2048 {
		return infraerrors.BadRequest("INVALID_DESCRIPTION", "description is too long")
	}
	if utf8.RuneCountInString(icon) > 128 {
		return infraerrors.BadRequest("INVALID_ICON", "icon is too long")
	}
	return nil
}

func validateReason(reason string) error {
	if utf8.RuneCountInString(strings.TrimSpace(reason)) < 5 {
		return infraerrors.BadRequest("REASON_REQUIRED", "reason must contain at least 5 characters")
	}
	return nil
}

func isTaskCategory(value string) bool {
	return value == "CHECKIN" || value == "DAILY" || value == "ACHIEVEMENT"
}

func defaultTriggerEvent(value string) string {
	if strings.TrimSpace(value) == "" {
		return "COMPLETION"
	}
	return strings.TrimSpace(value)
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
