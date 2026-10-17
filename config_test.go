package main

import (
	"strconv"
	"strings"
	"testing"
)

func TestOAuthConfigValidation(t *testing.T) {
	valid := testOAuthMCPConfig
	for _, test := range []struct {
		name, want string
		change     func(*MCPConfig)
	}{
		{"missing issuer", "issuer must be an absolute URL", func(c *MCPConfig) { c.OAuthIssuer = "" }},
		{"relative resource", "resource must be an absolute URL", func(c *MCPConfig) { c.OAuthResource = "/mcp" }},
		{"short secret", "at least 32", func(c *MCPConfig) { c.AuthSecretKey = "short" }},
		{"missing login user", "LoginUser must not be empty", func(c *MCPConfig) { c.OAuthLoginUser = "" }},
		{"bad hash", "LoginPasswordHash", func(c *MCPConfig) { c.OAuthLoginPasswordHash = "bad" }},
		{"refresh not after access", "RefreshTokenTTL must be greater", func(c *MCPConfig) { c.OAuthRefreshTTL = c.OAuthAccessTTL }},
		{"root path", "must not be /", func(c *MCPConfig) { c.HTTPPath = "/" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid(t)
			test.change(cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() = %v, want %q", err, test.want)
			}
		})
	}
	if err := valid(t).Validate(); err != nil {
		t.Fatalf("valid configuration: %v", err)
	}
}

func TestLoadMCPConfigOAuthImpliesAuthentication(t *testing.T) {
	t.Setenv("PUSHOVER_MCP_OAUTH", "true")
	if !loadMCPConfig(false).AuthEnabled {
		t.Fatal("OAuth did not enable auth")
	}
}

func TestMCPConfigValidationExistingRules(t *testing.T) {
	cfg := testMCPConfig()
	cfg.PushoverAppKey, cfg.PushoverRecipientKey = "key", "recipient"
	cfg.PushoverDefaultPriority = 0
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.PushoverDefaultPriority = 3
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid priority accepted")
	}
}

// TestMCPConfigRejectsExpireAboveMaximum verifies the configured Pushover expiration limit.
func TestMCPConfigRejectsExpireAboveMaximum(t *testing.T) {
	t.Setenv("APP_KEY", "app-key")
	t.Setenv("RECIPIENT_KEY", "recipient-key")
	t.Setenv("PUSHOVER_AUTH_ENABLED", "false")
	t.Setenv("PUSHOVER_MCP_OAUTH", "false")
	t.Setenv("PUSHOVER_PRIORITY", "0")
	t.Setenv("PUSHOVER_EXPIRE", strconv.Itoa(maxExpireSeconds+1))

	_, err := NewMCPConfig(false)
	if err == nil || !strings.Contains(err.Error(), "PUSHOVER_EXPIRE") {
		t.Fatalf("NewMCPConfig() error = %v, want PUSHOVER_EXPIRE limit error", err)
	}
}

// TestPriorityZeroConfigParsing verifies that both configuration paths accept priority zero.
func TestPriorityZeroConfigParsing(t *testing.T) {
	t.Run("LoadConfig defaults to low priority when unset", func(t *testing.T) {
		t.Setenv("PUSHOVER_PRIORITY", "")
		if priority := LoadConfig().Priority; priority != -1 {
			t.Errorf("LoadConfig().Priority = %d, want -1", priority)
		}
	})

	t.Run("LoadConfig accepts explicit zero", func(t *testing.T) {
		t.Setenv("PUSHOVER_PRIORITY", "0")
		if priority := LoadConfig().Priority; priority != 0 {
			t.Errorf("LoadConfig().Priority = %d, want 0", priority)
		}
	})

	t.Run("NewMCPConfig accepts explicit zero", func(t *testing.T) {
		t.Setenv("APP_KEY", "app-key")
		t.Setenv("RECIPIENT_KEY", "recipient-key")
		t.Setenv("PUSHOVER_AUTH_ENABLED", "false")
		t.Setenv("PUSHOVER_MCP_OAUTH", "false")
		t.Setenv("PUSHOVER_PRIORITY", "0")

		config, err := NewMCPConfig(false)
		if err != nil {
			t.Fatalf("NewMCPConfig() error = %v", err)
		}
		if config.PushoverDefaultPriority != 0 {
			t.Errorf("PushoverDefaultPriority = %d, want 0", config.PushoverDefaultPriority)
		}
	})
}
