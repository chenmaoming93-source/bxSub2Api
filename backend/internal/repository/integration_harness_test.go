//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	redisImageTag = "redis:8.4-alpine"
	mysqlImageTag = "mysql:8.4"
)

var (
	integrationDB        *sql.DB
	integrationEntClient *dbent.Client
	integrationRedis     *redisclient.Client

	redisNamespaceSeq uint64
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	if err := timezone.Init("UTC"); err != nil {
		log.Printf("failed to init timezone: %v", err)
		os.Exit(1)
	}

	if os.Getenv("OBSERVABILITY_PERF_SHADOW_TABLES") == "1" {
		os.Exit(runShadowTablePerformanceEnvironment(m))
	}

	if os.Getenv("OBSERVABILITY_PERF_EXTERNAL") == "1" {
		os.Exit(runExternalPerformanceEnvironment(m))
	}

	if !dockerIsAvailable(ctx) {
		// In CI we expect Docker to be available so integration tests should fail loudly.
		if os.Getenv("CI") != "" {
			log.Printf("docker is not available (CI=true); failing integration tests")
			os.Exit(1)
		}
		log.Printf("docker is not available; skipping integration tests (start Docker to enable)")
		os.Exit(0)
	}

	mysqlImage := selectDockerImage(ctx, mysqlImageTag)
	mysqlContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        mysqlImage,
			ExposedPorts: []string{"3306/tcp"},
			Env: map[string]string{
				"MYSQL_DATABASE":      "sub2api_test",
				"MYSQL_USER":          "sub2api",
				"MYSQL_PASSWORD":      "sub2api",
				"MYSQL_ROOT_PASSWORD": "root",
			},
			WaitingFor: wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		log.Printf("failed to start mysql container: %v", err)
		os.Exit(1)
	}
	defer func() { _ = mysqlContainer.Terminate(ctx) }()

	redisContainer, err := tcredis.Run(
		ctx,
		redisImageTag,
	)
	if err != nil {
		log.Printf("failed to start redis container: %v", err)
		os.Exit(1)
	}
	defer func() { _ = redisContainer.Terminate(ctx) }()

	mysqlHost, err := mysqlContainer.Host(ctx)
	if err != nil {
		log.Printf("failed to get mysql host: %v", err)
		os.Exit(1)
	}
	mysqlPort, err := mysqlContainer.MappedPort(ctx, "3306/tcp")
	if err != nil {
		log.Printf("failed to get mysql port: %v", err)
		os.Exit(1)
	}
	dsn := fmt.Sprintf("sub2api:sub2api@tcp(%s:%d)/sub2api_test?charset=utf8mb4&parseTime=true&loc=UTC&multiStatements=true", mysqlHost, mysqlPort.Int())

	integrationDB, err = openSQLWithRetry(ctx, dsn, 30*time.Second)
	if err != nil {
		log.Printf("failed to open sql db: %v", err)
		os.Exit(1)
	}
	if err := ApplyMigrations(ctx, integrationDB); err != nil {
		log.Printf("failed to apply db migrations: %v", err)
		os.Exit(1)
	}

	// 创建 ent client 用于集成测试
	drv := entsql.OpenDB(dialect.MySQL, integrationDB)
	integrationEntClient = dbent.NewClient(dbent.Driver(drv))

	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		log.Printf("failed to get redis host: %v", err)
		os.Exit(1)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		log.Printf("failed to get redis port: %v", err)
		os.Exit(1)
	}

	integrationRedis = redisclient.NewClient(&redisclient.Options{
		Addr: fmt.Sprintf("%s:%d", redisHost, redisPort.Int()),
		DB:   0,
	})
	if err := integrationRedis.Ping(ctx).Err(); err != nil {
		log.Printf("failed to ping redis: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	_ = integrationEntClient.Close()
	_ = integrationRedis.Close()
	_ = integrationDB.Close()

	os.Exit(code)
}

// runShadowTablePerformanceEnvironment gives the performance gate a
// connection-local set of temporary tables over the configured database. It is
// used when the MySQL account can write its application schema but cannot
// create a separate database. All fixture writes stay on one connection and
// disappear when m.Run returns.
func runShadowTablePerformanceEnvironment(m *testing.M) int {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Printf("load performance config: %v", err)
		return 1
	}
	integrationDB, err = openSQLWithRetry(ctx, cfg.Database.DSN(), 30*time.Second)
	if err != nil {
		log.Printf("connect performance database: %v", err)
		return 1
	}
	integrationDB.SetMaxOpenConns(1)
	integrationDB.SetMaxIdleConns(1)
	integrationDB.SetConnMaxIdleTime(0)
	integrationDB.SetConnMaxLifetime(0)

	tables := []string{
		"users",
		"groups",
		"accounts",
		"proxies",
		"api_keys",
		"account_groups",
		"user_allowed_groups",
		"group_model_route_accounts",
		"observability_request_bucket",
		"observability_concurrency_bucket",
		"ops_error_logs",
	}
	for _, table := range tables {
		quoted := "`" + table + "`"
		source := "`" + cfg.Database.DBName + "`." + quoted
		if _, err := integrationDB.ExecContext(ctx, "CREATE TEMPORARY TABLE "+quoted+" AS SELECT * FROM "+source+" WHERE 1=0"); err != nil {
			log.Printf("create performance shadow table %s: %v", table, err)
			_ = integrationDB.Close()
			return 1
		}
		if err := copyPerformanceIndexes(ctx, integrationDB, cfg.Database.DBName, table); err != nil {
			log.Printf("copy performance indexes for %s: %v", table, err)
			_ = integrationDB.Close()
			return 1
		}
	}

	drv := entsql.OpenDB(dialect.MySQL, integrationDB)
	integrationEntClient = dbent.NewClient(dbent.Driver(drv))
	integrationRedis = redisclient.NewClient(&redisclient.Options{
		Addr:     cfg.Redis.Address(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := integrationRedis.Ping(ctx).Err(); err != nil {
		log.Printf("ping performance Redis: %v", err)
		_ = integrationEntClient.Close()
		_ = integrationRedis.Close()
		_ = integrationDB.Close()
		return 1
	}

	code := m.Run()
	_ = integrationEntClient.Close()
	_ = integrationRedis.Close()
	_ = integrationDB.Close()
	return code
}

func copyPerformanceIndexes(ctx context.Context, db *sql.DB, sourceDatabase, table string) error {
	rows, err := db.QueryContext(ctx, "SHOW INDEX FROM `"+sourceDatabase+"`.`"+table+"`")
	if err != nil {
		return err
	}
	type indexDef struct {
		name    string
		unique  bool
		columns []string
	}
	byName := make(map[string]*indexDef)
	order := make([]string, 0)
	for rows.Next() {
		var tableName, nonUnique, keyName, collation, indexType, comment, indexComment string
		var seq, cardinality sql.NullInt64
		var columnName, subPart, packed sql.NullString
		var nullable string
		var visible sql.NullString
		var expression sql.NullString
		if err := rows.Scan(
			&tableName, &nonUnique, &keyName, &seq, &columnName, &collation,
			&cardinality, &subPart, &packed, &nullable, &indexType, &comment, &indexComment,
			&visible, &expression,
		); err != nil {
			_ = rows.Close()
			return err
		}
		item := byName[keyName]
		if item == nil {
			item = &indexDef{name: keyName, unique: nonUnique == "0"}
			byName[keyName] = item
			order = append(order, keyName)
		}
		if columnName.Valid && columnName.String != "" {
			item.columns = append(item.columns, "`"+columnName.String+"`")
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, keyName := range order {
		item := byName[keyName]
		if len(item.columns) == 0 {
			continue
		}
		statement := "CREATE INDEX `" + item.name + "` ON `" + table + "` (" + strings.Join(item.columns, ",") + ")"
		if item.name == "PRIMARY" {
			statement = "ALTER TABLE `" + table + "` ADD PRIMARY KEY (" + strings.Join(item.columns, ",") + ")"
		} else if item.unique {
			statement = "CREATE UNIQUE INDEX `" + item.name + "` ON `" + table + "` (" + strings.Join(item.columns, ",") + ")"
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("create index %s: %w", item.name, err)
		}
	}
	return nil
}

// runExternalPerformanceEnvironment uses the configured MySQL/Redis servers
// but creates a dedicated database and Redis namespace for the duration of the
// test process. It exists so performance release gates can run on hosts where
// Docker/Testcontainers is unavailable.
func runExternalPerformanceEnvironment(m *testing.M) int {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Printf("load performance config: %v", err)
		return 1
	}

	dbName := fmt.Sprintf("sub2api_obs_perf_%d", time.Now().UnixNano())
	adminConfig := cfg.Database
	adminConfig.DBName = ""
	adminDB, err := openSQLWithRetry(ctx, adminConfig.DSN(), 30*time.Second)
	if err != nil {
		log.Printf("connect performance MySQL admin database: %v", err)
		return 1
	}
	defer func() { _ = adminDB.Close() }()

	if _, err := adminDB.ExecContext(ctx,
		"CREATE DATABASE `"+dbName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
	); err != nil {
		log.Printf("create performance database: %v", err)
		return 1
	}
	dropDatabase := func() {
		if _, err := adminDB.ExecContext(context.Background(), "DROP DATABASE IF EXISTS `"+dbName+"`"); err != nil {
			log.Printf("drop performance database: %v", err)
		}
	}

	testConfig := cfg.Database
	testConfig.DBName = dbName
	integrationDB, err = openSQLWithRetry(ctx, testConfig.DSN(), 30*time.Second)
	if err != nil {
		log.Printf("connect performance database: %v", err)
		dropDatabase()
		return 1
	}
	if err := ApplyMigrations(ctx, integrationDB); err != nil {
		log.Printf("migrate performance database: %v", err)
		_ = integrationDB.Close()
		dropDatabase()
		return 1
	}
	drv := entsql.OpenDB(dialect.MySQL, integrationDB)
	integrationEntClient = dbent.NewClient(dbent.Driver(drv))

	integrationRedis = redisclient.NewClient(&redisclient.Options{
		Addr:     cfg.Redis.Address(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := integrationRedis.Ping(ctx).Err(); err != nil {
		log.Printf("ping performance Redis: %v", err)
		_ = integrationEntClient.Close()
		_ = integrationRedis.Close()
		_ = integrationDB.Close()
		dropDatabase()
		return 1
	}

	code := m.Run()

	_ = integrationEntClient.Close()
	_ = integrationRedis.Close()
	_ = integrationDB.Close()
	dropDatabase()
	return code
}

func dockerIsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Env = os.Environ()
	return cmd.Run() == nil
}

func selectDockerImage(ctx context.Context, preferred string) string {
	if dockerImageExists(ctx, preferred) {
		return preferred
	}

	return preferred
}

func dockerImageExists(ctx context.Context, image string) bool {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	cmd.Env = os.Environ()
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func openSQLWithRetry(ctx context.Context, dsn string, timeout time.Duration) (*sql.DB, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}

		if err := pingWithTimeout(ctx, db, 2*time.Second); err != nil {
			lastErr = err
			_ = db.Close()
			time.Sleep(250 * time.Millisecond)
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("db not ready after %s: %w", timeout, lastErr)
}

func pingWithTimeout(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return db.PingContext(pingCtx)
}

func testTx(t *testing.T) *sql.Tx {
	t.Helper()

	tx, err := integrationDB.BeginTx(context.Background(), nil)
	require.NoError(t, err, "begin tx")
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	return tx
}

// testEntClient 返回全局的 ent client，用于测试需要内部管理事务的代码（如 Create/Update 方法）。
// 注意：此 client 的操作会真正写入数据库，测试结束后不会自动回滚。
func testEntClient(t *testing.T) *dbent.Client {
	t.Helper()
	return integrationEntClient
}

// testEntTx 返回一个 ent 事务，用于需要事务隔离的测试。
// 测试结束后会自动回滚，不会影响数据库状态。
func testEntTx(t *testing.T) *dbent.Tx {
	t.Helper()

	tx, err := integrationEntClient.Tx(context.Background())
	require.NoError(t, err, "begin ent tx")
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	return tx
}

// testEntSQLTx 已弃用：不要在新测试中使用此函数。
// 基于 *sql.Tx 创建的 ent client 在调用 client.Tx() 时会 panic。
// 对于需要测试内部使用事务的代码，请使用 testEntClient。
// 对于需要事务隔离的测试，请使用 testEntTx。
//
// Deprecated: Use testEntClient or testEntTx instead.
func testEntSQLTx(t *testing.T) (*dbent.Client, *sql.Tx) {
	t.Helper()

	// 直接失败，避免旧测试误用导致的事务嵌套 panic。
	t.Fatalf("testEntSQLTx 已弃用：请使用 testEntClient 或 testEntTx")
	return nil, nil
}

func testRedis(t *testing.T) *redisclient.Client {
	t.Helper()

	prefix := fmt.Sprintf(
		"it:%s:%d:%d:",
		sanitizeRedisNamespace(t.Name()),
		time.Now().UnixNano(),
		atomic.AddUint64(&redisNamespaceSeq, 1),
	)

	opts := *integrationRedis.Options()
	rdb := redisclient.NewClient(&opts)
	rdb.AddHook(prefixHook{prefix: prefix})

	t.Cleanup(func() {
		ctx := context.Background()

		var cursor uint64
		for {
			keys, nextCursor, err := integrationRedis.Scan(ctx, cursor, prefix+"*", 500).Result()
			require.NoError(t, err, "scan redis keys for cleanup")
			if len(keys) > 0 {
				require.NoError(t, integrationRedis.Unlink(ctx, keys...).Err(), "unlink redis keys for cleanup")
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}

		_ = rdb.Close()
	})

	return rdb
}

func assertTTLWithin(t *testing.T, ttl time.Duration, min, max time.Duration) {
	t.Helper()
	require.GreaterOrEqual(t, ttl, min, "ttl should be >= min")
	require.LessOrEqual(t, ttl, max, "ttl should be <= max")
}

func sanitizeRedisNamespace(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

type prefixHook struct {
	prefix string
}

func (h prefixHook) DialHook(next redisclient.DialHook) redisclient.DialHook { return next }

func (h prefixHook) ProcessHook(next redisclient.ProcessHook) redisclient.ProcessHook {
	return func(ctx context.Context, cmd redisclient.Cmder) error {
		h.prefixCmd(cmd)
		return next(ctx, cmd)
	}
}

func (h prefixHook) ProcessPipelineHook(next redisclient.ProcessPipelineHook) redisclient.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redisclient.Cmder) error {
		for _, cmd := range cmds {
			h.prefixCmd(cmd)
		}
		return next(ctx, cmds)
	}
}

func (h prefixHook) prefixCmd(cmd redisclient.Cmder) {
	args := cmd.Args()
	if len(args) < 2 {
		return
	}

	prefixOne := func(i int) {
		if i < 0 || i >= len(args) {
			return
		}

		switch v := args[i].(type) {
		case string:
			if v != "" && !strings.HasPrefix(v, h.prefix) {
				args[i] = h.prefix + v
			}
		case []byte:
			s := string(v)
			if s != "" && !strings.HasPrefix(s, h.prefix) {
				args[i] = []byte(h.prefix + s)
			}
		}
	}

	switch strings.ToLower(cmd.Name()) {
	case "get", "set", "setnx", "setex", "psetex", "incr", "decr", "incrby", "expire", "pexpire", "ttl", "pttl",
		"hgetall", "hget", "hset", "hdel", "hincrbyfloat", "exists",
		"zadd", "zcard", "zrange", "zrangebyscore", "zrem", "zremrangebyscore", "zrevrange", "zrevrangebyscore", "zscore":
		prefixOne(1)
	case "mget":
		for i := 1; i < len(args); i++ {
			prefixOne(i)
		}
	case "del", "unlink":
		for i := 1; i < len(args); i++ {
			prefixOne(i)
		}
	case "eval", "evalsha", "eval_ro", "evalsha_ro":
		if len(args) < 3 {
			return
		}
		numKeys, err := strconv.Atoi(fmt.Sprint(args[2]))
		if err != nil || numKeys <= 0 {
			return
		}
		for i := 0; i < numKeys && 3+i < len(args); i++ {
			prefixOne(3 + i)
		}
	case "scan":
		for i := 2; i+1 < len(args); i++ {
			if strings.EqualFold(fmt.Sprint(args[i]), "match") {
				prefixOne(i + 1)
				break
			}
		}
	}
}

// IntegrationRedisSuite provides a base suite for Redis integration tests.
// Embedding suites should call SetupTest to initialize ctx and rdb.
type IntegrationRedisSuite struct {
	suite.Suite
	ctx context.Context
	rdb *redisclient.Client
}

// SetupTest initializes ctx and rdb for each test method.
func (s *IntegrationRedisSuite) SetupTest() {
	s.ctx = context.Background()
	s.rdb = testRedis(s.T())
}

// RequireNoError is a convenience method wrapping require.NoError with s.T().
func (s *IntegrationRedisSuite) RequireNoError(err error, msgAndArgs ...any) {
	s.T().Helper()
	require.NoError(s.T(), err, msgAndArgs...)
}

// AssertTTLWithin asserts that ttl is within [min, max].
func (s *IntegrationRedisSuite) AssertTTLWithin(ttl, min, max time.Duration) {
	s.T().Helper()
	assertTTLWithin(s.T(), ttl, min, max)
}

// IntegrationDBSuite provides a base suite for DB integration tests.
// Embedding suites should call SetupTest to initialize ctx and client.
type IntegrationDBSuite struct {
	suite.Suite
	ctx    context.Context
	client *dbent.Client
	tx     *dbent.Tx
}

// SetupTest initializes ctx and client for each test method.
func (s *IntegrationDBSuite) SetupTest() {
	s.ctx = context.Background()
	// 统一使用 ent.Tx，确保每个测试都有独立事务并自动回滚。
	tx := testEntTx(s.T())
	s.tx = tx
	s.client = tx.Client()
}

// RequireNoError is a convenience method wrapping require.NoError with s.T().
func (s *IntegrationDBSuite) RequireNoError(err error, msgAndArgs ...any) {
	s.T().Helper()
	require.NoError(s.T(), err, msgAndArgs...)
}
