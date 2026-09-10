package service

import (
	"errors"
	"testing"
)

func TestValidateModelRouteConcurrencySchedulesAllowsAdjacentAndIncompleteWindows(t *testing.T) {
	limit := 10
	err := ValidateModelRouteConcurrencySchedules([]ModelRouteConcurrencySchedule{
		{GroupID: 1, RouteAlias: "test", AccountID: 2, StartMinute: 0, EndMinute: 570, MaxConcurrency: &limit},
		{GroupID: 1, RouteAlias: "test", AccountID: 2, StartMinute: 570, EndMinute: 1440},
	})
	if err != nil {
		t.Fatalf("adjacent windows should be valid: %v", err)
	}

	err = ValidateModelRouteConcurrencySchedules([]ModelRouteConcurrencySchedule{
		{GroupID: 1, RouteAlias: "test", AccountID: 2, StartMinute: 120, EndMinute: 180},
	})
	if err != nil {
		t.Fatalf("incomplete coverage should be valid: %v", err)
	}
}

func TestValidateModelRouteConcurrencySchedulesRejectsOverlapAndInvalidValue(t *testing.T) {
	limit := 10
	err := ValidateModelRouteConcurrencySchedules([]ModelRouteConcurrencySchedule{
		{GroupID: 1, RouteAlias: "test", AccountID: 2, StartMinute: 0, EndMinute: 120, MaxConcurrency: &limit},
		{GroupID: 1, RouteAlias: "test", AccountID: 2, StartMinute: 119, EndMinute: 180, MaxConcurrency: &limit},
	})
	if !errors.Is(err, ErrInvalidModelRouteConcurrencySchedule) {
		t.Fatalf("overlap error = %v", err)
	}

	zero := 0
	err = ValidateModelRouteConcurrencySchedules([]ModelRouteConcurrencySchedule{
		{GroupID: 1, RouteAlias: "test", AccountID: 2, StartMinute: 0, EndMinute: 1, MaxConcurrency: &zero},
	})
	if !errors.Is(err, ErrInvalidModelRouteConcurrencySchedule) {
		t.Fatalf("invalid value error = %v", err)
	}
}

func TestDailyScheduleMinuteConversion(t *testing.T) {
	for _, tc := range []struct {
		value string
		end   bool
		want  int
	}{
		{"00:00", false, 0},
		{"09:30", false, 570},
		{"24:00", true, 1440},
	} {
		got, err := ParseDailyScheduleMinute(tc.value, tc.end)
		if err != nil || got != tc.want {
			t.Fatalf("ParseDailyScheduleMinute(%q, %t) = %d, %v; want %d", tc.value, tc.end, got, err, tc.want)
		}
	}
	if _, err := ParseDailyScheduleMinute("24:00", false); err == nil {
		t.Fatal("24:00 should not be a valid start")
	}
	if got, err := FormatDailyScheduleMinute(1440); err != nil || got != "24:00" {
		t.Fatalf("FormatDailyScheduleMinute(1440) = %q, %v", got, err)
	}
}
