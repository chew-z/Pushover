package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chew-z/Pushover/internal/oauth"
	"github.com/gregdel/pushover"
)

// Config holds the application configuration
type Config struct {
	AppKey       string
	RecipientKey string
	DeviceName   string
	DefaultTitle string
	Priority     int
	Sound        string
	ExpireTime   int
	RetryTime    int
}

// validateEmergency enforces the Pushover API rules for emergency priority:
// retry must be at least 30 seconds and expire must be 1..maxExpireSeconds.
// Other priorities have no retry/expire constraints.
func validateEmergency(priority, expire, retry int) error {
	if priority != int(pushover.PriorityEmergency) {
		return nil
	}
	if retry < 30 {
		return fmt.Errorf("retry must be at least 30 seconds for emergency priority, got %d", retry)
	}
	if expire < 1 || expire > maxExpireSeconds {
		return fmt.Errorf("expire must be between 1 and %d seconds for emergency priority, got %d", maxExpireSeconds, expire)
	}
	return nil
}

// Validate checks the Pushover configuration values.
func (c Config) Validate() error {
	if c.Priority < -2 || c.Priority > 2 {
		return fmt.Errorf("PUSHOVER_PRIORITY must be between -2 and 2, got %d", c.Priority)
	}
	if c.ExpireTime > maxExpireSeconds {
		return fmt.Errorf("PUSHOVER_EXPIRE must be at most %d seconds, got %d", maxExpireSeconds, c.ExpireTime)
	}
	return validateEmergency(c.Priority, c.ExpireTime, c.RetryTime)
}

// MCPConfig holds the MCP server configuration
type MCPConfig struct {
	// HTTP transport settings
	HTTPAddress                    string
	HTTPPath                       string
	HTTPStateless                  bool
	HTTPHeartbeat                  time.Duration
	HTTPTimeout                    time.Duration
	HTTPCORSEnabled                bool
	HTTPCORSOrigins                []string
	HTTPDisableLocalhostProtection bool

	// Authentication settings
	AuthEnabled            bool
	AuthSecretKey          string
	OAuthEnabled           bool
	OAuthIssuer            string
	OAuthResource          string
	OAuthLoginUser         string
	OAuthLoginPasswordHash string
	OAuthAccessTTL         time.Duration
	OAuthRefreshTTL        time.Duration

	// Pushover-specific settings (from existing Config)
	PushoverAppKey          string
	PushoverRecipientKey    string
	PushoverDeviceName      string
	PushoverDefaultTitle    string
	PushoverDefaultPriority int
	PushoverDefaultSound    string
	PushoverDefaultExpire   int
	PushoverDefaultRetry    int
}

// defaultHTTPAddress and related constants define the built-in configuration values.
//
// Pushover is always deployed behind a reverse proxy (nginx) that forwards
// the real Host header. mcp-go's DNS-rebinding protection only guards against
// a browser hitting a loopback-bound server directly, which doesn't apply
// here — so it must stay off by default or every proxied request 403s with
// "invalid Host header".
const (
	defaultHTTPAddress                    = "127.0.0.1:8080"
	defaultHTTPPath                       = "/mcp"
	defaultHTTPStateless                  = false
	defaultHTTPHeartbeat                  = 30 * time.Second
	defaultHTTPTimeout                    = 30 * time.Second
	defaultHTTPCORSEnabled                = true
	defaultHTTPDisableLocalhostProtection = true
	defaultAuthEnabled                    = false
	defaultOAuthAccessTTL                 = time.Hour
	defaultOAuthRefreshTTL                = 720 * time.Hour
	defaultExpireTime                     = 180
	defaultRetryTime                      = 60
)

// LoadConfig loads the configuration from environment variables.
func LoadConfig() (Config, error) {
	priority, err := parseEnvInt("PUSHOVER_PRIORITY", int(pushover.PriorityLow))
	if err != nil {
		return Config{}, err
	}

	sound := os.Getenv("PUSHOVER_SOUND")
	if sound == "" {
		sound = pushover.SoundVibrate
	}

	expireTime, err := parseEnvInt("PUSHOVER_EXPIRE", defaultExpireTime)
	if err != nil {
		return Config{}, err
	}

	retryTime, err := parseEnvInt("PUSHOVER_RETRY", defaultRetryTime)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppKey:       os.Getenv("APP_KEY"),
		RecipientKey: os.Getenv("RECIPIENT_KEY"),
		DeviceName:   os.Getenv("DEVICE_NAME"),
		DefaultTitle: os.Getenv("DEFAULT_TITLE"),
		Priority:     priority,
		Sound:        sound,
		ExpireTime:   expireTime,
		RetryTime:    retryTime,
	}, nil
}

// NewMCPConfig creates a new MCP configuration from environment variables and flags.
func NewMCPConfig(authEnabledFlag bool) (*MCPConfig, error) {
	config, err := loadMCPConfig(authEnabledFlag)
	if err != nil {
		return nil, fmt.Errorf("configuration loading failed: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	return config, nil
}

// loadMCPConfig reads MCP configuration without validating it.
func loadMCPConfig(authEnabledFlag bool) (*MCPConfig, error) {
	config := &MCPConfig{}

	// HTTP transport settings
	config.HTTPAddress = getEnvWithDefault("PUSHOVER_HTTP_ADDRESS", defaultHTTPAddress)
	config.HTTPPath = getEnvWithDefault("PUSHOVER_HTTP_PATH", defaultHTTPPath)
	var err error
	config.HTTPStateless, err = parseEnvBool("PUSHOVER_HTTP_STATELESS", defaultHTTPStateless)
	if err != nil {
		return nil, err
	}
	config.HTTPHeartbeat, err = parseEnvDuration("PUSHOVER_HTTP_HEARTBEAT", defaultHTTPHeartbeat)
	if err != nil {
		return nil, err
	}
	config.HTTPTimeout, err = parseEnvDuration("PUSHOVER_HTTP_TIMEOUT", defaultHTTPTimeout)
	if err != nil {
		return nil, err
	}
	config.HTTPCORSEnabled, err = parseEnvBool("PUSHOVER_HTTP_CORS_ENABLED", defaultHTTPCORSEnabled)
	if err != nil {
		return nil, err
	}
	config.HTTPCORSOrigins = parseEnvStringSlice("PUSHOVER_HTTP_CORS_ORIGINS", []string{})
	config.HTTPDisableLocalhostProtection, err = parseEnvBool("PUSHOVER_HTTP_DISABLE_LOCALHOST_PROTECTION", defaultHTTPDisableLocalhostProtection)
	if err != nil {
		return nil, err
	}

	// Authentication settings - prioritize command-line flags over environment
	envAuthEnabled, err := parseEnvBool("PUSHOVER_AUTH_ENABLED", defaultAuthEnabled)
	if err != nil {
		return nil, err
	}
	config.AuthEnabled = authEnabledFlag || envAuthEnabled
	config.AuthSecretKey = getEnvWithDefault("PUSHOVER_AUTH_SECRET_KEY", "")
	config.OAuthEnabled, err = parseEnvBool("PUSHOVER_MCP_OAUTH", false)
	if err != nil {
		return nil, err
	}
	config.OAuthIssuer = os.Getenv("PUSHOVER_MCP_ISSUER")
	config.OAuthResource = os.Getenv("PUSHOVER_MCP_RESOURCE")
	config.OAuthLoginUser = os.Getenv("PUSHOVER_MCP_LOGIN_USER")
	config.OAuthLoginPasswordHash = os.Getenv("PUSHOVER_MCP_LOGIN_PASSWORD_HASH")
	config.OAuthAccessTTL, err = parseEnvDuration("PUSHOVER_MCP_ACCESS_TTL", defaultOAuthAccessTTL)
	if err != nil {
		return nil, err
	}
	config.OAuthRefreshTTL, err = parseEnvDuration("PUSHOVER_MCP_REFRESH_TTL", defaultOAuthRefreshTTL)
	if err != nil {
		return nil, err
	}

	// Pushover-specific settings
	config.PushoverAppKey = os.Getenv("APP_KEY")
	config.PushoverRecipientKey = os.Getenv("RECIPIENT_KEY")
	config.PushoverDeviceName = os.Getenv("DEVICE_NAME")
	config.PushoverDefaultTitle = os.Getenv("DEFAULT_TITLE")
	if config.OAuthEnabled {
		config.AuthEnabled = true
	}

	config.PushoverDefaultPriority, err = parseEnvInt("PUSHOVER_PRIORITY", int(pushover.PriorityLow))
	if err != nil {
		return nil, err
	}

	// Parse Pushover sound with default
	config.PushoverDefaultSound = os.Getenv("PUSHOVER_SOUND")
	if config.PushoverDefaultSound == "" {
		config.PushoverDefaultSound = pushover.SoundVibrate
	}

	config.PushoverDefaultExpire, err = parseEnvInt("PUSHOVER_EXPIRE", defaultExpireTime)
	if err != nil {
		return nil, err
	}

	config.PushoverDefaultRetry, err = parseEnvInt("PUSHOVER_RETRY", defaultRetryTime)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks the configuration for required fields and valid values
func (c *MCPConfig) Validate() error {
	if !strings.HasPrefix(c.HTTPPath, "/") || c.HTTPPath == "/" {
		return errors.New("PUSHOVER_HTTP_PATH must start with / and must not be /")
	}
	// Validate required Pushover fields
	if c.PushoverAppKey == "" {
		return errors.New("APP_KEY environment variable is required")
	}

	if c.PushoverRecipientKey == "" {
		return errors.New("RECIPIENT_KEY environment variable is required")
	}

	if err := c.ToLegacyConfig().Validate(); err != nil {
		return err
	}

	// Validate auth secret key if auth is enabled
	if c.AuthEnabled && c.AuthSecretKey == "" {
		return errors.New("PUSHOVER_AUTH_SECRET_KEY is required when authentication is enabled")
	}
	if c.OAuthEnabled {
		if !c.AuthEnabled {
			return errors.New("OAuth requires authentication to be enabled")
		}
		if len(c.AuthSecretKey) < 32 {
			return errors.New("PUSHOVER_AUTH_SECRET_KEY must be at least 32 bytes when OAuth is enabled")
		}
		if _, err := c.OAuthConfig(); err != nil {
			return fmt.Errorf("validate OAuth configuration: %w", err)
		}
	}

	return nil
}

// OAuthConfig derives the OAuth server configuration from the MCP settings.
func (c *MCPConfig) OAuthConfig() (oauth.Config, error) {
	cfg := oauth.Config{
		Issuer: c.OAuthIssuer, Resource: c.OAuthResource, LoginUser: c.OAuthLoginUser,
		LoginPasswordHash: c.OAuthLoginPasswordHash, AccessTokenTTL: c.OAuthAccessTTL,
		RefreshTokenTTL: c.OAuthRefreshTTL,
		Signer:          HS256Signer{Secret: c.AuthSecretKey, Issuer: c.OAuthIssuer, Audience: c.OAuthResource},
	}
	if err := cfg.Validate(); err != nil {
		return oauth.Config{}, fmt.Errorf("validate OAuth config: %w", err)
	}
	return cfg, nil
}

// ToLegacyConfig converts MCPConfig to the legacy Config structure for compatibility
func (c *MCPConfig) ToLegacyConfig() Config {
	return Config{
		AppKey:       c.PushoverAppKey,
		RecipientKey: c.PushoverRecipientKey,
		DeviceName:   c.PushoverDeviceName,
		DefaultTitle: c.PushoverDefaultTitle,
		Priority:     c.PushoverDefaultPriority,
		Sound:        c.PushoverDefaultSound,
		ExpireTime:   c.PushoverDefaultExpire,
		RetryTime:    c.PushoverDefaultRetry,
	}
}

// Helper functions for environment variable parsing

// getEnvWithDefault returns the value of the environment variable or the default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseEnvBool parses a boolean environment variable with a default value.
func parseEnvBool(key string, defaultValue bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: invalid value %q: %w", key, value, err)
	}

	return parsed, nil
}

// parseEnvDuration parses a duration environment variable with a default value.
func parseEnvDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid value %q: %w", key, value, err)
	}

	return parsed, nil
}

// parseEnvInt parses an integer environment variable with a default value.
func parseEnvInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid value %q: %w", key, value, err)
	}

	return parsed, nil
}

// parseEnvStringSlice parses a comma-separated string environment variable into a slice
func parseEnvStringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}

	return result
}
