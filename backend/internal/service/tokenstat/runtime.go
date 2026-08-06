package tokenstat

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/redis/go-redis/v9"
)

const (
	runtimeEnabledKey     = "sub2api:dynamic_token_stats_runtime:v1:enabled"
	runtimeChangedChannel = "sub2api:dynamic_token_stats_runtime:v1:changed"
)

var defaultRuntimeController atomic.Pointer[RuntimeController]

// RuntimeController is the process-local, non-blocking gate for both dynamic
// token accounting and quota enforcement. Redis is only used by admin updates
// and cross-instance notification, never by the request-path gate itself.
type RuntimeController struct {
	redis     *redis.Client
	available bool
	enabled   atomic.Bool
}

type RuntimeState struct {
	Available bool `json:"available"`
	Enabled   bool `json:"enabled"`
}

func NewRuntimeController(client *redis.Client, cfg *config.Config) *RuntimeController {
	available := cfg != nil && cfg.Gateway.DynamicTokenStatistics.Enabled
	controller := &RuntimeController{redis: client, available: available}
	controller.enabled.Store(available)
	if available && client != nil {
		if raw, err := client.Get(context.Background(), runtimeEnabledKey).Result(); err == nil {
			if enabled, parseErr := strconv.ParseBool(raw); parseErr == nil {
				controller.enabled.Store(enabled)
			}
		}
		controller.subscribe()
	}
	defaultRuntimeController.Store(controller)
	return controller
}

func RuntimeEnabled() bool {
	controller := defaultRuntimeController.Load()
	// A nil controller is the test/standalone compatibility state. Production
	// always installs a controller during dependency initialization.
	return controller == nil || controller.Enabled()
}

func (c *RuntimeController) Enabled() bool {
	return c != nil && c.available && c.enabled.Load()
}

func (c *RuntimeController) State() RuntimeState {
	if c == nil {
		return RuntimeState{}
	}
	return RuntimeState{Available: c.available, Enabled: c.Enabled()}
}

func (c *RuntimeController) SetEnabled(ctx context.Context, enabled bool) error {
	if c == nil || !c.available {
		return fmt.Errorf("dynamic token statistics is disabled by startup configuration")
	}
	if c.redis == nil {
		c.enabled.Store(enabled)
		return nil
	}
	raw := strconv.FormatBool(enabled)
	if err := c.redis.Set(ctx, runtimeEnabledKey, raw, 0).Err(); err != nil {
		return fmt.Errorf("persist dynamic token statistics runtime state: %w", err)
	}
	c.enabled.Store(enabled)
	if err := c.redis.Publish(ctx, runtimeChangedChannel, raw).Err(); err != nil {
		return fmt.Errorf("publish dynamic token statistics runtime state: %w", err)
	}
	return nil
}

func (c *RuntimeController) subscribe() {
	go func() {
		pubsub := c.redis.Subscribe(context.Background(), runtimeChangedChannel)
		defer pubsub.Close()
		for message := range pubsub.Channel() {
			if enabled, err := strconv.ParseBool(message.Payload); err == nil {
				c.enabled.Store(enabled)
			}
		}
	}()
}
