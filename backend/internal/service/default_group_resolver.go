package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type DefaultGroupState string

const (
	DefaultGroupUnconfigured DefaultGroupState = "unconfigured"
	DefaultGroupMissing      DefaultGroupState = "missing"
	DefaultGroupFound        DefaultGroupState = "found"
)

type DefaultGroupResult struct {
	State DefaultGroupState
	Name  string
	Group *Group
}

type DefaultGroupNameReader interface {
	GetDefaultGroupName(ctx context.Context) string
}

type DefaultGroupNameLookup interface {
	GetByNameExact(ctx context.Context, name string) (*Group, error)
}

type DefaultGroupResolver struct {
	settings DefaultGroupNameReader
	groups   DefaultGroupNameLookup
}

func NewDefaultGroupResolver(settings DefaultGroupNameReader, groups DefaultGroupNameLookup) *DefaultGroupResolver {
	return &DefaultGroupResolver{settings: settings, groups: groups}
}

func (r *DefaultGroupResolver) Resolve(ctx context.Context) (DefaultGroupResult, error) {
	name := strings.TrimSpace(r.settings.GetDefaultGroupName(ctx))
	if name == "" {
		return DefaultGroupResult{State: DefaultGroupUnconfigured}, nil
	}

	group, err := r.groups.GetByNameExact(ctx, name)
	if errors.Is(err, ErrGroupNotFound) {
		return DefaultGroupResult{State: DefaultGroupMissing, Name: name}, nil
	}
	if err != nil {
		return DefaultGroupResult{}, fmt.Errorf("resolve default group %q: %w", name, err)
	}
	return DefaultGroupResult{State: DefaultGroupFound, Name: name, Group: group}, nil
}
