---
Title: MCP Server Authentication System Analysis
Repo: Pushover
Commit: 9ad9ee7
Index: Pushover MCP Server
Languages: Go
Date: September 12, 2025 at 08:30 AM
Model: claude-opus-4-1-20250805
---

# Code Research Report: MCP Server Authentication System

## 1. Inputs and Environment

Tools: Codanna MCP tools, file reading tools
Limits: Unknown

## 2. Investigation Path

| Step | Tool        | Input                  | Output summary          | Artifact             |
|------|-------------|------------------------|-------------------------|----------------------|
| 1    | Glob        | "**/*.go"              | Found 9 Go files        | see Evidence §5.1    |
| 2    | Read        | "/Users/rrj/Projekty/Go/src/Pushover/auth.go" | Complete auth implementation | see Evidence §5.2 |
| 3    | Read        | "/Users/rrj/Projekty/Go/src/Pushover/http_server.go" | HTTP server with auth middleware | see Evidence §5.3 |
| 4    | Read        | "/Users/rrj/Projekty/Go/src/Pushover/config.go" | Configuration management | see Evidence §5.4 |
| 5    | Read        | "/Users/rrj/Projekty/Go/src/Pushover/mcp_server.go" | MCP server setup | see Evidence §5.5 |
| 6    | Read        | "/Users/rrj/Projekty/Go/src/Pushover/mcp_cli.go" | CLI argument parsing | see Evidence §5.6 |

## 3. Mechanics of the Code

- JWT tokens use HMAC-SHA256 signing with base64 URL encoding
- Authentication middleware intercepts HTTP requests via context function
- Context propagation carries user identity through request lifecycle
- Token validation occurs before MCP tool execution
- Authentication can be disabled for development/testing
- Token generation available via CLI and HTTP endpoint

## 4. Quantified Findings

- Authentication components: 6 files
- Context keys defined: 6 (user_id, username, role, auth_error, http_method, authenticated)
- JWT claim fields: 5 (user_id, username, role, iat, exp)
- Helper functions: 7 (getUserID, getUsername, getRole, getAuthError, isAuthenticated, isHTTPRequest, extractTokenFromHeader)
- Token expiration default: 744 hours (31 days)
- Maximum message length: 1024 characters

## 5. Evidence

### 5.1 Project Structure
```
/Users/rrj/Projekty/Go/src/Pushover/
├── auth.go          # JWT authentication middleware
├── http_server.go   # HTTP transport with auth integration
├── config.go        # Configuration management
├── mcp_server.go    # MCP server setup and tool wrapping
├── mcp_cli.go       # CLI argument parsing
├── main.go          # Entry point
├── cli.go           # CLI functionality
├── pushover.go      # Pushover client
└── main_test.go     # Test suite
```

### 5.2 Authentication Middleware

```go
// AuthMiddleware handles JWT-based authentication for HTTP requests
type AuthMiddleware struct {
    secretKey []byte
    enabled   bool
}
// auth.go:16-19

// Claims represents the JWT claims structure
type Claims struct {
    UserID    string `json:"user_id"`
    Username  string `json:"username"`
    Role      string `json:"role"`
    IssuedAt  int64  `json:"iat"`
    ExpiresAt int64  `json:"exp"`
}
// auth.go:22-28

// HTTPContextFunc returns a function that adds authentication context to HTTP requests
func (am *AuthMiddleware) HTTPContextFunc() func(context.Context, *http.Request) context.Context
// auth.go:52
```

### 5.3 HTTP Server Integration

```go
// Configure authentication middleware
if hsm.config.AuthEnabled {
    authMiddleware := NewAuthMiddleware(hsm.config.AuthSecretKey, hsm.config.AuthEnabled)
    opts = append(opts, server.WithHTTPContextFunc(authMiddleware.HTTPContextFunc()))
}
// http_server.go:48-51

// Add token generation endpoint (if auth is enabled)
if hsm.config.AuthEnabled {
    mux.HandleFunc("/generate-token", hsm.handleGenerateToken)
}
// http_server.go:69-71
```

### 5.4 Configuration Management

```go
// MCPConfig holds the MCP server configuration
type MCPConfig struct {
    // Authentication settings
    AuthEnabled   bool
    AuthSecretKey string
    // ... other fields
}
// config.go:35-37

// Authentication settings - prioritize command-line flags over environment
AuthEnabled:   authEnabledFlag || parseEnvBool("PUSHOVER_AUTH_ENABLED", defaultAuthEnabled),
AuthSecretKey: getEnvWithDefault("PUSHOVER_AUTH_SECRET_KEY", ""),
// config.go:104-105
```

### 5.5 Tool Wrapping with Authentication

```go
// wrapWithAuth wraps tool handlers with authentication and logging
func wrapWithAuth(
    handler func(context.Context, mcp.CallToolRequest, *MCPConfig) (*mcp.CallToolResult, error),
    toolName string,
    config *MCPConfig,
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
// mcp_server.go:78-82

// Check authentication for HTTP requests
if isHTTPRequest(ctx) && config.AuthEnabled {
    if !isAuthenticated(ctx) {
        authError := getAuthError(ctx)
        return mcp.NewToolResultError(fmt.Sprintf("Authentication required: %s", authError)), nil
    }
    log.Printf("Tool '%s' called by user: %s", toolName, getUsername(ctx))
}
// mcp_server.go:87-93
```

### 5.6 Token Generation CLI

```go
// generateTokenFromArgs generates a JWT token using the provided arguments
func generateTokenFromArgs(userID, username, role string, expirationHours int) error
// mcp_cli.go:101

// Handle token generation
if *generateToken {
    return generateTokenFromArgs(*tokenUserID, *tokenUsername, *tokenRole, *tokenExpiration)
}
// mcp_cli.go:92-94
```

## 6. Implications

- **Security overhead**: Each authenticated request requires ~3 base64 decode operations + 1 HMAC computation
- **Token size**: Standard JWT with 5 claims ≈ 200-300 bytes
- **Context memory**: 6 context values × 8 bytes (pointer) = 48 bytes per request minimum
- **Default token lifetime**: 744 hours = 31 days of validity

## 7. Hidden Patterns

- Unused helper functions marked with `//nolint:unused` comment (getUserID, getRole) suggesting future feature expansion
- Token generation available both via CLI and HTTP endpoint for flexibility
- Authentication bypass for STDIO transport (only HTTP requests are authenticated)
- Context key type safety using custom `contextKey` type to prevent collisions
- Graceful degradation when auth disabled (marks all requests as authenticated)

## 8. Research Opportunities

- Investigate token refresh mechanism implementation
- Explore role-based access control for different MCP tools
- Analyze performance impact of authentication on high-frequency requests
- Review token storage and management strategies
- Examine integration with external identity providers

## 9. Code Map Table

| Component              | File                | Line  | Purpose                                          |
|------------------------|---------------------|-------|--------------------------------------------------|
| AuthMiddleware         | auth.go             | 16    | Main authentication middleware struct           |
| Claims                 | auth.go             | 22    | JWT token claims structure                      |
| HTTPContextFunc        | auth.go             | 52    | Context injection for HTTP requests             |
| validateJWT            | auth.go             | 113   | JWT signature validation                        |
| GenerateJWT            | auth.go             | 150   | JWT token generation                            |
| HTTPServerManager      | http_server.go      | 18    | HTTP transport manager                          |
| handleGenerateToken    | http_server.go      | 188   | HTTP endpoint for token generation              |
| MCPConfig              | config.go           | 25    | MCP server configuration struct                 |
| wrapWithAuth           | mcp_server.go       | 78    | Tool handler authentication wrapper             |
| handleSendNotification | mcp_server.go       | 110   | Main notification sending handler               |
| parseMCPArgs           | mcp_cli.go          | 51    | MCP subcommand argument parser                  |
| generateTokenFromArgs  | mcp_cli.go          | 101   | CLI token generation function                   |

## 10. Confidence and Limitations

- **JWT implementation**: High - Complete HMAC-SHA256 implementation visible
- **Middleware integration**: High - Clear HTTP context propagation
- **Token validation**: High - Explicit signature and expiration checks
- **Configuration flexibility**: High - Multiple configuration methods documented
- **Role-based access**: Low - Role field exists but no enforcement logic found
- **Token refresh**: Unknown - No refresh token mechanism identified

## 11. Footer

GeneratedAt=September 12, 2025 at 08:30 AM  Model=claude-opus-4-1-20250805