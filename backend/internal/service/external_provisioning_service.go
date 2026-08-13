package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
)

var (
	ErrProvisioningGroupInactive     = infraerrors.Conflict("GROUP_INACTIVE", "group is inactive")
	ErrProvisioningSubscriptionGroup = infraerrors.BadRequest("SUBSCRIPTION_GROUP_NOT_SUPPORTED", "subscription groups are not supported")
	ErrProvisioningGroupNotAllowed   = infraerrors.Forbidden("GROUP_NOT_ALLOWED", "user is not allowed to use this group")
)

// ExternalProvisioningService orchestrates user lookup, LDAP fallback,
// provisioning, and platform-key retrieval for external API callers.
type ExternalProvisioningService struct {
	users        ExternalProvisioningUserLookup
	ldap         ExternalProvisioningLDAPDirectory
	provisioner  ExternalUserProvisioner
	platformKeys *PlatformAPIKeyService
	groups       ExternalProvisioningGroupLookup
	accounts     ExternalProvisioningAccountLookup
}

// ExternalProvisioningUserLookup is the narrow user lookup needed by the
// external provisioning flow.
type ExternalProvisioningUserLookup interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// ExternalProvisioningLDAPDirectory exposes only the directory lookup used by
// the external provisioning flow.
type ExternalProvisioningLDAPDirectory interface {
	LookupUser(ctx context.Context, username string) (*ldapauth.User, error)
}

// ExternalUserProvisioner creates or retrieves a local user.
type ExternalUserProvisioner interface {
	Provision(ctx context.Context, input UserProvisioningInput) (*UserProvisioningResult, error)
}

type ExternalProvisioningGroupLookup interface {
	GetByNameExact(ctx context.Context, name string) (*Group, error)
}

// ExternalProvisioningAccountLookup resolves route-candidate accounts so the
// external model-routes listing can derive upstream models from each account's
// own model_mapping (same source as runtime account-bound routing).
type ExternalProvisioningAccountLookup interface {
	GetByIDs(ctx context.Context, ids []int64) ([]*Account, error)
}

// NewExternalProvisioningService constructs the orchestration service.
func NewExternalProvisioningService(
	users ExternalProvisioningUserLookup,
	ldap ExternalProvisioningLDAPDirectory,
	provisioner ExternalUserProvisioner,
	platformKeys *PlatformAPIKeyService,
	groups ExternalProvisioningGroupLookup,
	accounts ExternalProvisioningAccountLookup,
) *ExternalProvisioningService {
	return &ExternalProvisioningService{
		users:        users,
		ldap:         ldap,
		provisioner:  provisioner,
		platformKeys: platformKeys,
		groups:       groups,
		accounts:     accounts,
	}
}

// EnsurePlatformKeyInput carries the external request payload.
type EnsurePlatformKeyInput struct {
	User      string
	Platform  string
	GroupName string
}

// EnsurePlatformKeyResult carries the response.
type EnsurePlatformKeyResult struct {
	User        *User
	APIKey      *APIKey
	UserCreated bool
	KeyCreated  bool
	Group       *Group
}

type ListGroupModelRoutesInput struct {
	GroupName string
}

type GroupModelRouteProjection struct {
	RouteAlias     string
	UpstreamModels []string
}

type GroupModelRouteItemWithAttributes struct {
	Model      string                 `json:"model"`
	Attributes domain.ModelAttributes `json:"attributes"`
}

type GroupModelRoutesWithAttributesProjection struct {
	RouteAlias     string
	UpstreamModels []GroupModelRouteItemWithAttributes
}

func (s *ExternalProvisioningService) ListGroupModelRoutes(ctx context.Context, input ListGroupModelRoutesInput) ([]GroupModelRouteProjection, error) {
	config, err := s.resolveGroupModelRoutingConfig(ctx, input.GroupName)
	if err != nil {
		return nil, err
	}
	accountsByID := s.loadRouteAccountMap(ctx, config)
	aliases := sortedRouteAliases(config)

	result := make([]GroupModelRouteProjection, 0, len(aliases))
	for _, alias := range aliases {
		result = append(result, GroupModelRouteProjection{RouteAlias: alias, UpstreamModels: routeUpstreamModels(config[alias], accountsByID)})
	}
	return result, nil
}

// ListGroupModelRoutesWithAttributes 与 ListGroupModelRoutes 逻辑一致，但有两处不同：
//  1. upstream_models 从字符串列表变为 {model, attributes} 对象列表，attributes 为
//     该账号的模型基本属性（账号未配置属性时输出空对象 {}）；
//  2. 不做模型名去重：同一模型名可重复列出，每个候选账号一条（账号级去重保留，
//     同一账号只输出一次）。
func (s *ExternalProvisioningService) ListGroupModelRoutesWithAttributes(ctx context.Context, input ListGroupModelRoutesInput) ([]GroupModelRoutesWithAttributesProjection, error) {
	config, err := s.resolveGroupModelRoutingConfig(ctx, input.GroupName)
	if err != nil {
		return nil, err
	}
	accountsByID := s.loadRouteAccountMap(ctx, config)
	aliases := sortedRouteAliases(config)

	result := make([]GroupModelRoutesWithAttributesProjection, 0, len(aliases))
	for _, alias := range aliases {
		result = append(result, GroupModelRoutesWithAttributesProjection{
			RouteAlias:     alias,
			UpstreamModels: routeUpstreamModelItems(config[alias], accountsByID),
		})
	}
	return result, nil
}

// resolveGroupModelRoutingConfig 按名称查找分组并解析其模型路由配置（兼容 legacy 与
// 候选对象两种存储格式）。分组不存在返回 ErrGroupNotFound；未配置路由返回空配置。
func (s *ExternalProvisioningService) resolveGroupModelRoutingConfig(ctx context.Context, groupName string) (domain.ModelRoutingConfig, error) {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return nil, fmt.Errorf("group_name is required")
	}
	if s.groups == nil {
		return nil, fmt.Errorf("group lookup is not configured")
	}
	group, err := s.groups.GetByNameExact(ctx, groupName)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("lookup group routes: %w", err)
	}
	if group.ModelRouting == nil {
		return domain.ModelRoutingConfig{}, nil
	}
	data, err := json.Marshal(group.ModelRouting)
	if err != nil {
		return nil, fmt.Errorf("encode group model routing: %w", err)
	}
	config, err := domain.ParseModelRoutingConfig(data)
	if err != nil {
		return nil, fmt.Errorf("parse group model routing: %w", err)
	}
	return config, nil
}

// sortedRouteAliases 返回配置中的所有别名，按字典序排序，保证输出确定性。
func sortedRouteAliases(config domain.ModelRoutingConfig) []string {
	aliases := make([]string, 0, len(config))
	for alias := range config {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

// loadRouteAccountMap 汇总配置中所有候选账号 ID（去重）后批量查询一次，建立
// ID → 账号映射。账号查询不可用或失败时返回 nil（调用方按空映射处理）。
func (s *ExternalProvisioningService) loadRouteAccountMap(ctx context.Context, config domain.ModelRoutingConfig) map[int64]*Account {
	if s.accounts == nil {
		return nil
	}
	seenID := make(map[int64]struct{})
	for _, candidates := range config {
		for _, candidate := range candidates {
			for _, id := range candidate.AccountIDs {
				if _, exists := seenID[id]; exists {
					continue
				}
				seenID[id] = struct{}{}
			}
		}
	}
	if len(seenID) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(seenID))
	for id := range seenID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	accounts, err := s.accounts.GetByIDs(ctx, ids)
	if err != nil {
		return nil
	}
	accountsByID := make(map[int64]*Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			accountsByID[account.ID] = account
		}
	}
	return accountsByID
}

// routeUpstreamModels 按候选 priority 顺序（已由 ParseModelRoutingConfig 稳定排序）收集该路由
// 别名下的候选账号，并从每个账号自身的 model_mapping（FirstModelMappingValue）解析上游模型名，
// 去重后返回。账号查询不可用或失败时返回空列表；缺失账号被忽略。
func routeUpstreamModels(candidates []domain.ModelRouteCandidate, accountsByID map[int64]*Account) []string {
	if len(accountsByID) == 0 {
		return []string{}
	}
	modelSet := make(map[string]struct{})
	models := make([]string, 0, len(candidates))
	seenID := make(map[int64]struct{})
	for _, candidate := range candidates {
		for _, id := range candidate.AccountIDs {
			if _, exists := seenID[id]; exists {
				continue
			}
			seenID[id] = struct{}{}
			account := accountsByID[id]
			if account == nil {
				continue
			}
			model := strings.TrimSpace(account.FirstModelMappingValue())
			if model == "" {
				continue
			}
			if _, exists := modelSet[model]; exists {
				continue
			}
			modelSet[model] = struct{}{}
			models = append(models, model)
		}
	}
	return models
}

// routeUpstreamModelItems 按候选 priority 顺序收集该路由别名下每个候选账号的
// {模型名, 模型基本属性} 条目。与原 routeUpstreamModels 的区别：
//   - 不做模型名去重（同一模型名可重复列出，每个候选账号一条）；
//   - 每条附带账号的模型基本属性（ModelAttributes），未配置时输出空对象 {}；
//   - 账号级去重（seenID）保留：同一账号只输出一次。
func routeUpstreamModelItems(candidates []domain.ModelRouteCandidate, accountsByID map[int64]*Account) []GroupModelRouteItemWithAttributes {
	if len(accountsByID) == 0 {
		return []GroupModelRouteItemWithAttributes{}
	}
	items := make([]GroupModelRouteItemWithAttributes, 0, len(candidates))
	seenID := make(map[int64]struct{})
	for _, candidate := range candidates {
		for _, id := range candidate.AccountIDs {
			if _, exists := seenID[id]; exists {
				continue
			}
			seenID[id] = struct{}{}
			account := accountsByID[id]
			if account == nil {
				continue
			}
			model := strings.TrimSpace(account.FirstModelMappingValue())
			if model == "" {
				continue
			}
			attributes := account.ModelAttributes
			if attributes == nil {
				attributes = domain.ModelAttributes{}
			}
			items = append(items, GroupModelRouteItemWithAttributes{Model: model, Attributes: attributes})
		}
	}
	return items
}

// EnsurePlatformKey resolves a user (local or LDAP) and returns a
// platform-scoped API key. It returns ErrUserNotFound when neither local nor
// LDAP lookup succeeds, and the usual service errors for database or
// provisioning failures.
func (s *ExternalProvisioningService) EnsurePlatformKey(ctx context.Context, input EnsurePlatformKeyInput) (*EnsurePlatformKeyResult, error) {
	if err := ValidatePlatform(input.Platform); err != nil {
		return nil, fmt.Errorf("validate platform: %w", err)
	}

	email := strings.TrimSpace(strings.ToLower(input.User))
	if email == "" {
		return nil, fmt.Errorf("user_email is required")
	}
	groupName := strings.TrimSpace(input.GroupName)
	if groupName == "" {
		return nil, fmt.Errorf("group_name is required")
	}
	group, err := s.resolveAllowedGroup(ctx, groupName)
	if err != nil {
		return nil, err
	}

	// 1. Try local user lookup.
	user, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return s.ensureKeyForUser(ctx, user, input.Platform, group, false)
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("lookup local user: %w", err)
	}

	// 2. LDAP fallback.
	if s.ldap == nil {
		return nil, ErrUserNotFound
	}
	ldapUser, ldapErr := s.ldap.LookupUser(ctx, email)
	if ldapErr != nil {
		if errors.Is(ldapErr, ldapauth.ErrDirectoryNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("ldap lookup: %w", ldapErr)
	}

	// 3. Provision local user from LDAP identity.
	if s.provisioner == nil {
		return nil, fmt.Errorf("provisioner is not configured")
	}
	account := strings.TrimSpace(strings.ToLower(ldapUser.Username))
	if account == "" {
		account = email
	}
	displayName := strings.TrimSpace(ldapUser.DisplayName)
	if displayName == "" {
		displayName = account
	}
	if len([]rune(displayName)) > 100 {
		displayName = string([]rune(displayName)[:100])
	}

	result, err := s.provisioner.Provision(ctx, UserProvisioningInput{
		Email:        email,
		Username:     displayName,
		SignupSource: "ldap",
		Role:         RoleUser,
		Status:       StatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("provision ldap user: %w", err)
	}

	return s.ensureKeyForUser(ctx, result.User, input.Platform, group, result.Created)
}

func (s *ExternalProvisioningService) resolveAllowedGroup(ctx context.Context, groupName string) (*Group, error) {
	if s.groups == nil {
		return nil, fmt.Errorf("group lookup is not configured")
	}
	group, err := s.groups.GetByNameExact(ctx, groupName)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("lookup group: %w", err)
	}
	if !group.IsActive() {
		return nil, ErrProvisioningGroupInactive
	}
	if group.IsSubscriptionType() {
		return nil, ErrProvisioningSubscriptionGroup
	}
	return group, nil
}

func (s *ExternalProvisioningService) ensureKeyForUser(ctx context.Context, user *User, platform string, group *Group, userCreated bool) (*EnsurePlatformKeyResult, error) {
	if !user.IsActive() {
		return nil, fmt.Errorf("user %d is not active", user.ID)
	}
	if !user.CanBindGroup(group.ID, group.IsExclusive) {
		return nil, ErrProvisioningGroupNotAllowed
	}

	if s.platformKeys == nil {
		return nil, fmt.Errorf("platform key service is not configured")
	}
	key, err := s.platformKeys.GetOrCreatePlatformKey(ctx, user.ID, platform, group.ID)
	if err != nil {
		return nil, fmt.Errorf("get or create platform key: %w", err)
	}

	// A key is newly created when either the user was just created or the
	// key's creation timestamp is later than the user's.
	keyCreated := userCreated || key.CreatedAt.After(user.CreatedAt)

	return &EnsurePlatformKeyResult{
		User:        user,
		APIKey:      key,
		UserCreated: userCreated,
		KeyCreated:  keyCreated,
		Group:       group,
	}, nil
}
