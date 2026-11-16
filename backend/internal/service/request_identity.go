package service

import "context"

type superAdminContextKey struct{}

func WithSuperAdminExemption(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, superAdminContextKey{}, enabled)
}

func IsSuperAdminExempt(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(superAdminContextKey{}).(bool)
	return enabled
}
