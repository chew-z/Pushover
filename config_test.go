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
	config, err := loadMCPConfig(false)
	if err != nil {
		t.Fatal(err)
	}
	if !config.AuthEnabled {
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

func TestValidateEmergency(t *testing.T) {
	testCases := []struct {
		name                    string
		priority, expire, retry int
		want                    string
	}{
		{"non-emergency ignores retry and expire", 0, 0, 5, ""},
		{"minimum emergency values", 2, 1, 30, ""},
		{"maximum emergency expire", 2, maxExpireSeconds, 30, ""},
		{"short emergency retry", 2, 1, 29, "retry"},
		{"zero emergency expire", 2, 0, 30, "expire"},
		{"large emergency expire", 2, maxExpireSeconds + 1, 30, "expire"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEmergency(tc.priority, tc.expire, tc.retry)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateEmergency() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateEmergency() error = %v, want text %q", err, tc.want)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
		want   string
	}{
		{"high priority", Config{Priority: 3}, "PUSHOVER_PRIORITY"},
		{"low priority", Config{Priority: -3}, "PUSHOVER_PRIORITY"},
		{"large expire", Config{Priority: 0, ExpireTime: maxExpireSeconds + 1}, "PUSHOVER_EXPIRE"},
		{"ordinary priority ignores retry and expire", Config{Priority: 1}, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("Config.Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Config.Validate() error = %v, want text %q", err, tc.want)
			}
		})
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
		config, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if config.Priority != -1 {
			t.Errorf("LoadConfig().Priority = %d, want -1", config.Priority)
		}
	})

	t.Run("LoadConfig accepts explicit zero", func(t *testing.T) {
		t.Setenv("PUSHOVER_PRIORITY", "0")
		config, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if config.Priority != 0 {
			t.Errorf("LoadConfig().Priority = %d, want 0", config.Priority)
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

func TestMalformedEnvValues(t *testing.T) {
	testCases := []struct {
		name            string
		key             string
		value           string
		checkLoadConfig bool
	}{
		{"invalid auth enabled", "PUSHOVER_AUTH_ENABLED", "tur", false},
		{"invalid priority", "PUSHOVER_PRIORITY", "high", true},
		{"invalid expire", "PUSHOVER_EXPIRE", "1h", true},
		{"invalid retry", "PUSHOVER_RETRY", "soon", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APP_KEY", "app-key")
			t.Setenv("RECIPIENT_KEY", "recipient-key")
			t.Setenv(tc.key, tc.value)

			assertMalformed := func(source string, err error) {
				t.Helper()
				if err == nil || !strings.Contains(err.Error(), tc.key) || !strings.Contains(err.Error(), tc.value) {
					t.Fatalf("%s error = %v, want %q and %q", source, err, tc.key, tc.value)
				}
			}

			if tc.checkLoadConfig {
				_, err := LoadConfig()
				assertMalformed("LoadConfig()", err)
			}
			_, err := loadMCPConfig(false)
			assertMalformed("loadMCPConfig()", err)
		})
	}

	t.Run("empty priority uses default", func(t *testing.T) {
		t.Setenv("APP_KEY", "app-key")
		t.Setenv("RECIPIENT_KEY", "recipient-key")
		t.Setenv("PUSHOVER_PRIORITY", "")

		config, err := loadMCPConfig(false)
		if err != nil {
			t.Fatal(err)
		}
		if config.PushoverDefaultPriority != -1 {
			t.Errorf("PushoverDefaultPriority = %d, want -1", config.PushoverDefaultPriority)
		}
	})
}
