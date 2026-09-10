package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"golang.org/x/sync/singleflight"
)

// ConcurrencyCache 定义并发控制的缓存接口
// 使用有序集合存储槽位，按时间戳清理过期条目
type ConcurrencyCache interface {
	// 账号槽位管理
	// 键格式: concurrency:account:{accountID}（有序集合，成员为 requestID）
	AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error)
	ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error
	GetAccountConcurrency(ctx context.Context, accountID int64) (int, error)
	GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error)

	// 账号等待队列（账号级）
	IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error)
	DecrementAccountWaitCount(ctx context.Context, accountID int64) error
	GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error)

	// 用户槽位管理
	// 键格式: concurrency:user:{userID}（有序集合，成员为 requestID）
	AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error)
	ReleaseUserSlot(ctx context.Context, userID int64, requestID string) error
	GetUserConcurrency(ctx context.Context, userID int64) (int, error)

	// 等待队列计数（只在首次创建时设置 TTL）
	IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error)
	DecrementWaitCount(ctx context.Context, userID int64) error

	// 批量负载查询（只读）
	GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error)
	GetUsersLoadBatch(ctx context.Context, users []UserWithConcurrency) (map[int64]*UserLoadInfo, error)

	// 清理过期槽位（后台任务）
	CleanupExpiredAccountSlots(ctx context.Context, accountID int64) error

	// 启动时清理旧进程遗留槽位与等待计数
	CleanupStaleProcessSlots(ctx context.Context, activeRequestPrefix string) error
}

// RouteScheduleCache is an optional extension used by the background
// precompute task. It is separate from ConcurrencyCache so existing test
// doubles and request-path callers remain source-compatible.
type RouteScheduleCache interface {
	SetRouteScheduleConcurrencyLimit(context.Context, string, *int, time.Time) error
	SetRouteScheduleConcurrencyLimits(context.Context, map[string]*int, time.Time) error
	DeleteRouteScheduleConcurrencyLimit(context.Context, string) error
	DeleteRouteScheduleConcurrencyLimits(context.Context, []string) error
	ListRouteScheduleKeys(context.Context) ([]string, error)
}

// RouteScheduleRefreshLock is the optional distributed lock used by the
// background schedule materializer. It is separate from the request-path
// cache contracts so existing implementations remain source-compatible.
type RouteScheduleRefreshLock interface {
	TryAcquireRouteScheduleRefreshLock(context.Context, string, time.Duration) (bool, error)
	RenewRouteScheduleRefreshLock(context.Context, string, time.Duration) (bool, error)
	ReleaseRouteScheduleRefreshLock(context.Context, string) error
}

func (s *ConcurrencyService) SetRouteScheduleConcurrencyLimit(ctx context.Context, key string, limit *int, updatedAt time.Time) error {
	cache, ok := s.cache.(RouteScheduleCache)
	if !ok {
		return fmt.Errorf("route schedule cache is unavailable")
	}
	return cache.SetRouteScheduleConcurrencyLimit(ctx, key, limit, updatedAt)
}

func (s *ConcurrencyService) SetRouteScheduleConcurrencyLimits(ctx context.Context, limits map[string]*int, updatedAt time.Time) error {
	cache, ok := s.cache.(RouteScheduleCache)
	if !ok {
		return fmt.Errorf("route schedule cache is unavailable")
	}
	return cache.SetRouteScheduleConcurrencyLimits(ctx, limits, updatedAt)
}

func (s *ConcurrencyService) DeleteRouteScheduleConcurrencyLimit(ctx context.Context, key string) error {
	cache, ok := s.cache.(RouteScheduleCache)
	if !ok {
		return fmt.Errorf("route schedule cache is unavailable")
	}
	return cache.DeleteRouteScheduleConcurrencyLimit(ctx, key)
}

func (s *ConcurrencyService) DeleteRouteScheduleConcurrencyLimits(ctx context.Context, keys []string) error {
	cache, ok := s.cache.(RouteScheduleCache)
	if !ok {
		return fmt.Errorf("route schedule cache is unavailable")
	}
	return cache.DeleteRouteScheduleConcurrencyLimits(ctx, keys)
}

func (s *ConcurrencyService) ListRouteScheduleKeys(ctx context.Context) ([]string, error) {
	cache, ok := s.cache.(RouteScheduleCache)
	if !ok {
		return nil, fmt.Errorf("route schedule cache is unavailable")
	}
	return cache.ListRouteScheduleKeys(ctx)
}

func (s *ConcurrencyService) TryAcquireRouteScheduleRefreshLock(ctx context.Context, token string, ttl time.Duration) (bool, error) {
	cache, ok := s.cache.(RouteScheduleRefreshLock)
	if !ok {
		return false, fmt.Errorf("route schedule refresh lock is unavailable")
	}
	return cache.TryAcquireRouteScheduleRefreshLock(ctx, token, ttl)
}

func (s *ConcurrencyService) RenewRouteScheduleRefreshLock(ctx context.Context, token string, ttl time.Duration) (bool, error) {
	cache, ok := s.cache.(RouteScheduleRefreshLock)
	if !ok {
		return false, fmt.Errorf("route schedule refresh lock is unavailable")
	}
	return cache.RenewRouteScheduleRefreshLock(ctx, token, ttl)
}

func (s *ConcurrencyService) ReleaseRouteScheduleRefreshLock(ctx context.Context, token string) error {
	cache, ok := s.cache.(RouteScheduleRefreshLock)
	if !ok {
		return fmt.Errorf("route schedule refresh lock is unavailable")
	}
	return cache.ReleaseRouteScheduleRefreshLock(ctx, token)
}

// RouteConcurrencyCache is optional so existing test doubles remain compatible.
type RouteConcurrencyCache interface {
	AcquireRouteSlot(context.Context, string, int, string) (bool, error)
	ReleaseRouteSlot(context.Context, string, string) error
	GetRouteConcurrencyLimit(context.Context, string) (*int, bool, error)
	SetRouteConcurrencyLimit(context.Context, string, *int) error
}

// RouteConcurrencyLimit is the materialized route limit and whether Redis
// contained a value for the route. A nil Limit with Hit=true means unlimited;
// Hit=false means the caller should fall back to the legacy configuration.
type RouteConcurrencyLimit struct {
	Limit *int
	Hit   bool
}

// RouteConcurrencyLimitBatchCache is an optional batch extension used by
// admin/read-only views. It keeps the request path unchanged while avoiding
// one Redis pipeline per candidate when rendering a group.
type RouteConcurrencyLimitBatchCache interface {
	GetRouteConcurrencyLimitBatch(context.Context, []string) (map[string]RouteConcurrencyLimit, error)
}

// RouteLoadCache exposes active counts for route-scoped slots. It is kept
// separate from RouteConcurrencyCache so existing optional implementations
// remain source-compatible.
type RouteLoadCache interface {
	GetRouteConcurrencyBatch(context.Context, []string) (map[string]int, error)
}

type RouteLoadRequest struct {
	Key                   string
	AccountID             int64
	MaxConcurrency        *int
	AccountMaxConcurrency int
}

type RouteLoadInfo struct {
	Key                 string
	AccountID           int64
	CurrentConcurrency  int
	LoadRate            int
	UsedAccountFallback bool
}

// GetRouteConcurrencyLimitsBatch returns materialized route limits. When the
// cache implementation does not support batching, the optional single-route
// interface is used as a compatibility fallback.
func (s *ConcurrencyService) GetRouteConcurrencyLimitsBatch(ctx context.Context, keys []string) (map[string]RouteConcurrencyLimit, error) {
	result := make(map[string]RouteConcurrencyLimit, len(keys))
	if len(keys) == 0 || s == nil || s.cache == nil {
		return result, nil
	}
	if cache, ok := s.cache.(RouteConcurrencyLimitBatchCache); ok {
		return cache.GetRouteConcurrencyLimitBatch(ctx, keys)
	}
	cache, ok := s.cache.(RouteConcurrencyCache)
	if !ok {
		return nil, fmt.Errorf("route concurrency cache is unavailable")
	}
	for _, key := range keys {
		if _, exists := result[key]; exists {
			continue
		}
		limit, hit, err := cache.GetRouteConcurrencyLimit(ctx, key)
		if err != nil {
			return nil, err
		}
		result[key] = RouteConcurrencyLimit{Limit: limit, Hit: hit}
	}
	return result, nil
}

var (
	requestIDPrefix  = initRequestIDPrefix()
	requestIDCounter atomic.Uint64
)

func initRequestIDPrefix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err == nil {
		return "r" + strconv.FormatUint(binary.BigEndian.Uint64(b), 36)
	}
	fallback := uint64(time.Now().UnixNano()) ^ (uint64(os.Getpid()) << 16)
	return "r" + strconv.FormatUint(fallback, 36)
}

func RequestIDPrefix() string {
	return requestIDPrefix
}

func generateRequestID() string {
	seq := requestIDCounter.Add(1)
	return requestIDPrefix + "-" + strconv.FormatUint(seq, 36)
}

func (s *ConcurrencyService) CleanupStaleProcessSlots(ctx context.Context) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.CleanupStaleProcessSlots(ctx, RequestIDPrefix())
}

const (
	// 默认等待队列额外槽位
	defaultExtraWaitSlots = 20

	defaultAccountLoadBatchCacheTTL = 200 * time.Millisecond
	accountLoadBatchFetchTimeout    = 3 * time.Second
	maxAccountLoadBatchCacheEntries = 256
)

// ConcurrencyService 管理账号和用户的并发限制。
type ConcurrencyService struct {
	cache ConcurrencyCache

	accountLoadCacheTTL atomic.Int64
	accountLoadCacheMu  sync.RWMutex
	accountLoadCache    map[string]cachedAccountLoadBatch
	accountLoadGroup    singleflight.Group
}

type cachedAccountLoadBatch struct {
	loadMap   map[int64]*AccountLoadInfo
	expiresAt time.Time
}

// NewConcurrencyService 创建并发控制服务。
func NewConcurrencyService(cache ConcurrencyCache) *ConcurrencyService {
	svc := &ConcurrencyService{
		cache:            cache,
		accountLoadCache: make(map[string]cachedAccountLoadBatch),
	}
	svc.SetAccountLoadBatchCacheTTL(defaultAccountLoadBatchCacheTTL)
	return svc
}

// SetAccountLoadBatchCacheTTL 设置账号负载批量读取的极短 TTL 缓存；非正数表示禁用缓存。
func (s *ConcurrencyService) SetAccountLoadBatchCacheTTL(ttl time.Duration) {
	if s == nil {
		return
	}
	s.accountLoadCacheTTL.Store(int64(ttl))
	if ttl <= 0 {
		s.accountLoadCacheMu.Lock()
		s.accountLoadCache = make(map[string]cachedAccountLoadBatch)
		s.accountLoadCacheMu.Unlock()
	}
}

// AcquireResult represents the result of acquiring a concurrency slot
type AcquireResult struct {
	Acquired    bool
	ReleaseFunc func() // Must be called when done (typically via defer)
}

type AccountWithConcurrency struct {
	ID             int64
	MaxConcurrency int
}

type UserWithConcurrency struct {
	ID             int64
	MaxConcurrency int
}

type AccountLoadInfo struct {
	AccountID          int64
	CurrentConcurrency int
	WaitingCount       int
	LoadRate           int // 0-100+ (percent)
}

type UserLoadInfo struct {
	UserID             int64
	CurrentConcurrency int
	WaitingCount       int
	LoadRate           int // 0-100+ (percent)
}

// AcquireAccountSlot attempts to acquire a concurrency slot for an account.
// If the account is at max concurrency, it waits until a slot is available or timeout.
// Returns a release function that MUST be called when the request completes.
func (s *ConcurrencyService) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int) (*AcquireResult, error) {
	// If maxConcurrency is 0 or negative, no limit
	if maxConcurrency <= 0 {
		return &AcquireResult{
			Acquired:    true,
			ReleaseFunc: func() {}, // no-op
		}, nil
	}

	// Generate unique request ID for this slot
	requestID := generateRequestID()

	acquired, err := s.cache.AcquireAccountSlot(ctx, accountID, maxConcurrency, requestID)
	if err != nil {
		return nil, err
	}

	if acquired {
		return &AcquireResult{
			Acquired: true,
			ReleaseFunc: func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.cache.ReleaseAccountSlot(bgCtx, accountID, requestID); err != nil {
					logger.LegacyPrintf("service.concurrency", "Warning: failed to release account slot for %d (req=%s): %v", accountID, requestID, err)
				}
			},
		}, nil
	}

	return &AcquireResult{
		Acquired:    false,
		ReleaseFunc: nil,
	}, nil
}

func (s *ConcurrencyService) AcquireRouteSlot(ctx context.Context, key string, maxConcurrency int) (*AcquireResult, error) {
	if maxConcurrency <= 0 {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {}}, nil
	}
	c, ok := s.cache.(RouteConcurrencyCache)
	if !ok {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {}}, nil
	}
	requestID := generateRequestID()
	acquired, err := c.AcquireRouteSlot(ctx, key, maxConcurrency, requestID)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return &AcquireResult{Acquired: false}, nil
	}
	return &AcquireResult{Acquired: true, ReleaseFunc: func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.ReleaseRouteSlot(bgCtx, key, requestID)
	}}, nil
}

func (s *ConcurrencyService) GetRouteConcurrencyLimit(ctx context.Context, key string) (*int, bool, error) {
	c, ok := s.cache.(RouteConcurrencyCache)
	if !ok {
		return nil, false, nil
	}
	return c.GetRouteConcurrencyLimit(ctx, key)
}

func (s *ConcurrencyService) SetRouteConcurrencyLimit(ctx context.Context, key string, limit *int) error {
	c, ok := s.cache.(RouteConcurrencyCache)
	if !ok {
		return nil
	}
	return c.SetRouteConcurrencyLimit(ctx, key, limit)
}

// AcquireUserSlot attempts to acquire a concurrency slot for a user.
// If the user is at max concurrency, it waits until a slot is available or timeout.
// Returns a release function that MUST be called when the request completes.
func (s *ConcurrencyService) AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int) (*AcquireResult, error) {
	// If maxConcurrency is 0 or negative, no limit
	if maxConcurrency <= 0 {
		return &AcquireResult{
			Acquired:    true,
			ReleaseFunc: func() {}, // no-op
		}, nil
	}

	// Generate unique request ID for this slot
	requestID := generateRequestID()

	acquired, err := s.cache.AcquireUserSlot(ctx, userID, maxConcurrency, requestID)
	if err != nil {
		return nil, err
	}

	if acquired {
		return &AcquireResult{
			Acquired: true,
			ReleaseFunc: func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.cache.ReleaseUserSlot(bgCtx, userID, requestID); err != nil {
					logger.LegacyPrintf("service.concurrency", "Warning: failed to release user slot for %d (req=%s): %v", userID, requestID, err)
				}
			},
		}, nil
	}

	return &AcquireResult{
		Acquired:    false,
		ReleaseFunc: nil,
	}, nil
}

// ============================================
// Wait Queue Count Methods
// ============================================

// IncrementWaitCount attempts to increment the wait queue counter for a user.
// Returns true if successful, false if the wait queue is full.
// maxWait should be user.Concurrency + defaultExtraWaitSlots
func (s *ConcurrencyService) IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error) {
	if s.cache == nil {
		// Redis not available, allow request
		return true, nil
	}

	result, err := s.cache.IncrementWaitCount(ctx, userID, maxWait)
	if err != nil {
		// On error, allow the request to proceed (fail open)
		logger.LegacyPrintf("service.concurrency", "Warning: increment wait count failed for user %d: %v", userID, err)
		return true, nil
	}
	return result, nil
}

// DecrementWaitCount decrements the wait queue counter for a user.
// Should be called when a request completes or exits the wait queue.
func (s *ConcurrencyService) DecrementWaitCount(ctx context.Context, userID int64) {
	if s.cache == nil {
		return
	}

	// Use background context to ensure decrement even if original context is cancelled
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.cache.DecrementWaitCount(bgCtx, userID); err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: decrement wait count failed for user %d: %v", userID, err)
	}
}

// IncrementAccountWaitCount increments the wait queue counter for an account.
func (s *ConcurrencyService) IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error) {
	if s.cache == nil {
		return true, nil
	}

	result, err := s.cache.IncrementAccountWaitCount(ctx, accountID, maxWait)
	if err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: increment wait count failed for account %d: %v", accountID, err)
		return true, nil
	}
	return result, nil
}

// DecrementAccountWaitCount decrements the wait queue counter for an account.
func (s *ConcurrencyService) DecrementAccountWaitCount(ctx context.Context, accountID int64) {
	if s.cache == nil {
		return
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.cache.DecrementAccountWaitCount(bgCtx, accountID); err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: decrement wait count failed for account %d: %v", accountID, err)
	}
}

// GetAccountWaitingCount gets current wait queue count for an account.
func (s *ConcurrencyService) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	if s.cache == nil {
		return 0, nil
	}
	return s.cache.GetAccountWaitingCount(ctx, accountID)
}

// CalculateMaxWait calculates the maximum wait queue size for a user
// maxWait = userConcurrency + defaultExtraWaitSlots
func CalculateMaxWait(userConcurrency int) int {
	if userConcurrency <= 0 {
		userConcurrency = 1
	}
	return userConcurrency + defaultExtraWaitSlots
}

// GetAccountsLoadBatch 批量获取账号负载信息。
func (s *ConcurrencyService) GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	return s.getAccountsLoadBatch(ctx, accounts, true)
}

// GetRouteLoadsBatch returns route-scoped active concurrency and normalized
// load. Routes without an allocation fall back to account-wide active
// concurrency and effective load factor.
func (s *ConcurrencyService) GetRouteLoadsBatch(ctx context.Context, requests []RouteLoadRequest) (map[string]RouteLoadInfo, error) {
	result := make(map[string]RouteLoadInfo, len(requests))
	if len(requests) == 0 || s == nil || s.cache == nil {
		return result, nil
	}

	routeCache, ok := s.cache.(RouteLoadCache)
	if !ok {
		return nil, fmt.Errorf("route load cache is unavailable")
	}
	routeKeys := make([]string, 0, len(requests))
	accountIDs := make([]int64, 0, len(requests))
	for _, request := range requests {
		if request.MaxConcurrency != nil && *request.MaxConcurrency > 0 {
			routeKeys = append(routeKeys, request.Key)
		} else if request.AccountID > 0 {
			accountIDs = append(accountIDs, request.AccountID)
		}
	}

	routeCounts, err := routeCache.GetRouteConcurrencyBatch(ctx, routeKeys)
	if err != nil {
		return nil, err
	}
	accountCounts, err := s.cache.GetAccountConcurrencyBatch(ctx, uniqueInt64s(accountIDs))
	if err != nil {
		return nil, err
	}

	for _, request := range requests {
		info := RouteLoadInfo{Key: request.Key, AccountID: request.AccountID}
		maxConcurrency := 0
		current := 0
		if request.MaxConcurrency != nil && *request.MaxConcurrency > 0 {
			maxConcurrency = *request.MaxConcurrency
			current = routeCounts[request.Key]
		} else {
			maxConcurrency = request.AccountMaxConcurrency
			current = accountCounts[request.AccountID]
			info.UsedAccountFallback = true
		}
		info.CurrentConcurrency = current
		if maxConcurrency > 0 {
			info.LoadRate = current * 100 / maxConcurrency
		}
		result[request.Key] = info
	}
	return result, nil
}

func uniqueInt64s(values []int64) []int64 {
	if len(values) < 2 {
		return values
	}
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// GetAccountsLoadBatchFresh 绕过极短 TTL 缓存，用于抢槽失败后的实时刷新兜底。
func (s *ConcurrencyService) GetAccountsLoadBatchFresh(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	return s.getAccountsLoadBatch(ctx, accounts, false)
}

func (s *ConcurrencyService) getAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency, allowCache bool) (map[int64]*AccountLoadInfo, error) {
	if len(accounts) == 0 {
		return map[int64]*AccountLoadInfo{}, nil
	}
	if s.cache == nil {
		return map[int64]*AccountLoadInfo{}, nil
	}

	ttl := time.Duration(s.accountLoadCacheTTL.Load())
	if !allowCache || ttl <= 0 {
		return s.fetchAccountsLoadBatch(ctx, accounts)
	}

	key := accountLoadBatchCacheKey(accounts)
	if cached, ok := s.getCachedAccountLoadBatch(key, time.Now()); ok {
		return cached, nil
	}

	value, err, _ := s.accountLoadGroup.Do(key, func() (any, error) {
		now := time.Now()
		if cached, ok := s.getCachedAccountLoadBatch(key, now); ok {
			return cached, nil
		}
		loadMap, fetchErr := s.fetchAccountsLoadBatch(ctx, accounts)
		if fetchErr != nil {
			return nil, fetchErr
		}
		cached := cloneAccountLoadMap(loadMap)
		s.storeCachedAccountLoadBatch(key, cached, now.Add(ttl))
		return cached, nil
	})
	if err != nil {
		return nil, err
	}
	loadMap, _ := value.(map[int64]*AccountLoadInfo)
	if loadMap == nil {
		return map[int64]*AccountLoadInfo{}, nil
	}
	return loadMap, nil
}

func (s *ConcurrencyService) fetchAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	if s.cache == nil {
		return map[int64]*AccountLoadInfo{}, nil
	}
	baseCtx := context.Background()
	if ctx != nil {
		baseCtx = context.WithoutCancel(ctx)
	}
	redisCtx, cancel := context.WithTimeout(baseCtx, accountLoadBatchFetchTimeout)
	defer cancel()
	return s.cache.GetAccountsLoadBatch(redisCtx, accounts)
}

func (s *ConcurrencyService) getCachedAccountLoadBatch(key string, now time.Time) (map[int64]*AccountLoadInfo, bool) {
	s.accountLoadCacheMu.RLock()
	cached, ok := s.accountLoadCache[key]
	s.accountLoadCacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if !now.Before(cached.expiresAt) {
		s.accountLoadCacheMu.Lock()
		if current, exists := s.accountLoadCache[key]; exists && !now.Before(current.expiresAt) {
			delete(s.accountLoadCache, key)
		}
		s.accountLoadCacheMu.Unlock()
		return nil, false
	}
	return cached.loadMap, true
}

func (s *ConcurrencyService) storeCachedAccountLoadBatch(key string, loadMap map[int64]*AccountLoadInfo, expiresAt time.Time) {
	s.accountLoadCacheMu.Lock()
	if s.accountLoadCache == nil {
		s.accountLoadCache = make(map[string]cachedAccountLoadBatch)
	}
	if len(s.accountLoadCache) >= maxAccountLoadBatchCacheEntries {
		now := time.Now()
		for cacheKey, cached := range s.accountLoadCache {
			if !now.Before(cached.expiresAt) {
				delete(s.accountLoadCache, cacheKey)
			}
		}
		for len(s.accountLoadCache) >= maxAccountLoadBatchCacheEntries {
			for cacheKey := range s.accountLoadCache {
				delete(s.accountLoadCache, cacheKey)
				break
			}
		}
	}
	s.accountLoadCache[key] = cachedAccountLoadBatch{
		loadMap:   loadMap,
		expiresAt: expiresAt,
	}
	s.accountLoadCacheMu.Unlock()
}

func accountLoadBatchCacheKey(accounts []AccountWithConcurrency) string {
	hash := sha256.New()
	var buf [16]byte
	for _, account := range accounts {
		binary.LittleEndian.PutUint64(buf[:8], uint64(account.ID))
		binary.LittleEndian.PutUint64(buf[8:], uint64(int64(account.MaxConcurrency)))
		_, _ = hash.Write(buf[:])
	}
	sum := hash.Sum(nil)
	return strconv.Itoa(len(accounts)) + ":" + hex.EncodeToString(sum)
}

func cloneAccountLoadMap(loadMap map[int64]*AccountLoadInfo) map[int64]*AccountLoadInfo {
	if len(loadMap) == 0 {
		return map[int64]*AccountLoadInfo{}
	}
	clone := make(map[int64]*AccountLoadInfo, len(loadMap))
	for accountID, loadInfo := range loadMap {
		if loadInfo == nil {
			clone[accountID] = nil
			continue
		}
		copied := *loadInfo
		clone[accountID] = &copied
	}
	return clone
}

// GetUsersLoadBatch returns load info for multiple users.
func (s *ConcurrencyService) GetUsersLoadBatch(ctx context.Context, users []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	if s.cache == nil {
		return map[int64]*UserLoadInfo{}, nil
	}
	return s.cache.GetUsersLoadBatch(ctx, users)
}

// CleanupExpiredAccountSlots removes expired slots for one account (background task).
func (s *ConcurrencyService) CleanupExpiredAccountSlots(ctx context.Context, accountID int64) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.CleanupExpiredAccountSlots(ctx, accountID)
}

// StartSlotCleanupWorker starts a background cleanup worker for expired account slots.
func (s *ConcurrencyService) StartSlotCleanupWorker(accountRepo AccountRepository, interval time.Duration) {
	if s == nil || s.cache == nil || accountRepo == nil || interval <= 0 {
		return
	}

	runCleanup := func() {
		listCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		accounts, err := accountRepo.ListSchedulable(listCtx)
		cancel()
		if err != nil {
			logger.LegacyPrintf("service.concurrency", "Warning: list schedulable accounts failed: %v", err)
			return
		}
		for _, account := range accounts {
			accountCtx, accountCancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := s.cache.CleanupExpiredAccountSlots(accountCtx, account.ID)
			accountCancel()
			if err != nil {
				logger.LegacyPrintf("service.concurrency", "Warning: cleanup expired slots failed for account %d: %v", account.ID, err)
			}
		}
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		runCleanup()
		for range ticker.C {
			runCleanup()
		}
	}()
}

// GetAccountConcurrencyBatch gets current concurrency counts for multiple accounts.
// Uses a detached context with timeout to prevent HTTP request cancellation from
// causing the entire batch to fail (which would show all concurrency as 0).
func (s *ConcurrencyService) GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error) {
	if len(accountIDs) == 0 {
		return map[int64]int{}, nil
	}
	if s.cache == nil {
		result := make(map[int64]int, len(accountIDs))
		for _, accountID := range accountIDs {
			result[accountID] = 0
		}
		return result, nil
	}

	// Use a detached context so that a cancelled HTTP request doesn't cause
	// the Redis pipeline to fail and return all-zero concurrency counts.
	redisCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return s.cache.GetAccountConcurrencyBatch(redisCtx, accountIDs)
}
