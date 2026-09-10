//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminService_CreateGroup_PreservesOptionalSceneName(t *testing.T) {
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo}

	_, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
		Name:           "technical-group",
		SceneName:      "shared-scene",
		RateMultiplier: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, repo.created)
	require.Equal(t, "shared-scene", repo.created.SceneName)
}

func TestAdminService_UpdateGroup_AllowsDuplicateOrEmptySceneName(t *testing.T) {
	existing := &Group{ID: 1, Name: "technical-group", SceneName: "old-scene", RateMultiplier: 1, Status: StatusActive}
	repo := &groupRepoStubForAdmin{getByID: existing}
	svc := &adminServiceImpl{groupRepo: repo}
	empty := ""

	_, err := svc.UpdateGroup(context.Background(), 1, &UpdateGroupInput{SceneName: &empty})

	require.NoError(t, err)
	require.NotNil(t, repo.updated)
	require.Empty(t, repo.updated.SceneName)
}
