package service

import (
	"context"
	"testing"
	"time"
)

type securityRetentionSettingRepoStub struct {
	values map[string]string
}

func (r *securityRetentionSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (r *securityRetentionSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}
func (r *securityRetentionSettingRepoStub) GetMultiple(_ context.Context, _ []string) (map[string]string, error) {
	values := make(map[string]string, len(r.values))
	for key, value := range r.values {
		values[key] = value
	}
	return values, nil
}
func (r *securityRetentionSettingRepoStub) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}
func (r *securityRetentionSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		r.values[key] = value
	}
	return nil
}
func (r *securityRetentionSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r *securityRetentionSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestSecurityCheckLogRetentionConfigDefaultsAndRoundTrip(t *testing.T) {
	repo := &securityRetentionSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	config, err := svc.GetSecurityCheckLogRetentionConfig(context.Background())
	if err != nil {
		t.Fatalf("get defaults: %v", err)
	}
	if config.RetentionDays != 3 || config.CleanupTime != "03:00" {
		t.Fatalf("unexpected defaults: %+v", config)
	}

	want := SecurityCheckLogRetentionConfig{RetentionDays: 14, CleanupTime: "22:45"}
	if err := svc.SetSecurityCheckLogRetentionConfig(context.Background(), want); err != nil {
		t.Fatalf("set config: %v", err)
	}
	got, err := svc.GetSecurityCheckLogRetentionConfig(context.Background())
	if err != nil {
		t.Fatalf("get saved config: %v", err)
	}
	if got.RetentionDays != want.RetentionDays || got.CleanupTime != want.CleanupTime {
		t.Fatalf("saved config mismatch: got=%+v want=%+v", got, want)
	}
}

func TestSecurityCheckLogRetentionConfigRejectsInvalidValues(t *testing.T) {
	repo := &securityRetentionSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	for _, config := range []SecurityCheckLogRetentionConfig{
		{RetentionDays: 0, CleanupTime: "03:00"},
		{RetentionDays: 3651, CleanupTime: "03:00"},
		{RetentionDays: 3, CleanupTime: "3:00"},
		{RetentionDays: 3, CleanupTime: "24:00"},
	} {
		if err := svc.SetSecurityCheckLogRetentionConfig(context.Background(), config); err == nil {
			t.Fatalf("expected invalid config to fail: %+v", config)
		}
	}
}

func TestSecurityCleanupScheduleChanged(t *testing.T) {
	base := SecurityCheckLogRetentionConfig{RetentionDays: 3, CleanupTime: "03:00"}
	if securityCleanupScheduleChanged(base, base) {
		t.Fatal("unchanged schedule should not reset the cleanup day")
	}
	if !securityCleanupScheduleChanged(base, SecurityCheckLogRetentionConfig{RetentionDays: 1, CleanupTime: "03:00"}) {
		t.Fatal("retention change should reset the cleanup day")
	}
	if !securityCleanupScheduleChanged(base, SecurityCheckLogRetentionConfig{RetentionDays: 3, CleanupTime: "10:12"}) {
		t.Fatal("time change should reset the cleanup day")
	}
}

func TestSecurityCleanupScheduleDoesNotCatchUp(t *testing.T) {
	before := time.Date(2026, time.August, 31, 10, 11, 30, 0, time.Local)
	today := time.Date(2026, time.August, 31, 10, 12, 0, 0, time.Local)
	if got := nextSecurityCleanupAt(before, "10:12", true); !got.Equal(today) {
		t.Fatalf("expected today's scheduled time: got=%v want=%v", got, today)
	}

	after := time.Date(2026, time.August, 31, 10, 14, 0, 0, time.Local)
	tomorrow := time.Date(2026, time.September, 1, 10, 12, 0, 0, time.Local)
	if got := nextSecurityCleanupAt(after, "10:12", true); !got.Equal(tomorrow) {
		t.Fatalf("late startup must wait until tomorrow: got=%v want=%v", got, tomorrow)
	}
	if got := nextSecurityCleanupAt(after, "10:12", false); !got.Equal(tomorrow) {
		t.Fatalf("late config change must wait until tomorrow: got=%v want=%v", got, tomorrow)
	}

	inMinute := time.Date(2026, time.August, 31, 10, 12, 30, 0, time.Local)
	if !securityCleanupScheduledMinute(inMinute, today) {
		t.Fatal("the configured minute should be eligible")
	}
}

func TestSecurityCheckLogRetentionConfigNextCleanup(t *testing.T) {
	config := SecurityCheckLogRetentionConfig{RetentionDays: 3, CleanupTime: "03:00", Timezone: time.Local.String()}
	before := time.Date(2026, time.August, 31, 2, 59, 0, 0, time.Local)
	expectedSameDay := time.Date(2026, time.August, 31, 3, 0, 0, 0, time.Local).Format(time.RFC3339)
	if got := config.WithNextCleanup(before).NextCleanupAt; got != expectedSameDay {
		t.Fatalf("unexpected same-day next cleanup: got=%s want=%s", got, expectedSameDay)
	}
	after := time.Date(2026, time.August, 31, 3, 1, 0, 0, time.Local)
	expectedNextDay := time.Date(2026, time.September, 1, 3, 0, 0, 0, time.Local).Format(time.RFC3339)
	if got := config.WithNextCleanup(after).NextCleanupAt; got != expectedNextDay {
		t.Fatalf("unexpected next-day cleanup: got=%s want=%s", got, expectedNextDay)
	}
}
