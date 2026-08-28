package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	SecurityCheckConfigKeyPrefix = "sub2api:security-check:group:"
	SecurityCheckConfigChannel   = "sub2api:security-check:config-change"
	DefaultSecurityConfigTTL     = 5 * time.Second
)

// SecurityCheckConfigStore is the database source of truth for group policies.
type SecurityCheckConfigStore interface {
	GetSecurityCheckConfig(ctx context.Context, groupID int64) (domain.SecurityCheckConfig, error)
	UpdateSecurityCheckConfig(ctx context.Context, groupID int64, config domain.SecurityCheckConfig) error
}

type securityConfigCacheEntry struct {
	config   domain.SecurityCheckConfig
	loadedAt time.Time
}

// SecurityConfigProvider reads group policies from a local cache, Redis, then the database.
type SecurityConfigProvider struct {
	rdb   *redis.Client
	store SecurityCheckConfigStore
	ttl   time.Duration
	now   func() time.Time

	mu        sync.RWMutex
	local     map[int64]securityConfigCacheEntry
	lastValid map[int64]domain.SecurityCheckConfig
}

func NewSecurityConfigProvider(rdb *redis.Client, store SecurityCheckConfigStore, ttl time.Duration) *SecurityConfigProvider {
	if ttl <= 0 {
		ttl = DefaultSecurityConfigTTL
	}
	return &SecurityConfigProvider{
		rdb:       rdb,
		store:     store,
		ttl:       ttl,
		now:       time.Now,
		local:     make(map[int64]securityConfigCacheEntry),
		lastValid: make(map[int64]domain.SecurityCheckConfig),
	}
}

// Get returns a validated configuration without making cache failures fatal.
func (p *SecurityConfigProvider) Get(ctx context.Context, groupID int64) (domain.SecurityCheckConfig, error) {
	if groupID <= 0 {
		return domain.DefaultSecurityCheckConfig(), errors.New("invalid group id")
	}
	now := p.now()
	p.mu.RLock()
	entry, ok := p.local[groupID]
	stale, hasStale := p.lastValid[groupID]
	p.mu.RUnlock()
	if ok && now.Sub(entry.loadedAt) < p.ttl {
		return entry.config, nil
	}

	if p.rdb != nil {
		data, err := p.rdb.Get(ctx, securityConfigKey(groupID)).Bytes()
		if err == nil {
			config, parseErr := parseSecurityConfig(data)
			if parseErr == nil {
				p.saveLocal(groupID, config, now)
				return config, nil
			}
			slog.Warn("security_check_config.redis_decode_failed", "group_id", groupID, "err", parseErr)
		} else if !errors.Is(err, redis.Nil) {
			slog.Warn("security_check_config.redis_get_failed", "group_id", groupID, "err", err)
		}
	}

	if p.store != nil {
		config, err := p.store.GetSecurityCheckConfig(ctx, groupID)
		if err == nil {
			config = domain.NormalizeSecurityCheckConfig(config)
			if validationErr := domain.ValidateSecurityCheckConfig(config); validationErr == nil {
				p.saveLocal(groupID, config, now)
				if p.rdb != nil {
					if data, marshalErr := json.Marshal(config); marshalErr == nil {
						_ = p.rdb.Set(ctx, securityConfigKey(groupID), data, p.ttl).Err()
					}
				}
				return config, nil
			} else {
				slog.Warn("security_check_config.database_invalid", "group_id", groupID, "err", validationErr)
			}
		} else {
			slog.Warn("security_check_config.database_get_failed", "group_id", groupID, "err", err)
		}
	}

	if hasStale {
		return stale, nil
	}
	return domain.DefaultSecurityCheckConfig(), nil
}

// Update persists a validated policy, refreshes this instance, and notifies peers.
func (p *SecurityConfigProvider) Update(ctx context.Context, groupID int64, config domain.SecurityCheckConfig) error {
	config = domain.NormalizeSecurityCheckConfig(config)
	if err := domain.ValidateSecurityCheckConfig(config); err != nil {
		return err
	}
	if p.store == nil {
		return errors.New("security check config store is not configured")
	}
	if err := p.store.UpdateSecurityCheckConfig(ctx, groupID, config); err != nil {
		return err
	}
	p.saveLocal(groupID, config, p.now())
	if p.rdb == nil {
		return nil
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := p.rdb.Set(ctx, securityConfigKey(groupID), data, p.ttl).Err(); err != nil {
		return err
	}
	return p.publish(ctx, groupID, config.Version)
}

// Invalidate evicts one local entry. The next read will use Redis or the database.
func (p *SecurityConfigProvider) Invalidate(groupID int64) {
	p.mu.Lock()
	delete(p.local, groupID)
	p.mu.Unlock()
}

// Start subscribes to cross-instance configuration changes until ctx is cancelled.
func (p *SecurityConfigProvider) Start(ctx context.Context) {
	if p.rdb == nil {
		return
	}
	go func() {
		sub := p.rdb.Subscribe(ctx, SecurityCheckConfigChannel)
		defer func() { _ = sub.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-sub.Channel():
				if !ok || msg == nil {
					return
				}
				var change struct {
					GroupID int64 `json:"group_id"`
					Version int64 `json:"version"`
				}
				if err := json.Unmarshal([]byte(msg.Payload), &change); err != nil || change.GroupID <= 0 {
					continue
				}
				p.InvalidateIfOlder(change.GroupID, change.Version)
			}
		}
	}()
}

// InvalidateIfOlder evicts only when the notification is not older than local data.
func (p *SecurityConfigProvider) InvalidateIfOlder(groupID, version int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.local[groupID]
	if !ok || entry.config.Version <= version {
		delete(p.local, groupID)
	}
}

func (p *SecurityConfigProvider) saveLocal(groupID int64, config domain.SecurityCheckConfig, loadedAt time.Time) {
	p.mu.Lock()
	p.local[groupID] = securityConfigCacheEntry{config: config, loadedAt: loadedAt}
	p.lastValid[groupID] = config
	p.mu.Unlock()
}

func (p *SecurityConfigProvider) publish(ctx context.Context, groupID, version int64) error {
	payload, err := json.Marshal(struct {
		GroupID int64 `json:"group_id"`
		Version int64 `json:"version"`
	}{groupID, version})
	if err != nil {
		return err
	}
	return p.rdb.Publish(ctx, SecurityCheckConfigChannel, payload).Err()
}

func parseSecurityConfig(data []byte) (domain.SecurityCheckConfig, error) {
	var config domain.SecurityCheckConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	config = domain.NormalizeSecurityCheckConfig(config)
	if err := domain.ValidateSecurityCheckConfig(config); err != nil {
		return config, err
	}
	return config, nil
}

func securityConfigKey(groupID int64) string {
	return SecurityCheckConfigKeyPrefix + strconv.FormatInt(groupID, 10)
}
