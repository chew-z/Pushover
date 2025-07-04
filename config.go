package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gregdel/pushover"
)

// MCPConfig holds the MCP server configuration
type MCPConfig struct {
	// HTTP transport settings
	HTTPAddress     string
	HTTPStateless   bool
	HTTPHeartbeat   time.Duration
	HTTPTimeout     time.Duration
	HTTPCORSEnabled bool
	HTTPCORSOrigins []string

	// Authentication settings
	AuthEnabled   bool
	AuthSecretKey string

	// Pushover-specific settings (from existing Config)
	PushoverAppKey          string
	PushoverRecipientKey    string
	PushoverDeviceName      string
	PushoverDefaultTitle    string
	PushoverDefaultPriority int
	PushoverDefaultSound    string
	PushoverDefaultExpire   int
}

// Default configuration values
const (
	defaultHTTPAddress     = ":8080"
	defaultHTTPStateless   = false
	defaultHTTPHeartbeat   = 30 * time.Second
	defaultHTTPTimeout     = 30 * time.Second
	defaultHTTPCORSEnabled = true
	defaultAuthEnabled     = false
	defaultExpireTime      = 180
)

// NewMCPConfig creates a new MCP configuration from environment variables and flags
func NewMCPConfig(authEnabledFlag bool) (*MCPConfig, error) {
	config := &MCPConfig{
		// HTTP transport settings
		HTTPAddress:     getEnvWithDefault("PUSHOVER_HTTP_ADDRESS", defaultHTTPAddress),
		HTTPStateless:   parseEnvBool("PUSHOVER_HTTP_STATELESS", defaultHTTPStateless),
		HTTPHeartbeat:   parseEnvDuration("PUSHOVER_HTTP_HEARTBEAT", defaultHTTPHeartbeat),
		HTTPTimeout:     parseEnvDuration("PUSHOVER_HTTP_TIMEOUT", defaultHTTPTimeout),
		HTTPCORSEnabled: parseEnvBool("PUSHOVER_HTTP_CORS_ENABLED", defaultHTTPCORSEnabled),
		HTTPCORSOrigins: parseEnvStringSlice("PUSHOVER_HTTP_CORS_ORIGINS", []string{"*"}),

		// Authentication settings - prioritize command-line flags over environment
		AuthEnabled:   authEnabledFlag || parseEnvBool("PUSHOVER_AUTH_ENABLED", defaultAuthEnabled),
		AuthSecretKey: getEnvWithDefault("PUSHOVER_AUTH_SECRET_KEY", ""),

		// Pushover-specific settings
		PushoverAppKey:       os.Getenv("APP_KEY"),
		PushoverRecipientKey: os.Getenv("RECIPIENT_KEY"),
		PushoverDeviceName:   os.Getenv("DEVICE_NAME"),
		PushoverDefaultTitle: os.Getenv("DEFAULT_TITLE"),
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

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate checks the configuration for required fields and valid values
func (c *MCPConfig) Validate() error {
	// Validate required Pushover fields
	if c.PushoverAppKey == "" {
		return fmt.Errorf("APP_KEY environment variable is required")
	}

	if c.PushoverRecipientKey == "" {
		return fmt.Errorf("RECIPIENT_KEY environment variable is required")
	}

	// Validate priority range (-2 to 2)
	if c.PushoverDefaultPriority < -2 || c.PushoverDefaultPriority > 2 {
		return fmt.Errorf("PUSHOVER_PRIORITY must be between -2 and 2, got %d", c.PushoverDefaultPriority)
	}

	// Validate expire time for emergency priority
	if c.PushoverDefaultPriority == int(pushover.PriorityEmergency) && c.PushoverDefaultExpire <= 0 {
		return fmt.Errorf("PUSHOVER_EXPIRE must be > 0 for emergency priority messages")
	}

	// Validate auth secret key if auth is enabled
	if c.AuthEnabled && c.AuthSecretKey == "" {
		return fmt.Errorf("PUSHOVER_AUTH_SECRET_KEY is required when authentication is enabled")
	}

	return nil
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
