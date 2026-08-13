package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

// 复用 external_provisioning_service_test.go 中的 epGroupRepoStub / epAccountRepoStub。

func modelRoutesWithAttributesTestFixtures() (*epAccountRepoStub, *Group) {
	accounts := map[int64]*Account{
		1: {
			ID:        1,
			Platform:  PlatformAnthropic,
			Type:      AccountTypeAPIKey,
			Status:    StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"m1": "claude-3-5-sonnet"},
			},
			ModelAttributes: domain.ModelAttributes{
				"context_window": {Description: "上下文窗口", Value: 200000},
			},
		},
		2: {
			ID:        2,
			Platform:  PlatformAnthropic,
			Type:      AccountTypeAPIKey,
			Status:    StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"m2": "claude-3-5-sonnet"}, // 同名模型，未配置属性
			},
		},
		3: {
			ID:        3,
			Platform:  PlatformOpenAI,
			Type:      AccountTypeAPIKey,
			Status:    StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"m3": "gpt-5.2"},
			},
			ModelAttributes: domain.ModelAttributes{
				"supports_vision": {Description: "支持图片输入", Value: true},
			},
		},
		4: {
			ID:        4,
			Platform:  PlatformOpenAI,
			Type:      AccountTypeAPIKey,
			Status:    StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"m4": "   "}, // 模型名为空 → 跳过
			},
		},
	}
	group := &Group{
		ID:   10,
		Name: "public",
		ModelRouting: map[string]any{
			"alpha": []map[string]any{
				{"account_ids": []int64{1, 2}, "priority": 1},
				{"account_ids": []int64{3, 4}, "priority": 2},
			},
		},
	}
	return &epAccountRepoStub{accounts: accounts}, group
}

func TestListGroupModelRoutesWithAttributes(t *testing.T) {
	accounts, group := modelRoutesWithAttributesTestFixtures()
	svc := NewExternalProvisioningService(nil, nil, nil, nil, epGroups(group), accounts)

	routes, err := svc.ListGroupModelRoutesWithAttributes(context.Background(), ListGroupModelRoutesInput{GroupName: " public "})
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Equal(t, "alpha", routes[0].RouteAlias)

	items := routes[0].UpstreamModels
	// 完全不去重：账号 1、2 同名模型各一条；账号 4 模型名为空被跳过。
	require.Len(t, items, 3)

	// 账号 1：模型名 + 属性
	require.Equal(t, "claude-3-5-sonnet", items[0].Model)
	require.Equal(t, 200000, items[0].Attributes["context_window"].Value)

	// 账号 2：同名模型不去重；无属性 → 空对象（非 nil）
	require.Equal(t, "claude-3-5-sonnet", items[1].Model)
	require.NotNil(t, items[1].Attributes)
	require.Len(t, items[1].Attributes, 0)

	// 账号 3：另一模型
	require.Equal(t, "gpt-5.2", items[2].Model)
	require.Equal(t, true, items[2].Attributes["supports_vision"].Value)
}

func TestListGroupModelRoutesWithAttributes_AccountMissingAndEmptyMapping(t *testing.T) {
	// 候选引用不存在的账号 ID（5）与模型名为空的账号（4）都应被跳过。
	group := &Group{
		ID:   10,
		Name: "public",
		ModelRouting: map[string]any{
			"alpha": []map[string]any{
				{"account_ids": []int64{5}, "priority": 1},
				{"account_ids": []int64{4}, "priority": 2},
			},
		},
	}
	accounts := map[int64]*Account{
		4: {
			ID:        4,
			Credentials: map[string]any{"model_mapping": map[string]any{"m4": "   "}},
		},
	}
	svc := NewExternalProvisioningService(nil, nil, nil, nil, epGroups(group), &epAccountRepoStub{accounts: accounts})

	routes, err := svc.ListGroupModelRoutesWithAttributes(context.Background(), ListGroupModelRoutesInput{GroupName: "public"})
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Empty(t, routes[0].UpstreamModels)
}

func TestListGroupModelRoutesWithAttributes_ErrorsAndEmpty(t *testing.T) {
	t.Run("group not found", func(t *testing.T) {
		svc := NewExternalProvisioningService(nil, nil, nil, nil, epGroups(), nil)
		_, err := svc.ListGroupModelRoutesWithAttributes(context.Background(), ListGroupModelRoutesInput{GroupName: "missing"})
		require.ErrorIs(t, err, ErrGroupNotFound)
	})

	t.Run("no routing configured returns empty routes", func(t *testing.T) {
		svc := NewExternalProvisioningService(nil, nil, nil, nil, epGroups(&Group{ID: 1, Name: "plain"}), nil)
		routes, err := svc.ListGroupModelRoutesWithAttributes(context.Background(), ListGroupModelRoutesInput{GroupName: "plain"})
		require.NoError(t, err)
		require.Empty(t, routes)
	})

	t.Run("account lookup unavailable returns empty upstream list", func(t *testing.T) {
		group := &Group{
			ID:   1,
			Name: "g",
			ModelRouting: map[string]any{
				"alpha": []map[string]any{{"account_ids": []int64{1}, "priority": 1}},
			},
		}
		svc := NewExternalProvisioningService(nil, nil, nil, nil, epGroups(group), nil) // accounts nil
		routes, err := svc.ListGroupModelRoutesWithAttributes(context.Background(), ListGroupModelRoutesInput{GroupName: "g"})
		require.NoError(t, err)
		require.Len(t, routes, 1)
		require.Empty(t, routes[0].UpstreamModels)
	})
}

// TestListGroupModelRoutes_AfterRefactor 回归：原接口在抽取公共 helper 后行为不变
// （按模型名去重，输出字符串列表）。
func TestListGroupModelRoutes_AfterRefactor(t *testing.T) {
	accounts, group := modelRoutesWithAttributesTestFixtures()
	svc := NewExternalProvisioningService(nil, nil, nil, nil, epGroups(group), accounts)

	routes, err := svc.ListGroupModelRoutes(context.Background(), ListGroupModelRoutesInput{GroupName: "public"})
	require.NoError(t, err)
	require.Len(t, routes, 1)
	// 原语义：按模型名去重 → [claude-3-5-sonnet, gpt-5.2]
	require.Equal(t, []string{"claude-3-5-sonnet", "gpt-5.2"}, routes[0].UpstreamModels)
}
