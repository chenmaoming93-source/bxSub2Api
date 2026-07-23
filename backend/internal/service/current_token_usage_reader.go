package service

import (
	"context"
	"time"
)

type CurrentTokenUsageReadResult[T any] struct {
	Rows           []T
	InvalidEntries int
}

type CurrentTokenUsageReader interface {
	ReadModelUsage(context.Context, time.Time, []string) (CurrentTokenUsageReadResult[ModelTokenUsageRow], error)
	ReadRouteUsage(context.Context, time.Time, []RouteTokenUsageRow) (CurrentTokenUsageReadResult[RouteTokenUsageRow], error)
	ReadUserModelUsage(context.Context, time.Time, []UserTokenUsageRow) (CurrentTokenUsageReadResult[UserTokenUsageRow], error)
}

type CurrentTokenUsageRepairer interface {
	RepairModelUsage(context.Context, time.Time, []ModelTokenUsageRow) error
	RepairRouteUsage(context.Context, time.Time, []RouteTokenUsageRow) error
	RepairUserModelUsage(context.Context, time.Time, []UserTokenUsageRow) error
}
