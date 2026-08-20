package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ScheduleStartMinuteMin = 0
	ScheduleStartMinuteMax = 1439
	ScheduleEndMinuteMin   = 1
	ScheduleEndMinuteMax   = 1440
)

var ErrInvalidModelRouteConcurrencySchedule = errors.New("invalid model route concurrency schedule")

// ModelRouteConcurrencySchedule stores one daily [StartMinute, EndMinute)
// override for a model-route candidate. A nil MaxConcurrency means unlimited.
type ModelRouteConcurrencySchedule struct {
	ID             int64     `json:"id,omitempty"`
	GroupID        int64     `json:"group_id"`
	RouteAlias     string    `json:"route_alias"`
	AccountID      int64     `json:"account_id"`
	StartMinute    int       `json:"start_minute"`
	EndMinute      int       `json:"end_minute"`
	MaxConcurrency *int      `json:"max_concurrency"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// ModelRouteConcurrencyScheduleRepository is intentionally separate from the
// legacy GroupRepository contract so existing callers do not change.
type ModelRouteConcurrencyScheduleRepository interface {
	ListModelRouteConcurrencySchedules(ctx context.Context, groupID int64, routeAlias string, accountID int64) ([]ModelRouteConcurrencySchedule, error)
	ReplaceModelRouteConcurrencySchedules(ctx context.Context, groupID int64, routeAlias string, accountID int64, schedules []ModelRouteConcurrencySchedule) error
	ListAllModelRouteConcurrencySchedules(ctx context.Context) ([]ModelRouteConcurrencySchedule, error)
}

// ModelRouteConcurrencyScheduleCandidate is the scheduler's read model. It
// combines the new schedule rows with the unchanged legacy default value.
type ModelRouteConcurrencyScheduleCandidate struct {
	GroupID               int64
	RouteAlias            string
	AccountID             int64
	DefaultMaxConcurrency *int
	Schedules             []ModelRouteConcurrencySchedule
}

// ModelRouteConcurrencyScheduleRefreshRepository is separate from the admin
// CRUD repository because the refresh task needs the legacy default in the
// same database snapshot.
type ModelRouteConcurrencyScheduleRefreshRepository interface {
	ListModelRouteConcurrencyScheduleCandidates(context.Context) ([]ModelRouteConcurrencyScheduleCandidate, error)
}

// Validate checks one row's minute and value constraints.
func (s ModelRouteConcurrencySchedule) Validate() error {
	if s.GroupID <= 0 || s.AccountID <= 0 || strings.TrimSpace(s.RouteAlias) == "" {
		return fmt.Errorf("%w: candidate identity is required", ErrInvalidModelRouteConcurrencySchedule)
	}
	if s.StartMinute < ScheduleStartMinuteMin || s.StartMinute > ScheduleStartMinuteMax {
		return fmt.Errorf("%w: start minute must be between 0 and 1439", ErrInvalidModelRouteConcurrencySchedule)
	}
	if s.EndMinute < ScheduleEndMinuteMin || s.EndMinute > ScheduleEndMinuteMax || s.StartMinute >= s.EndMinute {
		return fmt.Errorf("%w: end minute must be between 1 and 1440 and after start", ErrInvalidModelRouteConcurrencySchedule)
	}
	if s.MaxConcurrency != nil && *s.MaxConcurrency <= 0 {
		return fmt.Errorf("%w: max concurrency must be positive or nil", ErrInvalidModelRouteConcurrencySchedule)
	}
	return nil
}

// ValidateModelRouteConcurrencySchedules validates a candidate's complete
// schedule list. Adjacent [start,end) intervals are valid; overlapping ones
// are rejected. The input order is preserved.
func ValidateModelRouteConcurrencySchedules(schedules []ModelRouteConcurrencySchedule) error {
	ordered := append([]ModelRouteConcurrencySchedule(nil), schedules...)
	for _, schedule := range ordered {
		if err := schedule.Validate(); err != nil {
			return err
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].StartMinute == ordered[j].StartMinute {
			return ordered[i].EndMinute < ordered[j].EndMinute
		}
		return ordered[i].StartMinute < ordered[j].StartMinute
	})
	for i := 1; i < len(ordered); i++ {
		if ordered[i].StartMinute < ordered[i-1].EndMinute {
			return fmt.Errorf("%w: time windows overlap", ErrInvalidModelRouteConcurrencySchedule)
		}
	}
	return nil
}

// ParseDailyScheduleMinute parses HH:mm. 24:00 is accepted only for an end.
func ParseDailyScheduleMinute(value string, allowEndOfDay bool) (int, error) {
	value = strings.TrimSpace(value)
	if allowEndOfDay && value == "24:00" {
		return ScheduleEndMinuteMax, nil
	}
	if len(value) != 5 || value[2] != ':' {
		return 0, fmt.Errorf("%w: time must use HH:mm", ErrInvalidModelRouteConcurrencySchedule)
	}
	hour, hourErr := strconv.Atoi(value[:2])
	minute, minuteErr := strconv.Atoi(value[3:])
	if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("%w: invalid time %q", ErrInvalidModelRouteConcurrencySchedule, value)
	}
	return hour*60 + minute, nil
}

func FormatDailyScheduleMinute(minute int) (string, error) {
	if minute < ScheduleStartMinuteMin || minute > ScheduleEndMinuteMax {
		return "", fmt.Errorf("%w: minute must be between 0 and 1440", ErrInvalidModelRouteConcurrencySchedule)
	}
	if minute == ScheduleEndMinuteMax {
		return "24:00", nil
	}
	return fmt.Sprintf("%02d:%02d", minute/60, minute%60), nil
}
