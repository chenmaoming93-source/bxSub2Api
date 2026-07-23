package rbac

import "context"

type AuthorizationRepository interface {
	LoadActiveGrants(ctx context.Context, userID int64) ([]Grant, error)
	GetUserVersion(ctx context.Context, userID int64) (int64, error)
	GetPolicyVersion(ctx context.Context) (int64, error)
}

type VersionMutationRepository interface {
	IncrementUserVersion(ctx context.Context, userID int64) error
	IncrementPolicyVersion(ctx context.Context) error
}
