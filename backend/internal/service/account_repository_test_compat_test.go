//go:build unit

package service

import "context"

// ExistsByName is not exercised by these legacy gateway/account mocks. Keep the
// compatibility methods isolated here so the mocks remain focused on the
// behavior each test actually covers.

func (*groupAwareMockAccountRepo) ExistsByName(context.Context, string, int64) (bool, error) {
	return false, nil
}

func (*mockAccountRepoForPlatform) ExistsByName(context.Context, string, int64) (bool, error) {
	return false, nil
}

func (*mockAccountRepoForGemini) ExistsByName(context.Context, string, int64) (bool, error) {
	return false, nil
}

func (*accountRepoStub) ExistsByName(context.Context, string, int64) (bool, error) {
	return false, nil
}

func (*openAIAccountTestRepo) ExistsByName(context.Context, string, int64) (bool, error) {
	return false, nil
}
