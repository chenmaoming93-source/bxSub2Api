package tokenstat

import "time"

type PeriodType string

const (
	PeriodDay   PeriodType = "D"
	PeriodWeek  PeriodType = "W"
	PeriodMonth PeriodType = "M"
)

type Period struct {
	Type  PeriodType
	Start time.Time
	End   time.Time
}

func NaturalPeriods(at time.Time, location *time.Location) []Period {
	local := at.In(location)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	weekday := int(local.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := dayStart.AddDate(0, 0, -weekday+1)
	monthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	return []Period{
		{Type: PeriodDay, Start: dayStart, End: dayStart.AddDate(0, 0, 1)},
		{Type: PeriodWeek, Start: weekStart, End: weekStart.AddDate(0, 0, 7)},
		{Type: PeriodMonth, Start: monthStart, End: monthStart.AddDate(0, 1, 0)},
	}
}

func AsiaShanghaiPeriods(at time.Time) ([]Period, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, err
	}
	return NaturalPeriods(at, location), nil
}
