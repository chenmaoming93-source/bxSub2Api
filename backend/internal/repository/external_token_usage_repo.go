package repository

import (
	"context"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type externalTokenUsageDimensionRepository struct{ client *dbent.Client }

func NewExternalTokenUsageDimensionRepository(client *dbent.Client) service.ExternalTokenUsageDimensionLookup {
	return &externalTokenUsageDimensionRepository{client: client}
}

func (r *externalTokenUsageDimensionRepository) FindUserByEmail(ctx context.Context, email string) (*service.User, error) {
	matches, err := r.client.User.Query().Where(userEmailLookupPredicate(email)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, service.ErrUserNotFound
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("normalized email lookup matched multiple users for %q", strings.TrimSpace(email))
	}
	return userEntityToService(matches[0]), nil
}

func (r *externalTokenUsageDimensionRepository) FindGroupByName(ctx context.Context, name string) (*service.Group, error) {
	m, err := r.client.Group.Query().Where(group.NameEQ(strings.TrimSpace(name)), group.DeletedAtIsNil()).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrGroupNotFound, nil)
	}
	return groupEntityToService(m), nil
}

func (r *externalTokenUsageDimensionRepository) FindAPIKeyByKey(ctx context.Context, key string) (*service.APIKey, error) {
	// api_keys.key 具有数据库唯一约束（见 ent/schema/api_key.go），直接按值精确查询。
	m, err := r.client.APIKey.Query().Where(
		apikey.KeyEQ(strings.TrimSpace(key)), apikey.DeletedAtIsNil(),
	).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAPIKeyNotFound, nil)
	}
	return apiKeyEntityToService(m), nil
}
