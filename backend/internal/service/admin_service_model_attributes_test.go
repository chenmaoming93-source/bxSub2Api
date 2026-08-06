//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

// accountRepoStubForModelAttributes 覆盖 UpdateAccount 用到的 GetByID / Update。
type accountRepoStubForModelAttributes struct {
	mockAccountRepoForGemini
	account     *Account
	updateCalls int
}

func (r *accountRepoStubForModelAttributes) GetByID(ctx context.Context, id int64) (*Account, error) {
	return r.account, nil
}

func (r *accountRepoStubForModelAttributes) Update(ctx context.Context, account *Account) error {
	r.updateCalls++
	r.account = account
	return nil
}

func newBaseAccountWithModelAttributes() *Account {
	return &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		ModelAttributes: domain.ModelAttributes{
			"context_window": {Description: "上下文窗口总大小（token）", Value: 200000},
		},
	}
}

func TestAdminService_UpdateAccount_ModelAttributes(t *testing.T) {
	t.Run("provided map overrides with normalize and keeps value as-is", func(t *testing.T) {
		repo := &accountRepoStubForModelAttributes{account: newBaseAccountWithModelAttributes()}
		svc := &adminServiceImpl{accountRepo: repo}
		in := &UpdateAccountInput{
			ModelAttributes: domain.ModelAttributes{
				"  supports_vision  ": {Description: "支持图片输入", Value: "true"}, // 字符串 "true" 不得改写为布尔
				"   ":                {Description: "should be dropped", Value: "x"},
			},
		}
		updated, err := svc.UpdateAccount(context.Background(), 1, in)
		require.NoError(t, err)
		require.Equal(t, 1, repo.updateCalls)
		// Normalize 生效：空 key 丢弃、key 去首尾空白
		require.Contains(t, updated.ModelAttributes, "supports_vision")
		require.NotContains(t, updated.ModelAttributes, "   ")
		// 覆盖语义：原 context_window 被替换，而非合并
		require.NotContains(t, updated.ModelAttributes, "context_window")
		// 信任前端：value 原样保留字符串 "true"
		require.Equal(t, "true", updated.ModelAttributes["supports_vision"].Value)
		require.Equal(t, "支持图片输入", updated.ModelAttributes["supports_vision"].Description)
	})

	t.Run("empty map clears attributes", func(t *testing.T) {
		repo := &accountRepoStubForModelAttributes{account: newBaseAccountWithModelAttributes()}
		svc := &adminServiceImpl{accountRepo: repo}
		updated, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{ModelAttributes: domain.ModelAttributes{}})
		require.NoError(t, err)
		require.NotNil(t, updated.ModelAttributes)
		require.Empty(t, updated.ModelAttributes)
	})

	t.Run("nil keeps existing attributes", func(t *testing.T) {
		repo := &accountRepoStubForModelAttributes{account: newBaseAccountWithModelAttributes()}
		svc := &adminServiceImpl{accountRepo: repo}
		updated, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{Name: "renamed"})
		require.NoError(t, err)
		require.Equal(t, "renamed", updated.Name)
		require.Equal(t, 200000, updated.ModelAttributes["context_window"].Value)
		require.Len(t, updated.ModelAttributes, 1)
	})
}
