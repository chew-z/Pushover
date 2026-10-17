package main

import (
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
