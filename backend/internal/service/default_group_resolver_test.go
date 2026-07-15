package service

import (
	"context"
	"errors"
	"testing"
)

type defaultGroupSettingsStub struct{ name string }

func (s defaultGroupSettingsStub) GetDefaultGroupName(context.Context) string { return s.name }

type defaultGroupLookupStub struct {
	group *Group
	err   error
	name  string
}

func (s *defaultGroupLookupStub) GetByNameExact(_ context.Context, name string) (*Group, error) {
	s.name = name
	return s.group, s.err
}

func TestDefaultGroupResolverStates(t *testing.T) {
	t.Run("unconfigured", func(t *testing.T) {
		lookup := &defaultGroupLookupStub{err: errors.New("must not be called")}
		result, err := NewDefaultGroupResolver(defaultGroupSettingsStub{name: "  "}, lookup).Resolve(context.Background())
		if err != nil || result.State != DefaultGroupUnconfigured || lookup.name != "" {
			t.Fatalf("result=%+v lookup=%q err=%v", result, lookup.name, err)
		}
	})

	t.Run("configured but missing", func(t *testing.T) {
		lookup := &defaultGroupLookupStub{err: ErrGroupNotFound}
		result, err := NewDefaultGroupResolver(defaultGroupSettingsStub{name: " default "}, lookup).Resolve(context.Background())
		if err != nil || result.State != DefaultGroupMissing || result.Name != "default" || result.Group != nil {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})

	t.Run("found", func(t *testing.T) {
		group := &Group{ID: 7, Name: "default"}
		result, err := NewDefaultGroupResolver(defaultGroupSettingsStub{name: "default"}, &defaultGroupLookupStub{group: group}).Resolve(context.Background())
		if err != nil || result.State != DefaultGroupFound || result.Group != group {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		want := errors.New("database unavailable")
		_, err := NewDefaultGroupResolver(defaultGroupSettingsStub{name: "default"}, &defaultGroupLookupStub{err: want}).Resolve(context.Background())
		if !errors.Is(err, want) {
			t.Fatalf("err=%v, want wrapped %v", err, want)
		}
	})
}
