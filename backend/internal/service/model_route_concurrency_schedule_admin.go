package service

import (
	"context"
	"fmt"
)

type groupModelRouteConcurrencyScheduleReader interface {
	ListModelRouteConcurrencySchedules(context.Context, int64, string, int64) ([]ModelRouteConcurrencySchedule, error)
	ReplaceModelRouteConcurrencySchedules(context.Context, int64, string, int64, []ModelRouteConcurrencySchedule) error
}

func (s *adminServiceImpl) ListGroupModelRouteConcurrencySchedules(ctx context.Context, groupID int64, routeAlias string, accountID int64) ([]ModelRouteConcurrencySchedule, error) {
	if _, err := s.groupRepo.GetByID(ctx, groupID); err != nil {
		return nil, err
	}
	reader, ok := s.groupRepo.(groupModelRouteConcurrencyScheduleReader)
	if !ok {
		return nil, fmt.Errorf("group repository does not support schedule configuration")
	}
	return reader.ListModelRouteConcurrencySchedules(ctx, groupID, routeAlias, accountID)
}

func (s *adminServiceImpl) ReplaceGroupModelRouteConcurrencySchedules(ctx context.Context, groupID int64, routeAlias string, accountID int64, schedules []ModelRouteConcurrencySchedule) error {
	if _, err := s.groupRepo.GetByID(ctx, groupID); err != nil {
		return err
	}
	reader, ok := s.groupRepo.(groupModelRouteConcurrencyScheduleReader)
	if !ok {
		return fmt.Errorf("group repository does not support schedule configuration")
	}
	return reader.ReplaceModelRouteConcurrencySchedules(ctx, groupID, routeAlias, accountID, schedules)
}
