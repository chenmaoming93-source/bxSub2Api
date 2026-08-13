package handler

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestContainsAccountID(t *testing.T) {
	if !containsAccountID([]int64{3, 7}, 3) || containsAccountID([]int64{3, 7}, 4) {
		t.Fatal("containsAccountID returned an unexpected result")
	}
}

func TestNoAvailableAccountsMessage(t *testing.T) {
	group := &service.Group{ID: 6, Name: "BaixinDefaultGroup"}

	cases := []struct {
		name     string
		model    string
		routed   bool
		group    *service.Group
		wantPart string
	}{
		{
			name:     "route hit with group",
			model:    "test",
			routed:   true,
			group:    group,
			wantPart: "test",
		},
		{
			name:     "route hit without group",
			model:    "chat",
			routed:   true,
			group:    nil,
			wantPart: "chat",
		},
		{
			name:     "plain no group name",
			model:    "test",
			routed:   false,
			group:    group,
			wantPart: "BaixinDefaultGroup",
		},
		{
			name:     "plain without group",
			model:    "test",
			routed:   false,
			group:    nil,
			wantPart: "test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := noAvailableAccountsMessage(tc.model, tc.routed, tc.group)
			if msg == "" || !strings.Contains(msg, tc.wantPart) {
				t.Fatalf("message = %q, want it to contain %q", msg, tc.wantPart)
			}
			if !strings.Contains(msg, tc.model) {
				t.Fatalf("message = %q, want it to contain model %q", msg, tc.model)
			}
		})
	}
}
