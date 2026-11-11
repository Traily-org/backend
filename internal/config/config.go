package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// minSecretLength is the minimum length, in bytes, of the JWT secrets.
const minSecretLength = 32

// Default token lifetimes, as specified in the architecture doc.
const (
	defaultAccessTTL  = 15 * time.Minute
	defaultRefreshTTL = 30 * 24 * time.Hour
)

// Config is the application configuration, read from the environment.
type Config struct {
	Port string
	DB   DBConfig
	Auth AuthConfig
	// TrustedProxies is the comma-separated TRUSTED_PROXIES list of proxy
	// IPs or CIDRs allowed to set X-Forwarded-For. Empty trusts none.
	TrustedProxies []string
}

// AuthConfig holds the JWT secrets and token lifetimes.
type AuthConfig struct {
	// AccessSecret signs access tokens (JWT_SECRET).
	AccessSecret string
	// RefreshSecret hashes refresh tokens before storage (JWT_REFRESH_SECRET).
	RefreshSecret string
	// AccessTTL is the access token lifetime (JWT_ACCESS_TTL, default 15m).
	AccessTTL time.Duration
	// RefreshTTL is the refresh token lifetime (JWT_REFRESH_TTL, default 720h).
	RefreshTTL time.Duration
}

// DBConfig holds the PostgreSQL connection settings.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// Load reads the configuration from the environment, applying defaults.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:           port,
		DB:             loadDBConfig(),
		Auth:           loadAuthConfig(),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}
}

// splitList splits a comma-separated value, dropping empty items.
func splitList(v string) []string {
	var items []string
	for _, item := range strings.Split(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func loadAuthConfig() AuthConfig {
	return AuthConfig{
		AccessSecret:  os.Getenv("JWT_SECRET"),
		RefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		AccessTTL:     durationEnv("JWT_ACCESS_TTL", defaultAccessTTL),
		RefreshTTL:    durationEnv("JWT_REFRESH_TTL", defaultRefreshTTL),
	}
}

// durationEnv parses the duration in the named variable, falling back to def
// when it is unset. An invalid value yields 0, which Validate rejects.
func durationEnv(name string, def time.Duration) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0
	}
	return d
}

// Validate reports a configuration that the API must not start with.
func (c AuthConfig) Validate() error {
	var errs []error
	if len(c.AccessSecret) < minSecretLength {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least %d bytes", minSecretLength))
	}
	if len(c.RefreshSecret) < minSecretLength {
		errs = append(errs, fmt.Errorf("JWT_REFRESH_SECRET must be at least %d bytes", minSecretLength))
	}
	if c.AccessSecret != "" && c.AccessSecret == c.RefreshSecret {
		errs = append(errs, errors.New("JWT_SECRET and JWT_REFRESH_SECRET must differ"))
	}
	if c.AccessTTL <= 0 {
		errs = append(errs, errors.New("JWT_ACCESS_TTL must be a positive duration"))
	}
	if c.RefreshTTL <= c.AccessTTL {
		errs = append(errs, errors.New("JWT_REFRESH_TTL must be longer than JWT_ACCESS_TTL"))
	}
	return errors.Join(errs...)
}

func loadDBConfig() DBConfig {
	dbConfig := DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	if dbConfig.Host == "" {
		dbConfig.Host = "localhost"
	}
	if dbConfig.Port == "" {
		dbConfig.Port = "5432"
	}
	if dbConfig.SSLMode == "" {
		dbConfig.SSLMode = "disable"
	}

	return dbConfig
}

// DSN returns the PostgreSQL connection string.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}
