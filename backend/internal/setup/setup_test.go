package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestNeedsSetupAtUsesInstallLockAsCommitMarker(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".installed")
	if !needsSetupAt(lockPath) {
		t.Fatal("missing install lock must require setup")
	}
	if err := os.WriteFile(lockPath, []byte("installed_at=test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if needsSetupAt(lockPath) {
		t.Fatal("existing install lock must mark setup complete")
	}
}

func TestApplyAdminBootstrapCredentialsPrefersConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	content := []byte("default:\n  admin_email: configured@example.com\n  admin_password: configured-secret\n")
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), content, 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &SetupConfig{Admin: AdminConfig{Email: "env@example.com", Password: "env-secret"}}
	if err := applyAdminBootstrapCredentials(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Admin.Email != "configured@example.com" || cfg.Admin.Password != "configured-secret" {
		t.Fatalf("config credentials not preferred: %#v", cfg.Admin)
	}
}

func TestApplyAdminBootstrapCredentialsUsesRequiredDefaults(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	cfg := &SetupConfig{}
	if err := applyAdminBootstrapCredentials(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Admin.Email != "admin@example.com" || cfg.Admin.Password != "admin123" {
		t.Fatalf("unexpected defaults: %#v", cfg.Admin)
	}
}

func TestDecideAdminBootstrap(t *testing.T) {
	tests := []struct {
		name       string
		adminUsers int64
		should     bool
		reason     string
	}{
		{name: "admin missing should create", adminUsers: 0, should: true, reason: adminBootstrapReasonAdminMissing},
		{name: "admin exists should skip", adminUsers: 1, should: false, reason: adminBootstrapReasonAdminExists},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decideAdminBootstrap(tc.adminUsers)
			if got.shouldCreate != tc.should || got.reason != tc.reason {
				t.Fatalf("decision=%#v, want should=%v reason=%q", got, tc.should, tc.reason)
			}
		})
	}
}

func TestSetupDefaultAdminConcurrency(t *testing.T) {
	t.Run("simple mode admin uses higher concurrency", func(t *testing.T) {
		t.Setenv("RUN_MODE", "simple")
		if got := setupDefaultAdminConcurrency(); got != simpleModeAdminConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, simpleModeAdminConcurrency)
		}
	})

	t.Run("standard mode keeps existing default", func(t *testing.T) {
		t.Setenv("RUN_MODE", "standard")
		if got := setupDefaultAdminConcurrency(); got != defaultUserConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, defaultUserConcurrency)
		}
	})
}

func TestWriteConfigFileKeepsDefaultUserConcurrency(t *testing.T) {
	t.Setenv("RUN_MODE", "simple")
	t.Setenv("DATA_DIR", t.TempDir())

	if err := writeConfigFile(&SetupConfig{}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}
	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "user_concurrency: 5") {
		t.Fatalf("config missing default user concurrency, got:\n%s", string(data))
	}
}

func TestBuildDatabaseConnectionDSNsUsesServerConnectionForBootstrap(t *testing.T) {
	cfg := &DatabaseConfig{
		Host: "db", Port: 3306, User: "sub2api", Password: "secret",
		DBName: "sub2api", SSLMode: "disable",
	}
	bootstrapDSN, targetDSN := buildDatabaseConnectionDSNs(cfg)
	if strings.Contains(bootstrapDSN, "/sub2api?") {
		t.Fatalf("bootstrap DSN = %q, should not connect to target database before checking/creating it", bootstrapDSN)
	}
	if !strings.Contains(targetDSN, "/sub2api?") {
		t.Fatalf("target DSN = %q, want configured database", targetDSN)
	}
}

func TestSetupConfigFromAppConfig(t *testing.T) {
	appCfg := &config.Config{
		Database: config.DatabaseConfig{Host: "db", Port: 3307, User: "user", Password: "pass", DBName: "name", SSLMode: "require"},
		Redis:    config.RedisConfig{Host: "cache", Port: 6380, Password: "redis-pass", DB: 2, EnableTLS: true},
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 9090, Mode: "release"},
		JWT:      config.JWTConfig{Secret: "jwt-secret", ExpireHour: 48},
		Default:  config.DefaultConfig{AdminEmail: "admin@configured.test", AdminPassword: "configured-pass"},
		Timezone: "Asia/Shanghai",
	}
	got := setupConfigFromAppConfig(appCfg)
	if got.Database.Host != "db" || got.Database.Port != 3307 || got.Database.DBName != "name" {
		t.Fatalf("database config not copied: %#v", got.Database)
	}
	if got.Redis.Host != "cache" || !got.Redis.EnableTLS || got.Redis.DB != 2 {
		t.Fatalf("redis config not copied: %#v", got.Redis)
	}
	if got.Admin.Email != "admin@configured.test" || got.Admin.Password != "configured-pass" {
		t.Fatalf("admin config not copied: %#v", got.Admin)
	}
	if got.JWT.Secret != "jwt-secret" || got.Server.Port != 9090 || got.Timezone != "Asia/Shanghai" {
		t.Fatalf("runtime config not copied: %#v", got)
	}
}
