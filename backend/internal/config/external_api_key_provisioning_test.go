package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestExternalAPIKeyProvisioningDefaultsDisabled(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ExternalAPIKeyProvisioning.Enabled || cfg.ExternalAPIKeyProvisioning.AccessToken != "" {
		t.Fatalf("unexpected defaults: enabled=%v token_present=%v", cfg.ExternalAPIKeyProvisioning.Enabled, cfg.ExternalAPIKeyProvisioning.AccessToken != "")
	}
}

func TestExternalAPIKeyProvisioningEnvironmentOverridesFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	dir := t.TempDir()
	configFile := []byte("external_api_key_provisioning:\n  enabled: false\n  access_token: file-token-that-is-not-selected\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), configFile, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATA_DIR", dir)
	t.Setenv("JWT_SECRET", strings.Repeat("j", 32))
	t.Setenv("EXTERNAL_API_KEY_PROVISIONING_ENABLED", "true")
	t.Setenv("EXTERNAL_API_KEY_PROVISIONING_ACCESS_TOKEN", "env_0123456789abcdef0123456789abcdef")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ExternalAPIKeyProvisioning.Enabled {
		t.Fatal("environment did not override file enabled=false")
	}
	if !strings.HasPrefix(cfg.ExternalAPIKeyProvisioning.AccessToken, "env_") {
		t.Fatal("environment token was not selected")
	}
}

func TestExternalAPIKeyProvisioningValidation(t *testing.T) {
	base := Config{ExternalAPIKeyProvisioning: ExternalAPIKeyProvisioningConfig{Enabled: true}}
	base.JWT.Secret = strings.Repeat("j", 32)
	base.Log.Level = "info"
	base.Log.Format = "console"
	base.Log.StacktraceLevel = "error"
	base.Log.Output.ToStdout = true
	base.Log.Rotation.MaxSizeMB = 1
	base.JWT.ExpireHour = 24
	base.JWT.RefreshTokenExpireDays = 30

	for _, token := range []string{"", "too-short", "0123456789012345678901234567890 "} {
		cfg := base
		cfg.ExternalAPIKeyProvisioning.AccessToken = token
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "external_api_key_provisioning.access_token") {
			t.Fatalf("expected sanitized access token validation error")
		}
	}
}
