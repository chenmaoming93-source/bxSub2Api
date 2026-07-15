package service

import (
	"context"
	"time"
)

type TokenStatisticsIncrement struct {
	UserID        int64
	GroupID       int64
	RouteAlias    string
	Model         string
	UpstreamModel string
	UsageDate     time.Time
	TotalTokens   int64
}

type TokenStatisticsAccumulator interface {
	Accumulate(context.Context, TokenStatisticsIncrement) error
}

type TokenStatisticsDateSyncer interface {
	SyncDate(context.Context, time.Time) error
}
