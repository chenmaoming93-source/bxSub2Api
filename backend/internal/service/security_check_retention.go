package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultSecurityCheckLogRetentionDays = 3
	DefaultSecurityCheckLogCleanupTime   = "03:00"
	MinSecurityCheckLogRetentionDays     = 1
	MaxSecurityCheckLogRetentionDays     = 3650
)

// SecurityCheckLogRetentionConfig controls the daily cleanup of security logs.
// Timezone is the process-local timezone and is not persisted.
type SecurityCheckLogRetentionConfig struct {
	RetentionDays int    `json:"retention_days"`
	CleanupTime   string `json:"cleanup_time"`
	Timezone      string `json:"timezone"`
	NextCleanupAt string `json:"next_cleanup_at"`
}

func DefaultSecurityCheckLogRetentionConfig() SecurityCheckLogRetentionConfig {
	return SecurityCheckLogRetentionConfig{
		RetentionDays: DefaultSecurityCheckLogRetentionDays,
		CleanupTime:   DefaultSecurityCheckLogCleanupTime,
		Timezone:      time.Local.String(),
	}
}

func (c SecurityCheckLogRetentionConfig) Validate() error {
	if c.RetentionDays < MinSecurityCheckLogRetentionDays || c.RetentionDays > MaxSecurityCheckLogRetentionDays {
		return fmt.Errorf("retention_days must be between %d and %d", MinSecurityCheckLogRetentionDays, MaxSecurityCheckLogRetentionDays)
	}
	if !isValidSecurityCleanupTime(c.CleanupTime) {
		return fmt.Errorf("cleanup_time must use HH:mm format")
	}
	return nil
}

func isValidSecurityCleanupTime(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	hour, hourErr := strconv.Atoi(value[:2])
	minute, minuteErr := strconv.Atoi(value[3:])
	return hourErr == nil && minuteErr == nil && hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}

func (s *SettingService) GetSecurityCheckLogRetentionConfig(ctx context.Context) (SecurityCheckLogRetentionConfig, error) {
	config := DefaultSecurityCheckLogRetentionConfig()
	if s == nil || s.settingRepo == nil {
		return config, nil
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeySecurityCheckLogRetentionDays, SettingKeySecurityCheckLogCleanupTime})
	if err != nil {
		return config, fmt.Errorf("get security log retention settings: %w", err)
	}
	if raw := strings.TrimSpace(values[SettingKeySecurityCheckLogRetentionDays]); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			slog.Warn("security_check.retention_config_invalid", "key", SettingKeySecurityCheckLogRetentionDays, "value", raw, "err", parseErr)
			return DefaultSecurityCheckLogRetentionConfig(), nil
		}
		config.RetentionDays = value
	}
	if raw := strings.TrimSpace(values[SettingKeySecurityCheckLogCleanupTime]); raw != "" {
		config.CleanupTime = raw
	}
	if err := config.Validate(); err != nil {
		slog.Warn("security_check.retention_config_invalid", "err", err)
		return DefaultSecurityCheckLogRetentionConfig(), nil
	}
	return config, nil
}

func (s *SettingService) SetSecurityCheckLogRetentionConfig(ctx context.Context, config SecurityCheckLogRetentionConfig) error {
	config.Timezone = time.Local.String()
	config.NextCleanupAt = ""
	if err := config.Validate(); err != nil {
		return err
	}
	if s == nil || s.settingRepo == nil {
		return fmt.Errorf("setting service is unavailable")
	}
	return s.settingRepo.SetMultiple(ctx, map[string]string{
		SettingKeySecurityCheckLogRetentionDays: strconv.Itoa(config.RetentionDays),
		SettingKeySecurityCheckLogCleanupTime:   config.CleanupTime,
	})
}

func nextSecurityCleanupAt(now time.Time, cleanupTime string, includeCurrentMinute bool) time.Time {
	localNow := now.In(time.Local)
	hour, _ := strconv.Atoi(cleanupTime[:2])
	minute, _ := strconv.Atoi(cleanupTime[3:])
	target := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, time.Local)
	if localNow.Before(target) || (includeCurrentMinute && localNow.Hour() == hour && localNow.Minute() == minute) {
		return target
	}
	return target.AddDate(0, 0, 1)
}

func securityCleanupScheduledMinute(now, scheduledAt time.Time) bool {
	localNow := now.In(time.Local)
	target := scheduledAt.In(time.Local)
	return localNow.Year() == target.Year() && localNow.YearDay() == target.YearDay() && localNow.Hour() == target.Hour() && localNow.Minute() == target.Minute()
}

func (c SecurityCheckLogRetentionConfig) WithNextCleanup(now time.Time) SecurityCheckLogRetentionConfig {
	if c.Timezone == "" {
		c.Timezone = time.Local.String()
	}
	localNow := now.In(time.Local)
	hour, _ := strconv.Atoi(c.CleanupTime[:2])
	minute, _ := strconv.Atoi(c.CleanupTime[3:])
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, time.Local)
	if !next.After(localNow) {
		next = next.AddDate(0, 0, 1)
	}
	c.NextCleanupAt = next.Format(time.RFC3339)
	return c
}
