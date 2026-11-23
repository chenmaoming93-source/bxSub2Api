package growth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthaccountledger"
	"github.com/Wei-Shaw/sub2api/ent/growthexchangeorder"
	"github.com/Wei-Shaw/sub2api/ent/growthproduct"
	"github.com/Wei-Shaw/sub2api/ent/growthproductinventory"
	"github.com/Wei-Shaw/sub2api/ent/growthproductversion"
	"github.com/Wei-Shaw/sub2api/ent/growthrewardclaim"
	"github.com/Wei-Shaw/sub2api/ent/growthtask"
	"github.com/Wei-Shaw/sub2api/ent/growthtaskrule"
	"github.com/Wei-Shaw/sub2api/ent/growthusertask"
	"github.com/Wei-Shaw/sub2api/ent/growthwallet"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type UOverviewView struct {
	UserID           int64            `json:"user_id"`
	AvailablePoints  int64            `json:"available_points"`
	EarnedTotal      int64            `json:"earned_total"`
	SpentTotal       int64            `json:"spent_total"`
	BalanceUpdatedAt *time.Time       `json:"balance_updated_at,omitempty"`
	Checkin          UCheckinView     `json:"checkin"`
	TaskSummary      UTaskSummaryView `json:"task_summary"`
	ServerTime       string           `json:"server_time"`
}

type UPointsRankView struct {
	UserID          int64  `json:"user_id"`
	AvailablePoints int64  `json:"available_points"`
	Rank            int64  `json:"rank"`
	TotalUsers      int64  `json:"total_users"`
	RankText        string `json:"rank_text"`
}

type UCheckinView struct {
	BusinessDate string `json:"business_date"`
	Timezone     string `json:"timezone"`
	Status       string `json:"status"`
	RuleVersion  int    `json:"rule_version"`
	RewardHint   int64  `json:"reward_hint"`
}

type UTaskSummaryView struct {
	InProgress     int `json:"in_progress"`
	CompletedToday int `json:"completed_today"`
}

type WOverviewView struct {
	AvailablePoints int64        `json:"available_points"`
	TotalEarned     int64        `json:"total_earned"`
	TotalSpent      int64        `json:"total_spent"`
	MonthEarned     int64        `json:"month_earned"`
	Checkin         WCheckinView `json:"checkin"`
	ServerTime      string       `json:"server_time"`
}

type WCheckinView struct {
	ClaimedToday   bool       `json:"claimed_today"`
	StreakDays     int        `json:"streak_days"`
	NextEligibleAt *time.Time `json:"next_eligible_at,omitempty"`
}

type UTaskItemView struct {
	TaskID                    int64      `json:"task_id"`
	TaskCode                  string     `json:"task_code"`
	Title                     string     `json:"title"`
	Category                  string     `json:"category"`
	TaskType                  string     `json:"task_type"`
	Status                    string     `json:"status"`
	Progress                  int64      `json:"progress"`
	Target                    int64      `json:"target"`
	CycleKey                  string     `json:"cycle_key"`
	RewardHint                int64      `json:"reward_hint"`
	RewardPool                string     `json:"reward_pool"`
	RuleVersion               int        `json:"rule_version"`
	Action                    string     `json:"action"`
	Icon                      string     `json:"icon"`
	SortOrder                 int        `json:"sort_order"`
	Description               string     `json:"description,omitempty"`
	EffectiveFrom             *time.Time `json:"effective_from,omitempty"`
	EffectiveTo               *time.Time `json:"effective_to,omitempty"`
	PeriodType                string     `json:"period_type,omitempty"`
	DailyLimit                int        `json:"daily_limit,omitempty"`
	PerUserLimit              int        `json:"per_user_limit,omitempty"`
	MyCompletedCount          int64      `json:"my_completed_count,omitempty"`
	CompletionRequestEndpoint string     `json:"completion_request_endpoint,omitempty"`
	CompletionHint            string     `json:"completion_hint,omitempty"`
}

type WTaskItemView struct {
	TaskID         int64  `json:"task_id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Points         int64  `json:"points"`
	Period         string `json:"period"`
	DailyLimit     int    `json:"daily_limit"`
	PerUserLimit   int    `json:"per_user_limit"`
	CompletedToday int64  `json:"completed_today"`
	Status         string `json:"status"`
	Action         string `json:"action"`
}

type UProductItemView struct {
	ProductID       int64  `json:"product_id"`
	ProductCode     string `json:"product_code"`
	Title           string `json:"title"`
	ProductType     string `json:"product_type"`
	Version         int    `json:"version"`
	PointsCost      int64  `json:"points_cost"`
	BenefitSummary  string `json:"benefit_summary"`
	StockHint       string `json:"stock_hint"`
	PerUserLimit    int    `json:"per_user_limit"`
	MyRedeemedCount int64  `json:"my_redeemed_count"`
	Status          string `json:"status"`
}

type UProductDetailView struct {
	UProductItemView
	BenefitDetail    map[string]any `json:"benefit_detail"`
	RedeemWindow     string         `json:"redeem_window"`
	RemainingQuota   int64          `json:"remaining_quota"`
	Eligible         bool           `json:"eligible"`
	IneligibleReason string         `json:"ineligible_reason"`
}

type WProductItemView struct {
	ProductID       int64          `json:"product_id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	PointsCost      int64          `json:"points_cost"`
	Stock           int64          `json:"stock"`
	PerUserLimit    int            `json:"per_user_limit"`
	CanExchange     bool           `json:"can_exchange"`
	Reason          string         `json:"reason,omitempty"`
	FulfillmentNote string         `json:"fulfillment_note,omitempty"`
	BenefitDetail   map[string]any `json:"benefit_detail,omitempty"`
}

type UExchangeItemView struct {
	OrderID      int64      `json:"order_id"`
	ProductTitle string     `json:"product_title"`
	PointsCost   int64      `json:"points_cost"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	FulfilledAt  *time.Time `json:"fulfilled_at,omitempty"`
}

type WExchangeItemView struct {
	OrderID        int64     `json:"order_id"`
	OrderNo        string    `json:"order_no"`
	ProductTitle   string    `json:"product_title"`
	ProductVersion int       `json:"product_version"`
	PointsCost     int64     `json:"points_cost"`
	Status         string    `json:"status"`
	FailureReason  string    `json:"failure_reason,omitempty"`
	Refunded       bool      `json:"refunded"`
	CreatedAt      time.Time `json:"created_at"`
}

type LedgerPage struct {
	Items      []LedgerItem `json:"items"`
	NextCursor string       `json:"next_cursor"`
	HasMore    bool         `json:"has_more"`
}

type LedgerItem struct {
	EntryID      int64     `json:"entry_id"`
	TransferID   int64     `json:"transfer_id"`
	Type         string    `json:"type"`
	Direction    string    `json:"direction"`
	Points       int64     `json:"points"`
	BalanceAfter *int64    `json:"balance_after"`
	BizType      string    `json:"biz_type"`
	Title        string    `json:"title"`
	CreatedAt    time.Time `json:"created_at"`
}

type ReadService struct {
	client   *ent.Client
	location *time.Location
	timezone string
	now      func() time.Time
}

func NewReadService(client *ent.Client, cfg *config.Config) *ReadService {
	timezone := "Asia/Shanghai"
	if cfg != nil && strings.TrimSpace(cfg.Timezone) != "" {
		timezone = strings.TrimSpace(cfg.Timezone)
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	return &ReadService{client: client, location: location, timezone: timezone, now: time.Now}
}

func (s *ReadService) UOverview(ctx context.Context, userID int64) (UOverviewView, error) {
	core, err := s.readOverview(ctx, userID)
	if err != nil {
		return UOverviewView{}, err
	}
	return UOverviewView{
		UserID:           userID,
		AvailablePoints:  core.wallet,
		EarnedTotal:      core.earned,
		SpentTotal:       core.spent,
		BalanceUpdatedAt: core.balanceUpdated,
		Checkin: UCheckinView{
			BusinessDate: core.businessDate,
			Timezone:     s.timezone,
			Status:       core.checkinStatus,
			RuleVersion:  core.ruleVersion,
			RewardHint:   core.rewardHint,
		},
		TaskSummary: UTaskSummaryView{InProgress: core.inProgress, CompletedToday: core.completedToday},
		ServerTime:  s.serverTime(),
	}, nil
}

func (s *ReadService) UPointsRank(ctx context.Context, userID int64) (UPointsRankView, error) {
	const excludedEmail = "admin@example.com"
	var result UPointsRankView
	rows, err := s.client.QueryContext(ctx, `
SELECT
  u.id,
  COALESCE(w.available_points, 0),
  1 + (
    SELECT COUNT(*)
    FROM users peer
    LEFT JOIN growth_wallets peer_wallet ON peer_wallet.user_id = peer.id
    WHERE peer.deleted_at IS NULL
      AND LOWER(peer.email) <> ?
      AND COALESCE(peer_wallet.available_points, 0) > COALESCE(w.available_points, 0)
  ),
  (
    SELECT COUNT(*)
    FROM users counted
    WHERE counted.deleted_at IS NULL
      AND LOWER(counted.email) <> ?
  )
FROM users u
LEFT JOIN growth_wallets w ON w.user_id = u.id
WHERE u.id = ?
  AND u.deleted_at IS NULL
  AND LOWER(u.email) <> ?`,
		excludedEmail, excludedEmail, userID, excludedEmail,
	)
	if err != nil {
		return UPointsRankView{}, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return UPointsRankView{}, err
		}
		return UPointsRankView{}, infraerrors.NotFound("POINTS_RANK_UNAVAILABLE", "current user is excluded from points ranking")
	}
	if err := rows.Scan(&result.UserID, &result.AvailablePoints, &result.Rank, &result.TotalUsers); err != nil {
		return UPointsRankView{}, err
	}
	result.RankText = fmt.Sprintf("%d/%d", result.Rank, result.TotalUsers)
	return result, nil
}

func (s *ReadService) WOverview(ctx context.Context, userID int64) (WOverviewView, error) {
	core, err := s.readOverview(ctx, userID)
	if err != nil {
		return WOverviewView{}, err
	}
	return WOverviewView{
		AvailablePoints: core.wallet,
		TotalEarned:     core.earned,
		TotalSpent:      core.spent,
		MonthEarned:     core.monthEarned,
		Checkin: WCheckinView{
			ClaimedToday:   core.claimedToday,
			StreakDays:     core.streakDays,
			NextEligibleAt: core.nextEligibleAt,
		},
		ServerTime: s.serverTime(),
	}, nil
}

type overviewCore struct {
	wallet         int64
	earned         int64
	spent          int64
	monthEarned    int64
	balanceUpdated *time.Time
	businessDate   string
	checkinStatus  string
	claimedToday   bool
	streakDays     int
	nextEligibleAt *time.Time
	ruleVersion    int
	rewardHint     int64
	inProgress     int
	completedToday int
}

func (s *ReadService) readOverview(ctx context.Context, userID int64) (overviewCore, error) {
	if userID <= 0 {
		return overviewCore{}, infraerrors.BadRequest("INVALID_USER_ID", "user_id must be positive")
	}
	now := s.now().In(s.location)
	result := overviewCore{businessDate: now.Format("2006-01-02"), checkinStatus: "NOT_ELIGIBLE"}
	wallet, err := s.client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(userID)).Only(ctx)
	if err == nil {
		result.wallet = wallet.AvailablePoints
		result.earned = wallet.EarnedTotal
		result.spent = wallet.SpentTotal
		result.balanceUpdated = &wallet.BalanceUpdatedAt
	} else if !ent.IsNotFound(err) {
		return overviewCore{}, err
	}

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, s.location)
	monthEarnedRows, err := s.client.QueryContext(ctx, `
SELECT COALESCE(SUM(points), 0)
FROM growth_account_ledger
WHERE user_id = ?
  AND account_type = 'WALLET'
  AND direction = 'IN'
  AND created_at >= ?`,
		userID,
		monthStart,
	)
	if err != nil {
		return overviewCore{}, err
	}
	defer monthEarnedRows.Close()
	monthEarned := int64(0)
	if monthEarnedRows.Next() {
		if err := monthEarnedRows.Scan(&monthEarned); err != nil {
			return overviewCore{}, err
		}
	}
	if err := monthEarnedRows.Err(); err != nil {
		return overviewCore{}, err
	}
	result.monthEarned = monthEarned

	inProgress, err := s.client.GrowthUserTask.Query().
		Where(growthusertask.UserIDEQ(userID), growthusertask.ProgressStatusEQ("IN_PROGRESS")).
		Count(ctx)
	if err != nil {
		return overviewCore{}, err
	}
	result.inProgress = inProgress
	completedToday, err := s.client.GrowthUserTask.Query().
		Where(growthusertask.UserIDEQ(userID), growthusertask.RewardedAtGTE(dayStart)).
		Count(ctx)
	if err != nil {
		return overviewCore{}, err
	}
	result.completedToday = completedToday

	checkinTask, err := s.client.GrowthTask.Query().Where(growthtask.TaskCodeEQ("daily_checkin"), growthtask.TaskStatusEQ("ACTIVE")).Only(ctx)
	if ent.IsNotFound(err) {
		return result, nil
	}
	if err != nil {
		return overviewCore{}, err
	}
	rule, err := s.client.GrowthTaskRule.Query().
		Where(
			growthtaskrule.TaskIDEQ(checkinTask.ID),
			growthtaskrule.PublishedAtNotNil(),
			growthtaskrule.EffectiveFromLTE(now),
			growthtaskrule.Or(growthtaskrule.EffectiveToIsNil(), growthtaskrule.EffectiveToGT(now)),
		).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		First(ctx)
	if ent.IsNotFound(err) {
		return result, nil
	}
	if err != nil {
		return overviewCore{}, err
	}
	result.ruleVersion = rule.Version
	result.rewardHint = rule.RewardPoints
	cycleKey, _ := BusinessCycleKey(now, s.location, "DAILY")
	claimCount, err := s.client.GrowthRewardClaim.Query().
		Where(
			growthrewardclaim.UserIDEQ(userID),
			growthrewardclaim.TaskIDEQ(checkinTask.ID),
			growthrewardclaim.CycleKeyEQ(cycleKey),
		).
		Count(ctx)
	if err != nil {
		return overviewCore{}, err
	}
	if claimCount > 0 {
		result.checkinStatus = "CLAIMED"
		result.claimedToday = true
		next := dayStart.Add(24 * time.Hour)
		result.nextEligibleAt = &next
	} else {
		result.checkinStatus = "AVAILABLE"
		result.nextEligibleAt = &now
	}
	result.streakDays = s.checkinStreak(ctx, userID, checkinTask.ID, now)
	return result, nil
}

func (s *ReadService) checkinStreak(ctx context.Context, userID, taskID int64, now time.Time) int {
	claims, err := s.client.GrowthRewardClaim.Query().
		Where(growthrewardclaim.UserIDEQ(userID), growthrewardclaim.TaskIDEQ(taskID)).
		Order(ent.Desc(growthrewardclaim.FieldClaimedAt)).
		Limit(90).
		All(ctx)
	if err != nil {
		return 0
	}
	dates := make(map[string]bool, len(claims))
	for _, claim := range claims {
		dates[claim.ClaimedAt.In(s.location).Format("2006-01-02")] = true
	}
	cursor := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
	if !dates[cursor.Format("2006-01-02")] {
		cursor = cursor.Add(-24 * time.Hour)
	}
	streak := 0
	for dates[cursor.Format("2006-01-02")] {
		streak++
		cursor = cursor.Add(-24 * time.Hour)
	}
	return streak
}

func (s *ReadService) UTasks(ctx context.Context, userID int64, page, pageSize int, statusFilter string) ([]UTaskItemView, int64, error) {
	tasks, total, err := s.activeTasks(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	now := s.now().In(s.location)
	items := make([]UTaskItemView, 0, len(tasks))
	for _, task := range tasks {
		rule, _ := s.currentRule(ctx, task.ID, now)
		if rule == nil {
			continue
		}
		progress, status, cycleKey, completedCount := s.taskProgress(ctx, userID, task.ID, rule, now)
		action := "OPEN_TASK"
		endpoint := fmt.Sprintf("/growth/tasks/%d/completion-requests", task.ID)
		if task.TaskType == "CHECKIN" {
			action = "CHECKIN"
			endpoint = "/growth/checkins"
		}
		item := UTaskItemView{
			TaskID: task.ID, TaskCode: task.TaskCode, Title: task.Title,
			Category: task.Category, TaskType: task.TaskType, Status: status,
			Progress: progress, Target: rule.Target, CycleKey: cycleKey,
			RewardHint: rule.RewardPoints, RewardPool: rewardPool(task.TaskType),
			RuleVersion: rule.Version, Action: action, Icon: task.Icon, SortOrder: task.SortOrder,
			Description: task.Remark, PeriodType: rule.PeriodType,
			DailyLimit: rule.DailyLimit, PerUserLimit: rule.PerUserLimit,
			MyCompletedCount: completedCount, CompletionRequestEndpoint: endpoint,
			CompletionHint: "Complete the task in AIWork",
		}
		if strings.TrimSpace(statusFilter) == "" || item.Status == strings.TrimSpace(statusFilter) {
			items = append(items, item)
		}
	}
	return items, total, nil
}

func (s *ReadService) UTask(ctx context.Context, userID, taskID int64) (UTaskItemView, error) {
	task, err := s.client.GrowthTask.Get(ctx, taskID)
	if ent.IsNotFound(err) || task.TaskStatus != "ACTIVE" {
		return UTaskItemView{}, ErrGrowthTaskNotFound
	}
	if err != nil {
		return UTaskItemView{}, err
	}
	rule, err := s.currentRule(ctx, taskID, s.now().In(s.location))
	if err != nil || rule == nil {
		return UTaskItemView{}, ErrGrowthTaskNotFound
	}
	progress, status, cycleKey, completedCount := s.taskProgress(ctx, userID, taskID, rule, s.now().In(s.location))
	action := "OPEN_TASK"
	endpoint := fmt.Sprintf("/growth/tasks/%d/completion-requests", task.ID)
	if task.TaskType == "CHECKIN" {
		action = "CHECKIN"
		endpoint = "/growth/checkins"
	}
	return UTaskItemView{
		TaskID: task.ID, TaskCode: task.TaskCode, Title: task.Title,
		Category: task.Category, TaskType: task.TaskType, Status: status,
		Progress: progress, Target: rule.Target, CycleKey: cycleKey,
		RewardHint: rule.RewardPoints, RewardPool: rewardPool(task.TaskType),
		RuleVersion: rule.Version, Action: action, Icon: task.Icon, SortOrder: task.SortOrder,
		Description: task.Remark, EffectiveFrom: &rule.EffectiveFrom, EffectiveTo: task.EffectiveTo,
		PeriodType: rule.PeriodType, DailyLimit: rule.DailyLimit, PerUserLimit: rule.PerUserLimit,
		MyCompletedCount: completedCount, CompletionRequestEndpoint: endpoint,
		CompletionHint: "Complete the task in AIWork",
	}, nil
}

func (s *ReadService) WTasks(ctx context.Context, userID int64) ([]WTaskItemView, error) {
	tasks, _, err := s.activeTasks(ctx, 1, 100)
	if err != nil {
		return nil, err
	}
	now := s.now().In(s.location)
	result := make([]WTaskItemView, 0, len(tasks))
	for _, task := range tasks {
		rule, _ := s.currentRule(ctx, task.ID, now)
		if rule == nil {
			continue
		}
		_, status, _, completed := s.taskProgress(ctx, userID, task.ID, rule, now)
		result = append(result, WTaskItemView{
			TaskID: task.ID, Title: task.Title, Description: task.Remark,
			Points: rule.RewardPoints, Period: rule.PeriodType,
			DailyLimit: rule.DailyLimit, PerUserLimit: rule.PerUserLimit,
			CompletedToday: completed, Status: status, Action: "OPEN_AIWORK",
		})
	}
	return result, nil
}

func (s *ReadService) activeTasks(ctx context.Context, page, pageSize int) ([]*ent.GrowthTask, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.client.GrowthTask.Query().Where(growthtask.TaskStatusEQ("ACTIVE"))
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	tasks, err := query.Order(ent.Asc(growthtask.FieldSortOrder), ent.Desc(growthtask.FieldID)).
		Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	return tasks, int64(total), err
}

func (s *ReadService) currentRule(ctx context.Context, taskID int64, now time.Time) (*ent.GrowthTaskRule, error) {
	rule, err := s.client.GrowthTaskRule.Query().
		Where(
			growthtaskrule.TaskIDEQ(taskID),
			growthtaskrule.PublishedAtNotNil(),
			growthtaskrule.EffectiveFromLTE(now),
			growthtaskrule.Or(growthtaskrule.EffectiveToIsNil(), growthtaskrule.EffectiveToGT(now)),
		).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return rule, err
}

func (s *ReadService) taskProgress(ctx context.Context, userID, taskID int64, rule *ent.GrowthTaskRule, now time.Time) (int64, string, string, int64) {
	cycleKey, _ := BusinessCycleKey(now, s.location, rule.PeriodType)
	userTask, err := s.client.GrowthUserTask.Query().
		Where(
			growthusertask.UserIDEQ(userID),
			growthusertask.TaskIDEQ(taskID),
			growthusertask.RuleIDEQ(rule.ID),
			growthusertask.CycleKeyEQ(cycleKey),
		).
		Only(ctx)
	progress := int64(0)
	completed := int64(0)
	status := "AVAILABLE"
	if err == nil {
		progress = userTask.Progress
		completed = userTask.CompletedCount
		if progress > 0 {
			status = "IN_PROGRESS"
		}
	} else if !ent.IsNotFound(err) {
		return 0, "PAUSED", cycleKey, 0
	}
	cycleClaims, err := s.client.GrowthRewardClaim.Query().
		Where(
			growthrewardclaim.UserIDEQ(userID),
			growthrewardclaim.TaskIDEQ(taskID),
			growthrewardclaim.CycleKeyEQ(cycleKey),
			growthrewardclaim.ClaimStatusEQ("SUCCEEDED"),
		).
		Count(ctx)
	if err != nil {
		return progress, status, cycleKey, completed
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
	dailyClaims, err := s.client.GrowthRewardClaim.Query().
		Where(
			growthrewardclaim.UserIDEQ(userID),
			growthrewardclaim.TaskIDEQ(taskID),
			growthrewardclaim.ClaimedAtGTE(dayStart),
			growthrewardclaim.ClaimStatusEQ("SUCCEEDED"),
		).
		Count(ctx)
	if err != nil {
		return progress, status, cycleKey, completed
	}
	perUserClaims, err := s.client.GrowthRewardClaim.Query().
		Where(
			growthrewardclaim.UserIDEQ(userID),
			growthrewardclaim.TaskIDEQ(taskID),
			growthrewardclaim.ClaimStatusEQ("SUCCEEDED"),
		).
		Count(ctx)
	if err != nil {
		return progress, status, cycleKey, completed
	}
	if (rule.DailyLimit > 0 && dailyClaims >= rule.DailyLimit) ||
		(rule.PerUserLimit > 0 && perUserClaims >= rule.PerUserLimit) {
		status = "REWARDED"
	} else if cycleClaims > 0 || progress > 0 {
		status = "AVAILABLE"
	}
	return progress, status, cycleKey, completed
}

func (s *ReadService) UProducts(ctx context.Context, userID int64, page, pageSize int) ([]UProductItemView, int64, error) {
	products, total, err := s.activeProducts(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	result := make([]UProductItemView, 0, len(products))
	for _, product := range products {
		item, eligible, _ := s.uProduct(ctx, userID, product)
		item.Status = "ON_SHELF"
		if !eligible {
			item.Status = "UNAVAILABLE"
		}
		result = append(result, item)
	}
	return result, total, nil
}

func (s *ReadService) UProduct(ctx context.Context, userID, productID int64) (UProductDetailView, error) {
	product, err := s.client.GrowthProduct.Get(ctx, productID)
	if ent.IsNotFound(err) || product.ProductStatus != "ACTIVE" {
		return UProductDetailView{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return UProductDetailView{}, err
	}
	item, eligible, reason := s.uProduct(ctx, userID, product)
	inventory, _ := s.client.GrowthProductInventory.Query().Where(growthproductinventory.ProductIDEQ(productID)).Only(ctx)
	remaining := int64(-1)
	if inventory != nil && inventory.StockMode == "LIMITED" {
		remaining = inventory.Available
	}
	return UProductDetailView{
		UProductItemView: item,
		BenefitDetail:    currentBenefitDetail(ctx, s.client, productID, product.CurrentVersion),
		RedeemWindow:     "兑换后进入履约队列",
		RemainingQuota:   remaining,
		Eligible:         eligible,
		IneligibleReason: reason,
	}, nil
}

func (s *ReadService) uProduct(ctx context.Context, userID int64, product *ent.GrowthProduct) (UProductItemView, bool, string) {
	version, err := s.client.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(product.ID), growthproductversion.VersionEQ(product.CurrentVersion)).
		Only(ctx)
	if err != nil {
		return UProductItemView{}, false, "商品版本不可用"
	}
	inventory, _ := s.client.GrowthProductInventory.Query().Where(growthproductinventory.ProductIDEQ(product.ID)).Only(ctx)
	count, _ := s.client.GrowthExchangeOrder.Query().
		Where(
			growthexchangeorder.UserIDEQ(userID),
			growthexchangeorder.ProductIDEQ(product.ID),
			growthexchangeorder.OrderStatusIn("CREATED", "FULFILLING", "SUCCEEDED"),
		).
		Count(ctx)
	stockHint := "不限"
	eligible := true
	reason := ""
	if inventory != nil && inventory.StockMode == "LIMITED" {
		if inventory.Available <= 0 {
			stockHint = "已售罄"
			eligible = false
			reason = "PRODUCT_SOLD_OUT"
		} else {
			stockHint = fmt.Sprintf("%d 件", inventory.Available)
		}
	}
	if version.PerUserLimit > 0 && int64(version.PerUserLimit) <= int64(count) {
		eligible = false
		if reason == "" {
			reason = "PURCHASE_LIMIT_REACHED"
		}
	}
	wallet, _ := s.client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(userID)).Only(ctx)
	if wallet == nil || wallet.AvailablePoints < version.PointsCost {
		eligible = false
		if reason == "" {
			reason = "INSUFFICIENT_POINTS"
		}
	}
	return UProductItemView{
		ProductID: product.ID, ProductCode: product.ProductCode, Title: product.Title,
		ProductType: product.ProductType, Version: version.Version, PointsCost: version.PointsCost,
		BenefitSummary: version.BenefitSummary, StockHint: stockHint,
		PerUserLimit: version.PerUserLimit, MyRedeemedCount: int64(count),
	}, eligible, reason
}

func (s *ReadService) WProducts(ctx context.Context, userID int64) ([]WProductItemView, error) {
	products, _, err := s.activeProducts(ctx, 1, 100)
	if err != nil {
		return nil, err
	}
	result := make([]WProductItemView, 0, len(products))
	for _, product := range products {
		item, eligible, reason := s.uProduct(ctx, userID, product)
		stock := int64(-1)
		inventory, _ := s.client.GrowthProductInventory.Query().Where(growthproductinventory.ProductIDEQ(product.ID)).Only(ctx)
		if inventory != nil && inventory.StockMode == "LIMITED" {
			stock = inventory.Available
		}
		note := "兑换后进入履约队列"
		if product.FulfillmentMode == "MANUAL" {
			note = "运营人工处理"
		}
		result = append(result, WProductItemView{
			ProductID: product.ID, Name: item.Title, Description: item.BenefitSummary,
			PointsCost: item.PointsCost, Stock: stock, PerUserLimit: item.PerUserLimit,
			CanExchange: eligible, Reason: reason, FulfillmentNote: note,
			BenefitDetail: currentBenefitDetail(ctx, s.client, product.ID, product.CurrentVersion),
		})
	}
	return result, nil
}

func (s *ReadService) WProduct(ctx context.Context, userID, productID int64) (WProductItemView, error) {
	product, err := s.client.GrowthProduct.Get(ctx, productID)
	if ent.IsNotFound(err) || product.ProductStatus != "ACTIVE" {
		return WProductItemView{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return WProductItemView{}, err
	}
	item, eligible, reason := s.uProduct(ctx, userID, product)
	stock := int64(-1)
	inventory, _ := s.client.GrowthProductInventory.Query().Where(growthproductinventory.ProductIDEQ(productID)).Only(ctx)
	if inventory != nil && inventory.StockMode == "LIMITED" {
		stock = inventory.Available
	}
	note := "兑换后进入履约队列"
	if product.FulfillmentMode == "MANUAL" {
		note = "运营人工处理"
	}
	return WProductItemView{
		ProductID: product.ID, Name: item.Title, Description: item.BenefitSummary,
		PointsCost: item.PointsCost, Stock: stock, PerUserLimit: item.PerUserLimit,
		CanExchange: eligible, Reason: reason, FulfillmentNote: note,
		BenefitDetail: currentBenefitDetail(ctx, s.client, productID, product.CurrentVersion),
	}, nil
}

func currentBenefitDetail(ctx context.Context, client *ent.Client, productID int64, version int) map[string]any {
	entity, err := client.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(productID), growthproductversion.VersionEQ(version)).
		Only(ctx)
	if err != nil {
		return nil
	}
	return normalizeJSONMap(entity.BenefitDetail)
}

func (s *ReadService) activeProducts(ctx context.Context, page, pageSize int) ([]*ent.GrowthProduct, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.client.GrowthProduct.Query().Where(growthproduct.ProductStatusEQ("ACTIVE"))
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	products, err := query.Order(ent.Asc(growthproduct.FieldSortOrder), ent.Desc(growthproduct.FieldID)).
		Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	return products, int64(total), err
}

func (s *ReadService) UExchanges(ctx context.Context, userID int64, page, pageSize int, status string) ([]UExchangeItemView, int64, error) {
	orders, total, err := s.orders(ctx, userID, page, pageSize, status)
	if err != nil {
		return nil, 0, err
	}
	result := make([]UExchangeItemView, 0, len(orders))
	for _, order := range orders {
		title, _ := normalizeJSONMap(order.ProductSnapshot)["title"].(string)
		result = append(result, UExchangeItemView{
			OrderID: order.ID, ProductTitle: title, PointsCost: order.PointsCost,
			Status: order.OrderStatus, CreatedAt: order.CreatedAt, FulfilledAt: order.FulfilledAt,
		})
	}
	return result, total, nil
}

func (s *ReadService) WExchanges(ctx context.Context, userID int64, page, pageSize int, status string) ([]WExchangeItemView, int64, error) {
	orders, total, err := s.orders(ctx, userID, page, pageSize, status)
	if err != nil {
		return nil, 0, err
	}
	result := make([]WExchangeItemView, 0, len(orders))
	for _, order := range orders {
		title, _ := normalizeJSONMap(order.ProductSnapshot)["title"].(string)
		result = append(result, WExchangeItemView{
			OrderID: order.ID, OrderNo: order.OrderNo, ProductTitle: title,
			ProductVersion: order.ProductVersion, PointsCost: order.PointsCost,
			Status: order.OrderStatus, FailureReason: order.FailureReason,
			Refunded: order.OrderStatus == "REFUNDED", CreatedAt: order.CreatedAt,
		})
	}
	return result, total, nil
}

func (s *ReadService) orders(ctx context.Context, userID int64, page, pageSize int, status string) ([]*ent.GrowthExchangeOrder, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.client.GrowthExchangeOrder.Query().Where(growthexchangeorder.UserIDEQ(userID))
	if strings.TrimSpace(status) != "" {
		query = query.Where(growthexchangeorder.OrderStatusEQ(strings.TrimSpace(status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	orders, err := query.Order(ent.Desc(growthexchangeorder.FieldCreatedAt)).
		Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	return orders, int64(total), err
}

func (s *ReadService) Ledger(ctx context.Context, userID int64, beforeEntryID int64, limit int, ledgerType string) (LedgerPage, error) {
	if userID <= 0 {
		return LedgerPage{}, infraerrors.BadRequest("INVALID_USER_ID", "user_id must be positive")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := s.client.GrowthAccountLedger.Query().
		Where(growthaccountledger.UserIDEQ(userID), growthaccountledger.AccountTypeEQ("WALLET"))
	if beforeEntryID > 0 {
		query = query.Where(growthaccountledger.IDLT(beforeEntryID))
	}
	switch strings.ToUpper(strings.TrimSpace(ledgerType)) {
	case "EARN":
		query = query.Where(growthaccountledger.DirectionEQ("IN"), growthaccountledger.BizTypeNotIn("REFUND", "ADJUST"))
	case "REDEEM":
		query = query.Where(growthaccountledger.DirectionEQ("OUT"))
	case "REFUND":
		query = query.Where(growthaccountledger.BizTypeEQ("REFUND"))
	case "ADJUST":
		query = query.Where(growthaccountledger.BizTypeEQ("ADJUST"))
	}
	entries, err := query.Order(ent.Desc(growthaccountledger.FieldID)).Limit(limit + 1).All(ctx)
	if err != nil {
		return LedgerPage{}, err
	}
	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}
	items := make([]LedgerItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, LedgerItem{
			EntryID: entry.ID, TransferID: entry.TransferID, Type: ledgerTypeFor(entry),
			Direction: entry.Direction, Points: entry.Points,
			BalanceAfter: entry.BalanceAfter, BizType: entry.BizType,
			Title: entry.Title, CreatedAt: entry.CreatedAt,
		})
	}
	nextCursor := ""
	if len(items) > 0 {
		nextCursor = fmt.Sprintf("%d", items[len(items)-1].EntryID)
	}
	return LedgerPage{Items: items, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func ledgerTypeFor(entry *ent.GrowthAccountLedger) string {
	switch strings.ToUpper(entry.BizType) {
	case "REFUND":
		return "REFUND"
	case "ADJUST":
		return "ADJUST"
	}
	if entry.Direction == "IN" {
		return "EARN"
	}
	return "REDEEM"
}

func (s *ReadService) serverTime() string {
	return s.now().In(s.location).Format(time.RFC3339Nano)
}

func rewardPool(taskType string) string {
	if taskType == "CHECKIN" {
		return "CHECKIN"
	}
	return "TASK"
}
