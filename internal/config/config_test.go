package config

import (
	"strings"
	"testing"
	"time"
)

func validAuthConfig() AuthConfig {
	return AuthConfig{
		AccessSecret:  strings.Repeat("a", minSecretLength),
		RefreshSecret: strings.Repeat("b", minSecretLength),
		AccessTTL:     defaultAccessTTL,
		RefreshTTL:    defaultRefreshTTL,
	}
}

func TestAuthConfigValidate_ValidConfig_ReturnsNil(t *testing.T) {
	if err := validAuthConfig().Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestAuthConfigValidate_InvalidConfig_ReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AuthConfig)
		want   string
	}{
		{"short access secret", func(c *AuthConfig) { c.AccessSecret = "short" }, "JWT_SECRET"},
		{"short refresh secret", func(c *AuthConfig) { c.RefreshSecret = "" }, "JWT_REFRESH_SECRET"},
		{"same secrets", func(c *AuthConfig) { c.RefreshSecret = c.AccessSecret }, "must differ"},
		{"invalid access ttl", func(c *AuthConfig) { c.AccessTTL = 0 }, "JWT_ACCESS_TTL"},
		{"refresh shorter than access", func(c *AuthConfig) { c.RefreshTTL = time.Minute }, "JWT_REFRESH_TTL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validAuthConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestLoad_DurationEnv_ParsesOrFallsBack(t *testing.T) {
	t.Setenv("JWT_ACCESS_TTL", "")
	t.Setenv("JWT_REFRESH_TTL", "not-a-duration")

	cfg := Load()
	if cfg.Auth.AccessTTL != defaultAccessTTL {
		t.Fatalf("AccessTTL = %v, want default %v", cfg.Auth.AccessTTL, defaultAccessTTL)
	}
	if cfg.Auth.RefreshTTL != 0 {
		t.Fatalf("RefreshTTL = %v, want 0 for an invalid value", cfg.Auth.RefreshTTL)
	}
}

func TestLoad_TrustedProxies_SplitsAndTrims(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", " 10.0.0.1 , ,192.168.0.0/16 ")

	got := Load().TrustedProxies
	if len(got) != 2 || got[0] != "10.0.0.1" || got[1] != "192.168.0.0/16" {
		t.Fatalf("TrustedProxies = %q, want [10.0.0.1 192.168.0.0/16]", got)
	}
}

func TestLoad_NoTrustedProxies_ReturnsEmpty(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	if got := Load().TrustedProxies; len(got) != 0 {
		t.Fatalf("TrustedProxies = %q, want empty", got)
	}
}
