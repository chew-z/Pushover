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

// LoadConfig loads the configuration from environment variables
func LoadConfig() Config {
	// Parse priority with error handling, default to PriorityLow
	priority := int(pushover.PriorityLow)
	if p, err := strconv.Atoi(os.Getenv("PUSHOVER_PRIORITY")); err == nil {
		priority = p
	}

	sound := os.Getenv("PUSHOVER_SOUND")
	if sound == "" {
		sound = pushover.SoundVibrate
	}

	// Parse expire time with error handling, default to 180 seconds
	expireTime := 180
	if e, err := strconv.Atoi(os.Getenv("PUSHOVER_EXPIRE")); err == nil && e != 0 {
		expireTime = e
	}

	// Parse retry time with error handling, default to 60 seconds
	retryTime := defaultRetryTime
	if r, err := strconv.Atoi(os.Getenv("PUSHOVER_RETRY")); err == nil && r >= 30 {
		retryTime = r
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
	}
}

// NewMCPConfig creates a new MCP configuration from environment variables and flags
func NewMCPConfig(authEnabledFlag bool) (*MCPConfig, error) {
	config := loadMCPConfig(authEnabledFlag)
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	return config, nil
}

// loadMCPConfig reads MCP configuration without validating it.
func loadMCPConfig(authEnabledFlag bool) *MCPConfig {
	config := &MCPConfig{
		// HTTP transport settings
		HTTPAddress:                    getEnvWithDefault("PUSHOVER_HTTP_ADDRESS", defaultHTTPAddress),
		HTTPPath:                       getEnvWithDefault("PUSHOVER_HTTP_PATH", defaultHTTPPath),
		HTTPStateless:                  parseEnvBool("PUSHOVER_HTTP_STATELESS", defaultHTTPStateless),
		HTTPHeartbeat:                  parseEnvDuration("PUSHOVER_HTTP_HEARTBEAT", defaultHTTPHeartbeat),
		HTTPTimeout:                    parseEnvDuration("PUSHOVER_HTTP_TIMEOUT", defaultHTTPTimeout),
		HTTPCORSEnabled:                parseEnvBool("PUSHOVER_HTTP_CORS_ENABLED", defaultHTTPCORSEnabled),
		HTTPCORSOrigins:                parseEnvStringSlice("PUSHOVER_HTTP_CORS_ORIGINS", []string{}),
		HTTPDisableLocalhostProtection: parseEnvBool("PUSHOVER_HTTP_DISABLE_LOCALHOST_PROTECTION", defaultHTTPDisableLocalhostProtection),

		// Authentication settings - prioritize command-line flags over environment
		AuthEnabled:            authEnabledFlag || parseEnvBool("PUSHOVER_AUTH_ENABLED", defaultAuthEnabled),
		AuthSecretKey:          getEnvWithDefault("PUSHOVER_AUTH_SECRET_KEY", ""),
		OAuthEnabled:           parseEnvBool("PUSHOVER_MCP_OAUTH", false),
		OAuthIssuer:            os.Getenv("PUSHOVER_MCP_ISSUER"),
		OAuthResource:          os.Getenv("PUSHOVER_MCP_RESOURCE"),
		OAuthLoginUser:         os.Getenv("PUSHOVER_MCP_LOGIN_USER"),
		OAuthLoginPasswordHash: os.Getenv("PUSHOVER_MCP_LOGIN_PASSWORD_HASH"),
		OAuthAccessTTL:         parseEnvDuration("PUSHOVER_MCP_ACCESS_TTL", defaultOAuthAccessTTL),
		OAuthRefreshTTL:        parseEnvDuration("PUSHOVER_MCP_REFRESH_TTL", defaultOAuthRefreshTTL),

		// Pushover-specific settings
		PushoverAppKey:       os.Getenv("APP_KEY"),
		PushoverRecipientKey: os.Getenv("RECIPIENT_KEY"),
		PushoverDeviceName:   os.Getenv("DEVICE_NAME"),
		PushoverDefaultTitle: os.Getenv("DEFAULT_TITLE"),
	}
	if config.OAuthEnabled {
		config.AuthEnabled = true
	}

	// Parse Pushover priority with error handling, default to PriorityLow
	config.PushoverDefaultPriority = int(pushover.PriorityLow)
	if p, err := strconv.Atoi(os.Getenv("PUSHOVER_PRIORITY")); err == nil {
		config.PushoverDefaultPriority = p
	}

	// Parse Pushover sound with default
	config.PushoverDefaultSound = os.Getenv("PUSHOVER_SOUND")
	if config.PushoverDefaultSound == "" {
		config.PushoverDefaultSound = pushover.SoundVibrate
	}

	// Parse expire time with error handling, default to 180 seconds
	config.PushoverDefaultExpire = defaultExpireTime
	if e, err := strconv.Atoi(os.Getenv("PUSHOVER_EXPIRE")); err == nil && e != 0 {
		config.PushoverDefaultExpire = e
	}

	// Parse retry time with error handling, default to 60 seconds
	config.PushoverDefaultRetry = defaultRetryTime
	if r, err := strconv.Atoi(os.Getenv("PUSHOVER_RETRY")); err == nil && r >= 30 {
		config.PushoverDefaultRetry = r
	}

	return config
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

	// Validate priority range (-2 to 2)
	if c.PushoverDefaultPriority < -2 || c.PushoverDefaultPriority > 2 {
		return fmt.Errorf("PUSHOVER_PRIORITY must be between -2 and 2, got %d", c.PushoverDefaultPriority)
	}

	// Validate expire and retry for emergency priority
	if c.PushoverDefaultPriority == int(pushover.PriorityEmergency) && c.PushoverDefaultExpire <= 0 {
		return errors.New("PUSHOVER_EXPIRE must be > 0 for emergency priority messages")
	}
	if c.PushoverDefaultPriority == int(pushover.PriorityEmergency) && c.PushoverDefaultRetry < 30 {
		return errors.New("PUSHOVER_RETRY must be >= 30 seconds for emergency priority messages")
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

// parseEnvBool parses a boolean environment variable with a default value
func parseEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

// parseEnvDuration parses a duration environment variable with a default value
func parseEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return parsed
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
