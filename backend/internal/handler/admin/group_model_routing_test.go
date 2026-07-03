package admin

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
)

func TestGroupRequestsAcceptLegacyAndCandidateModelRouting(t *testing.T) {
	tests := []string{
		`{"name":"legacy","model_routing":{"claude-opus-*":[9,3,7]}}`,
		`{"name":"new","model_routing":{"fast-code":[{"model":"gpt-5","account_ids":[4,5],"priority":2,"daily_token_limit":600}]}}`,
	}
	for _, body := range tests {
		var req CreateGroupRequest
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			t.Fatalf("json.Unmarshal(%s) error = %v", body, err)
		}
		if err := validateModelRoutingJSON(req.ModelRouting); err != nil {
			t.Fatalf("validateModelRoutingJSON(%s) error = %v", body, err)
		}
	}
}

func TestGroupRequestRejectsInvalidCandidateModelRouting(t *testing.T) {
	var req UpdateGroupRequest
	if err := json.Unmarshal([]byte(`{"model_routing":{"fast-code":[{"model":"","account_ids":[],"priority":-1}]}}`), &req); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if err := validateModelRoutingJSON(req.ModelRouting); err == nil {
		t.Fatal("validateModelRoutingJSON() error = nil, want validation error")
	}
}

func TestAdminGroupExposesRoutingWhilePublicGroupDoesNot(t *testing.T) {
	routing := domain.NewModelRoutingJSON([]byte(`{"fast-code":[{"model":"gpt-5","account_ids":[4],"priority":1,"daily_token_limit":600}]}`))
	adminJSON, err := json.Marshal(dto.AdminGroup{ModelRouting: routing})
	if err != nil {
		t.Fatalf("json.Marshal(AdminGroup) error = %v", err)
	}
	var adminObject map[string]any
	if err := json.Unmarshal(adminJSON, &adminObject); err != nil {
		t.Fatalf("json.Unmarshal(AdminGroup) error = %v", err)
	}
	if _, ok := adminObject["model_routing"]; !ok {
		t.Fatalf("AdminGroup JSON has no model_routing: %s", adminJSON)
	}

	publicJSON, err := json.Marshal(dto.Group{})
	if err != nil {
		t.Fatalf("json.Marshal(Group) error = %v", err)
	}
	var publicObject map[string]any
	if err := json.Unmarshal(publicJSON, &publicObject); err != nil {
		t.Fatalf("json.Unmarshal(Group) error = %v", err)
	}
	if _, ok := publicObject["model_routing"]; ok {
		t.Fatalf("public Group leaked model_routing: %s", publicJSON)
	}
}
