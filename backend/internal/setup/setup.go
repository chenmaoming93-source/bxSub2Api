package setup

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

// Config paths
const (
	ConfigFileName             = "config.yaml"
	defaultUserConcurrency     = 5
	simpleModeAdminConcurrency = 30
)

func setupDefaultAdminConcurrency() int {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("RUN_MODE")), config.RunModeSimple) {
		return simpleModeAdminConcurrency
	}
	return defaultUserConcurrency
}

// GetDataDir returns the data directory for storing config and lock files.
// Default is /opt/iba/sub2api; can be overridden by DATA_DIR env for testing.
func GetDataDir() string {
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		return dir
	}
	return "/opt/iba/sub2api/resource"
}

// GetConfigFilePath returns the full path to config.yaml
func GetConfigFilePath() string {
	return GetDataDir() + "/" + ConfigFileName
}

// GetInstallLockPath returns the full path to .installed lock file
func GetInstallLockPath() string {
	return "/app/.installed"
}

// SetupConfig holds the setup configuration
type SetupConfig struct {
	Database DatabaseConfig `json:"database" yaml:"database"`
	Redis    RedisConfig    `json:"redis" yaml:"redis"`
	Admin    AdminConfig    `json:"admin" yaml:"-"` // Not stored in config file
	Server   ServerConfig   `json:"server" yaml:"server"`
	JWT      JWTConfig      `json:"jwt" yaml:"jwt"`
	Timezone string         `json:"timezone" yaml:"timezone"` // e.g. "Asia/Shanghai", "UTC"
}

type DatabaseConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	DBName   string `json:"dbname" yaml:"dbname"`
	SSLMode  string `json:"sslmode" yaml:"sslmode"`
}

type RedisConfig struct {
	Host      string `json:"host" yaml:"host"`
	Port      int    `json:"port" yaml:"port"`
	Password  string `json:"password" yaml:"password"`
	DB        int    `json:"db" yaml:"db"`
	EnableTLS bool   `json:"enable_tls" yaml:"enable_tls"`
}

type AdminConfig struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ServerConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
	Mode string `json:"mode" yaml:"mode"`
}

type JWTConfig struct {
	Secret     string `json:"secret" yaml:"secret"`
	ExpireHour int    `json:"expire_hour" yaml:"expire_hour"`
}

const (
	adminBootstrapReasonAdminMissing = "admin_missing"
	adminBootstrapReasonAdminExists  = "admin_exists"
)

type adminBootstrapDecision struct {
	shouldCreate bool
	reason       string
}

func decideAdminBootstrap(adminUsers int64) adminBootstrapDecision {
	if adminUsers > 0 {
		return adminBootstrapDecision{
			shouldCreate: false,
			reason:       adminBootstrapReasonAdminExists,
		}
	}
	return adminBootstrapDecision{
		shouldCreate: true,
		reason:       adminBootstrapReasonAdminMissing,
	}
}

// NeedsSetup checks whether the authoritative installation marker is missing.
func NeedsSetup() bool {
	// The installation lock is the commit marker for a completed installation.
	// A config file alone is not sufficient: it may be shipped by the operator
	// before migrations and the initial administrator have been created.
	return needsSetupAt(GetInstallLockPath())
}

func needsSetupAt(lockPath string) bool {
	_, err := os.Stat(lockPath)
	return os.IsNotExist(err)
}

func buildGoldenDBDSN(cfg *DatabaseConfig, dbName string) string {
	databasePart := "/"
	if dbName != "" {
		databasePart += dbName
	}
	tlsMode := "false"
	switch strings.ToLower(strings.TrimSpace(cfg.SSLMode)) {
	case "prefer", "require", "true", "1", "skip-verify":
		tlsMode = "skip-verify"
	case "verify-ca", "verify-full":
		tlsMode = "true"
	case "", "disable", "false", "0":
		tlsMode = "false"
	default:
		tlsMode = cfg.SSLMode
	}
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)%s?charset=utf8mb4&parseTime=true&loc=UTC&tls=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, databasePart, tlsMode,
	)
}

func buildDatabaseConnectionDSNs(cfg *DatabaseConfig) (bootstrapDSN, targetDSN string) {
	return buildGoldenDBDSN(cfg, ""), buildGoldenDBDSN(cfg, cfg.DBName)
}

// TestDatabaseConnection tests the database connection and creates database if not exists
func TestDatabaseConnection(cfg *DatabaseConfig) error {
	// First, connect without selecting a database so we can check/create the
	// target schema before opening cfg.DBName.
	defaultDSN, targetDSN := buildDatabaseConnectionDSNs(cfg)

	db, err := sql.Open("mysql", defaultDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to GoldenDB: %w", err)
	}

	defer func() {
		if db == nil {
			return
		}
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close GoldenDB connection: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Check if target database exists
	var exists bool
	row := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?)", cfg.DBName)
	if err := row.Scan(&exists); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	// Create database if not exists
	if !exists {
		// 注意：数据库名不能参数化，依赖前置输入校验保障安全。
		// Note: Database names cannot be parameterized, but we've already validated cfg.DBName
		// in the handler using validateDBName() which only allows [a-zA-Z][a-zA-Z0-9_]*
		_, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.DBName))
		if err != nil {
			return fmt.Errorf("failed to create database '%s': %w", cfg.DBName, err)
		}
		logger.LegacyPrintf("setup", "Database '%s' created successfully", cfg.DBName)
	}

	// Now connect to the target database to verify
	if err := db.Close(); err != nil {
		logger.LegacyPrintf("setup", "failed to close GoldenDB connection: %v", err)
	}
	db = nil

	targetDB, err := sql.Open("mysql", targetDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", cfg.DBName, err)
	}

	defer func() {
		if err := targetDB.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close GoldenDB connection: %v", err)
		}
	}()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	if err := targetDB.PingContext(ctx2); err != nil {
		return fmt.Errorf("ping target database failed: %w", err)
	}

	return nil
}

// TestRedisConnection tests the Redis connection
func TestRedisConnection(cfg *RedisConfig) error {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Host,
		}
	}

	rdb := redis.NewClient(opts)
	defer func() {
		if err := rdb.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close redis client: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// Install performs the installation with the given configuration
func Install(cfg *SetupConfig) error {
	// Security check: prevent re-installation if already installed
	if !NeedsSetup() {
		return fmt.Errorf("system is already installed, re-installation is not allowed")
	}

	// Generate JWT secret if not provided
	if cfg.JWT.Secret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return fmt.Errorf("failed to generate jwt secret: %w", err)
		}
		cfg.JWT.Secret = secret
		logger.LegacyPrintf("setup", "%s", "Warning: JWT secret auto-generated. Consider setting a fixed secret for production.")
	}
	if err := applyAdminBootstrapCredentials(cfg); err != nil {
		return fmt.Errorf("load admin bootstrap configuration: %w", err)
	}

	// Test connections
	if err := TestDatabaseConnection(&cfg.Database); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	if err := TestRedisConnection(&cfg.Redis); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	// Initialize database
	if err := initializeDatabase(cfg); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}

	// Create admin user (only when database is empty and no admin exists).
	if _, _, err := createAdminUser(cfg); err != nil {
		return fmt.Errorf("admin user creation failed: %w", err)
	}

	// Preserve an operator-provided config file. It may contain many settings
	// that are intentionally outside the setup form.
	_, configExists, err := existingBootstrapConfigFile()
	if err != nil {
		return fmt.Errorf("check config file: %w", err)
	}
	if !configExists {
		if err := writeConfigFile(cfg); err != nil {
			return fmt.Errorf("config file creation failed: %w", err)
		}
	}

	// Create installation lock file to prevent re-setup attacks
	if err := createInstallLock(); err != nil {
		return fmt.Errorf("failed to create install lock: %w", err)
	}

	return nil
}

// createInstallLock creates a lock file to prevent re-installation attacks
func createInstallLock() error {
	content := fmt.Sprintf("installed_at=%s\n", time.Now().UTC().Format(time.RFC3339))
	return os.WriteFile(GetInstallLockPath(), []byte(content), 0400) // Read-only for owner
}

func applyAdminBootstrapCredentials(cfg *SetupConfig) error {
	if cfg == nil {
		return fmt.Errorf("nil setup config")
	}

	// Configuration file values have the highest priority during bootstrap.
	var fileConfig struct {
		Default struct {
			AdminEmail    string `yaml:"admin_email"`
			AdminPassword string `yaml:"admin_password"`
		} `yaml:"default"`
	}
	configPath, exists, err := existingBootstrapConfigFile()
	if err != nil {
		return err
	}
	if exists {
		content, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", configPath, err)
		}
		if err := yaml.Unmarshal(content, &fileConfig); err != nil {
			return fmt.Errorf("parse %s: %w", configPath, err)
		}
		if value := strings.TrimSpace(fileConfig.Default.AdminEmail); value != "" {
			cfg.Admin.Email = value
		}
		if value := strings.TrimSpace(fileConfig.Default.AdminPassword); value != "" {
			cfg.Admin.Password = value
		}
	}

	if strings.TrimSpace(cfg.Admin.Email) == "" {
		cfg.Admin.Email = "admin@example.com"
	}
	if strings.TrimSpace(cfg.Admin.Password) == "" {
		cfg.Admin.Password = "admin123"
	}
	return nil
}

func existingBootstrapConfigFile() (string, bool, error) {
	candidates := []string{
		GetConfigFilePath(),
		"config.yaml",
		"config/config.yaml",
		"/etc/sub2api/config.yaml",
	}
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true, nil
		} else if !os.IsNotExist(err) {
			return "", false, fmt.Errorf("stat %s: %w", candidate, err)
		}
	}
	return "", false, nil
}

// HasBootstrapConfig reports whether startup can perform a non-interactive
// installation from an existing config.yaml.
func HasBootstrapConfig() (bool, error) {
	_, exists, err := existingBootstrapConfigFile()
	return exists, err
}

// AutoSetupFromConfig initializes a first-run installation from config.yaml.
// This is the local/source-run counterpart of AutoSetupFromEnv.
func AutoSetupFromConfig() error {
	appCfg, err := config.LoadForBootstrap()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg := setupConfigFromAppConfig(appCfg)
	logger.LegacyPrintf("setup", "Auto setup enabled by existing config: %s", GetConfigFilePath())
	return Install(cfg)
}

func setupConfigFromAppConfig(appCfg *config.Config) *SetupConfig {
	if appCfg == nil {
		return &SetupConfig{}
	}
	return &SetupConfig{
		Database: DatabaseConfig{
			Host: appCfg.Database.Host, Port: appCfg.Database.Port,
			User: appCfg.Database.User, Password: appCfg.Database.Password,
			DBName: appCfg.Database.DBName, SSLMode: appCfg.Database.SSLMode,
		},
		Redis: RedisConfig{
			Host: appCfg.Redis.Host, Port: appCfg.Redis.Port,
			Password: appCfg.Redis.Password, DB: appCfg.Redis.DB,
			EnableTLS: appCfg.Redis.EnableTLS,
		},
		Admin: AdminConfig{
			Email: appCfg.Default.AdminEmail, Password: appCfg.Default.AdminPassword,
		},
		Server: ServerConfig{
			Host: appCfg.Server.Host, Port: appCfg.Server.Port, Mode: appCfg.Server.Mode,
		},
		JWT: JWTConfig{
			Secret: appCfg.JWT.Secret, ExpireHour: appCfg.JWT.ExpireHour,
		},
		Timezone: appCfg.Timezone,
	}
}

func initializeDatabase(cfg *SetupConfig) error {
	dsn := buildGoldenDBDSN(&cfg.Database, cfg.Database.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close GoldenDB connection: %v", err)
		}
	}()

	migrationCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	return repository.ApplyMigrations(migrationCtx, db)
}

func createAdminUser(cfg *SetupConfig) (bool, string, error) {
	dsn := buildGoldenDBDSN(&cfg.Database, cfg.Database.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, "", err
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close GoldenDB connection: %v", err)
		}
	}()

	// 使用超时上下文避免安装流程因数据库异常而长时间阻塞。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var adminUsers int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE role = ?", service.RoleAdmin).Scan(&adminUsers); err != nil {
		return false, "", err
	}
	decision := decideAdminBootstrap(adminUsers)
	if !decision.shouldCreate {
		return false, decision.reason, nil
	}

	if strings.TrimSpace(cfg.Admin.Password) == "" {
		password, genErr := generateSecret(16)
		if genErr != nil {
			return false, "", fmt.Errorf("failed to generate admin password: %w", genErr)
		}
		cfg.Admin.Password = password
		fmt.Printf("Generated admin password (one-time): %s\n", cfg.Admin.Password)
		fmt.Println("IMPORTANT: Save this password! It will not be shown again.")
	}

	admin := &service.User{
		Email:       cfg.Admin.Email,
		Role:        service.RoleAdmin,
		Status:      service.StatusActive,
		Balance:     0,
		Concurrency: setupDefaultAdminConcurrency(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := admin.SetPassword(cfg.Admin.Password); err != nil {
		return false, "", err
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO users (email, password_hash, role, balance, concurrency, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		admin.Email,
		admin.PasswordHash,
		admin.Role,
		admin.Balance,
		admin.Concurrency,
		admin.Status,
		admin.CreatedAt,
		admin.UpdatedAt,
	)
	if err != nil {
		return false, "", err
	}
	return true, decision.reason, nil
}

func writeConfigFile(cfg *SetupConfig) error {
	// Ensure timezone has a default value
	tz := cfg.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	// Prepare config for YAML (exclude sensitive data and admin config)
	yamlConfig := struct {
		Server   ServerConfig   `yaml:"server"`
		Database DatabaseConfig `yaml:"database"`
		Redis    RedisConfig    `yaml:"redis"`
		JWT      struct {
			Secret     string `yaml:"secret"`
			ExpireHour int    `yaml:"expire_hour"`
		} `yaml:"jwt"`
		Default struct {
			UserConcurrency int     `yaml:"user_concurrency"`
			UserBalance     float64 `yaml:"user_balance"`
			APIKeyPrefix    string  `yaml:"api_key_prefix"`
			RateMultiplier  float64 `yaml:"rate_multiplier"`
		} `yaml:"default"`
		RateLimit struct {
			RequestsPerMinute int `yaml:"requests_per_minute"`
			BurstSize         int `yaml:"burst_size"`
		} `yaml:"rate_limit"`
		Timezone string `yaml:"timezone"`
	}{
		Server:   cfg.Server,
		Database: cfg.Database,
		Redis:    cfg.Redis,
		JWT: struct {
			Secret     string `yaml:"secret"`
			ExpireHour int    `yaml:"expire_hour"`
		}{
			Secret:     cfg.JWT.Secret,
			ExpireHour: cfg.JWT.ExpireHour,
		},
		Default: struct {
			UserConcurrency int     `yaml:"user_concurrency"`
			UserBalance     float64 `yaml:"user_balance"`
			APIKeyPrefix    string  `yaml:"api_key_prefix"`
			RateMultiplier  float64 `yaml:"rate_multiplier"`
		}{
			UserConcurrency: defaultUserConcurrency,
			UserBalance:     0,
			APIKeyPrefix:    "sk-",
			RateMultiplier:  1.0,
		},
		RateLimit: struct {
			RequestsPerMinute int `yaml:"requests_per_minute"`
			BurstSize         int `yaml:"burst_size"`
		}{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
		Timezone: tz,
	}

	data, err := yaml.Marshal(&yamlConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(GetConfigFilePath(), data, 0600)
}

func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// =============================================================================
// Auto Setup for Docker Deployment
// =============================================================================

// AutoSetupEnabled checks if auto setup is enabled via environment variable
func AutoSetupEnabled() bool {
	val := os.Getenv("AUTO_SETUP")
	return val == "true" || val == "1" || val == "yes"
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// getEnvIntOrDefault gets environment variable as int or returns default value
func getEnvIntOrDefault(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

// AutoSetupFromEnv performs automatic setup using environment variables
// This is designed for Docker deployment where all config is passed via env vars
func AutoSetupFromEnv() error {
	logger.LegacyPrintf("setup", "%s", "Auto setup enabled, configuring from environment variables...")
	logger.LegacyPrintf("setup", "Data directory: %s", GetDataDir())

	// Get timezone from TZ or TIMEZONE env var (TZ is standard for Docker)
	tz := getEnvOrDefault("TZ", "")
	if tz == "" {
		tz = getEnvOrDefault("TIMEZONE", "Asia/Shanghai")
	}

	// Build config from environment variables
	cfg := &SetupConfig{
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DATABASE_HOST", "localhost"),
			Port:     getEnvIntOrDefault("DATABASE_PORT", 3306),
			User:     getEnvOrDefault("DATABASE_USER", "goldendb"),
			Password: getEnvOrDefault("DATABASE_PASSWORD", ""),
			DBName:   getEnvOrDefault("DATABASE_DBNAME", "sub2api"),
			SSLMode:  getEnvOrDefault("DATABASE_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:      getEnvOrDefault("REDIS_HOST", "localhost"),
			Port:      getEnvIntOrDefault("REDIS_PORT", 6379),
			Password:  getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:        getEnvIntOrDefault("REDIS_DB", 0),
			EnableTLS: getEnvOrDefault("REDIS_ENABLE_TLS", "false") == "true",
		},
		Admin: AdminConfig{
			Email:    getEnvOrDefault("ADMIN_EMAIL", ""),
			Password: getEnvOrDefault("ADMIN_PASSWORD", ""),
		},
		Server: ServerConfig{
			Host: getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			Port: getEnvIntOrDefault("SERVER_PORT", 8080),
			Mode: getEnvOrDefault("SERVER_MODE", "release"),
		},
		JWT: JWTConfig{
			Secret:     getEnvOrDefault("JWT_SECRET", ""),
			ExpireHour: getEnvIntOrDefault("JWT_EXPIRE_HOUR", 24),
		},
		Timezone: tz,
	}

	// Generate JWT secret if not provided
	if cfg.JWT.Secret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return fmt.Errorf("failed to generate jwt secret: %w", err)
		}
		cfg.JWT.Secret = secret
		logger.LegacyPrintf("setup", "%s", "Warning: JWT secret auto-generated. Consider setting a fixed secret for production.")
	}
	if err := applyAdminBootstrapCredentials(cfg); err != nil {
		return fmt.Errorf("load admin bootstrap configuration: %w", err)
	}

	// Test database connection
	logger.LegacyPrintf("setup", "%s", "Testing database connection...")
	if err := TestDatabaseConnection(&cfg.Database); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Database connection successful")

	// Test Redis connection
	logger.LegacyPrintf("setup", "%s", "Testing Redis connection...")
	if err := TestRedisConnection(&cfg.Redis); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Redis connection successful")

	// Initialize database
	logger.LegacyPrintf("setup", "%s", "Initializing database...")
	if err := initializeDatabase(cfg); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Database initialized successfully")

	// Create admin user
	logger.LegacyPrintf("setup", "%s", "Creating admin user...")
	created, reason, err := createAdminUser(cfg)
	if err != nil {
		return fmt.Errorf("admin user creation failed: %w", err)
	}
	if created {
		logger.LegacyPrintf("setup", "Admin user created: %s", cfg.Admin.Email)
	} else {
		switch reason {
		case adminBootstrapReasonAdminExists:
			logger.LegacyPrintf("setup", "%s", "Admin user already exists, skipping admin bootstrap")
		default:
			logger.LegacyPrintf("setup", "%s", "Admin bootstrap skipped")
		}
	}

	// Preserve an existing operator-provided configuration.
	_, configExists, err := existingBootstrapConfigFile()
	if err != nil {
		return fmt.Errorf("check config file: %w", err)
	}
	if !configExists {
		logger.LegacyPrintf("setup", "%s", "Writing configuration file...")
		if err := writeConfigFile(cfg); err != nil {
			return fmt.Errorf("config file creation failed: %w", err)
		}
		logger.LegacyPrintf("setup", "%s", "Configuration file created")
	} else {
		logger.LegacyPrintf("setup", "%s", "Existing configuration file preserved")
	}

	// Create installation lock file
	if err := createInstallLock(); err != nil {
		return fmt.Errorf("failed to create install lock: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Installation lock created")

	logger.LegacyPrintf("setup", "%s", "Auto setup completed successfully!")
	return nil
}
