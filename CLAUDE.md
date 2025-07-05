# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@/Users/rrj/Projekty/CodeAssist/Prompts/GOLANG.md

## Project Overview

This is a well-architected Go command-line utility for sending push notifications via the Pushover service. The application demonstrates excellent software engineering practices with a focus on testability, maintainability, and robust error handling through a single-package design with clear functional separation.

## Architecture & Design Patterns

### Core Design Principles
- **Single Package Architecture**: All functionality in `main` package with clear functional separation
- **Interface-Based Design**: `PushoverClient` interface enables comprehensive mocking and testing
- **Dependency Injection**: `Run()` function accepts client interface for flexible testing scenarios
- **Configuration Management**: Environment-based config with CLI argument override capability
- **Error Handling**: Proper error propagation without `log.Fatal` (follows project conventions)

### Key Design Strengths
1. **Testability**: Interface abstraction enables comprehensive unit and integration testing
2. **Maintainability**: Clear separation of concerns and single responsibility principle
3. **Flexibility**: Multiple configuration methods (environment, CLI args, positional)
4. **Robustness**: Comprehensive input validation and graceful error handling
5. **User Experience**: Rich help system with environment variable documentation

## Key Components & Functions

### Configuration System
- **`Config` struct**: Holds complete application configuration from environment variables
- **`LoadConfig()`**: Loads and validates configuration with sensible defaults
- **Environment Variables**: `APP_KEY`, `RECIPIENT_KEY`, `DEVICE_NAME`, `DEFAULT_TITLE`, `PUSHOVER_PRIORITY`, `PUSHOVER_SOUND`, `PUSHOVER_EXPIRE`

### Command-Line Interface
- **`CLIArgs` struct**: Comprehensive CLI argument structure
- **`ParseArgs()`**: Supports both flag-based and positional arguments with backwards compatibility
- **Features**: Help/version commands, priority levels (-2 to 2), 23 sound options, device targeting

### Client Abstraction
- **`PushoverClient` interface**: Abstracts Pushover API for testability
- **`RealPushoverApp`**: Production implementation wrapping official Pushover client
- **`MockPushoverClient`**: Test implementation with controllable behavior and call verification

### Core Application Logic
- **`CreateMessage()`**: Constructs Pushover messages with full API feature support
- **`SendNotification()`**: Handles API communication with proper error handling
- **`NewPushoverClient()`**: Factory function for client creation with validation
- **`Run()`**: Main application orchestration with dependency injection

## Development Commands

### Build System (Makefile)
```bash
make build              # Build binary with version injection
make clean              # Clean build artifacts
make test               # Run all tests
make lint               # Run golangci-lint
make fmt                # Format code
make install            # Install binary system-wide
```

### Manual Build
```bash
go build -o bin/push .                    # Build to bin/push
go build -o bin/pushover .                # Build to bin/pushover
```

### Testing Commands
```bash
./run_test.sh                             # Run all tests with verbose output
go test -v ./...                          # Direct test command
go test -v -run TestFunctionName          # Run specific test
go test -v -run TestLoadConfig            # Example: test config loading
go test -v -run TestRun                   # Example: test main orchestration
```

### Code Quality
```bash
./run_format.sh                           # Format code with gofmt
./run_lint.sh                             # Run golangci-lint with fixes
```

### Development Usage Examples
```bash
# Basic usage
./bin/push "Hello World"

# With flags
./bin/push -m "Message" -t "Title" -p 1 -s "magic"

# With environment variables
APP_KEY=xxx RECIPIENT_KEY=yyy ./bin/push "Test message"
```

## Environment Configuration

### Required Variables
- **`APP_KEY`**: Pushover application key (required)
- **`RECIPIENT_KEY`**: Pushover recipient key (required)

### Optional Variables (with defaults)
- **`DEVICE_NAME`**: Target device name
- **`DEFAULT_TITLE`**: Default message title
- **`PUSHOVER_PRIORITY`**: Default priority (-2 to 2, default: 0)
- **`PUSHOVER_SOUND`**: Default sound (default: "vibrate")
- **`PUSHOVER_EXPIRE`**: Default expiration for emergency messages (default: 180 seconds)

### Configuration Loading
- Automatic `.env` file loading via `godotenv/autoload` import
- CLI arguments override environment variables
- Sensible defaults for optional parameters

## Dependencies & External APIs

### Core Dependencies
- **`github.com/gregdel/pushover v1.3.1`**: Official Pushover Go client
  - Complete Pushover API coverage
  - Priority constants, sound definitions, error types
  - Supports attachments, emergency notifications, glances

- **`github.com/joho/godotenv v1.5.1`**: Environment variable loading
  - Automatic `.env` file loading via `autoload` import
  - Development-friendly environment management

- **`github.com/stretchr/testify v1.9.0`**: Testing framework
  - Mock generation and verification
  - Rich assertion library
  - Test suite organization

### Pushover API Features Supported
- **Priority Levels**: -2 (silent) to 2 (emergency with retry)
- **Sound Options**: 23 different notification sounds
- **Device Targeting**: Send to specific registered devices
- **Emergency Messages**: Retry until acknowledged with custom expiration
- **Timestamps**: Automatic message timestamping

## Testing Strategy & Patterns

### Testing Architecture
- **Mock-Based Testing**: `MockPushoverClient` with controllable behavior
- **Dependency Injection**: `Run()` function accepts client interface for testing
- **Table-Driven Tests**: Systematic testing with comprehensive input combinations
- **Integration Testing**: End-to-end testing through main application flow

### Test Coverage Areas
1. **Configuration Loading**: Environment variables, defaults, validation
2. **CLI Argument Parsing**: Flags, positional args, help/version commands
3. **Client Creation**: API key validation, error handling
4. **Message Construction**: Priority handling, sound selection, device targeting
5. **API Communication**: Success scenarios, error simulation, retry logic
6. **Integration Flow**: Complete application execution with various configurations

### Testing Best Practices
- Environment variable cleanup after tests
- Mock verification for API calls
- Error condition simulation
- Backwards compatibility validation
- Input validation testing

## File Structure & Key Locations

### Core Files
- **`main.go`**: Complete application logic (single file architecture)
- **`main_test.go`**: Comprehensive test suite with mock implementations
- **`go.mod`**: Dependency management and Go version requirements
- **`CLAUDE.md`**: This development guide

### Build & Development
- **`Makefile`**: Complete build system with all development targets
- **`run_test.sh`**: Test execution script with verbose output
- **`run_format.sh`**: Code formatting automation
- **`run_lint.sh`**: Linting with automatic fixes

### Binary Output
- **`bin/`**: Build output directory
- **`bin/push`**: Primary binary name
- **`bin/pushover`**: Alternative binary name

## Architecture Insights for Development

### When Adding Features
1. **Maintain Interface Contract**: Ensure `PushoverClient` interface remains stable
2. **Add Corresponding Tests**: Every new feature requires mock testing
3. **Update CLI Help**: Document new flags and options
4. **Environment Variable Support**: Consider environment variable equivalents
5. **Error Handling**: Follow existing error propagation patterns

### When Modifying Configuration
1. **Preserve Backwards Compatibility**: Maintain existing environment variable names
2. **Add Validation**: Ensure new config options have proper validation
3. **Update Documentation**: Document new variables and their defaults
4. **Test Override Behavior**: Verify CLI arguments properly override environment

### When Debugging Issues
1. **Check Environment Loading**: Verify `.env` file format and variable names
2. **Validate API Keys**: Ensure APP_KEY and RECIPIENT_KEY are correct
3. **Test with Mock**: Use mock client to isolate API vs. application issues
4. **Check Priority Values**: Verify priority is within valid range (-2 to 2)
5. **Validate Device Names**: Ensure device names match registered devices

## MCP Server Capabilities

### Overview
The Pushover application now includes **Model Context Protocol (MCP) server capabilities**, allowing it to serve as an MCP server that AI systems can connect to for sending push notifications. This functionality follows the TimeMCP reference architecture with authentication, HTTP transport, and comprehensive configuration management.

### MCP Server Architecture

#### **Core Components**
- **`main.go`**: Enhanced with MCP server logic and tool definitions
- **`config.go`**: MCP-specific configuration management with environment variables
- **`auth.go`**: JWT-based authentication middleware for HTTP transport
- **`http_server.go`**: HTTP transport server with health endpoints and CORS support

#### **Transport Methods**

**STDIO Transport** (Default):
```bash
./bin/pushover mcp
./bin/pushover mcp -transport stdio
```
- Best for command-line integrations and AI systems
- Uses standard input/output for JSON-RPC communication
- Automatic signal handling for graceful shutdown

**HTTP Transport**:
```bash
./bin/pushover mcp -transport http
```
- RESTful API with optional JWT authentication
- Health endpoints: `/health`, `/capabilities`, `/generate-token`
- CORS support for web integrations
- Configurable address, path, and middleware

#### **Available Tools**

**`send_notification`**: Send push notifications via Pushover
- **Parameters**:
  - `message` (required): Notification message text (max 1024 chars)
  - `title` (optional): Notification title
  - `priority` (optional): Priority level (-2 to 2)
  - `device` (optional): Target device name
  - `sound` (optional): Notification sound name
  - `expire` (optional): Expiration time for emergency messages

### MCP Configuration

#### **Environment Variables**

**HTTP Transport Settings**:
```bash
PUSHOVER_HTTP_ADDRESS=":8080"          # Server bind address
PUSHOVER_HTTP_PATH="/mcp"              # Endpoint path
PUSHOVER_HTTP_STATELESS="false"        # Session management mode
PUSHOVER_HTTP_HEARTBEAT="30s"          # Heartbeat interval
PUSHOVER_HTTP_TIMEOUT="30s"            # Request timeout
PUSHOVER_HTTP_CORS_ENABLED="true"      # Enable CORS
PUSHOVER_HTTP_CORS_ORIGINS="*"         # Allowed origins (comma-separated)
```

**Authentication Settings**:
```bash
PUSHOVER_AUTH_ENABLED="false"          # Enable JWT authentication
PUSHOVER_AUTH_SECRET_KEY="secret"      # JWT signing secret
```

**Pushover API Settings** (same as CLI mode):
```bash
APP_KEY="your_app_key"                 # Required
RECIPIENT_KEY="your_recipient_key"     # Required
DEVICE_NAME="optional_device"          # Optional
DEFAULT_TITLE="default_title"          # Optional
PUSHOVER_PRIORITY="0"                  # Default priority
PUSHOVER_SOUND="vibrate"               # Default sound
PUSHOVER_EXPIRE="180"                  # Default expiration
```

### MCP Server Usage

#### **Command-Line Interface**

**Main Help** (shows both CLI and MCP modes):
```bash
./bin/pushover -h
Usage: ./bin/pushover [OPTIONS] | ./bin/pushover SUBCOMMAND [OPTIONS]

Send push notifications via Pushover

Subcommands:
  mcp               Start MCP server mode

Options:
  [CLI options for notification sending]
```

**MCP Subcommand Help**:
```bash
./bin/pushover mcp -h
Usage: ./bin/pushover mcp [OPTIONS]

Start MCP server for push notifications

Options:
  -auth-enabled
        Enable JWT authentication for HTTP transport
  -generate-token
        Generate a JWT token and exit
  -token-expiration int
        Token expiration in hours (default: 744 = 31 days) (default 744)
  -token-role string
        Role for token generation (default "admin")
  -token-user-id string
        User ID for token generation (default "user1")
  -token-username string
        Username for token generation (default "admin")
  -transport string
        Transport mode: 'stdio' (default) or 'http' (default "stdio")
```

#### **Starting the Server**

**STDIO Mode** (for AI systems):
```bash
./bin/pushover mcp
./bin/pushover mcp -transport stdio
```

**HTTP Mode** (for web integrations):
```bash
./bin/pushover mcp -transport http
```

**HTTP Mode with Authentication**:
```bash
./bin/pushover mcp -transport http -auth-enabled
```

#### **Authentication (HTTP Mode)**

**1. Generate Token via Command Line**:
```bash
# Set secret key and generate token
PUSHOVER_AUTH_SECRET_KEY="your-secret-key" \
./bin/pushover mcp -generate-token

# Generate custom token
PUSHOVER_AUTH_SECRET_KEY="your-secret-key" \
./bin/pushover mcp -generate-token \
  -token-user-id="test_user" \
  -token-username="Test User" \
  -token-role="user" \
  -token-expiration=24
```

**2. Use Token in HTTP Requests**:
```bash
Authorization: Bearer <jwt_token>
```

**3. Alternative: Generate Token via HTTP API**:
```bash
curl -X POST http://localhost:8080/generate-token \
  -H "Content-Type: application/json" \
  -d '{"user_id": "test_user", "username": "Test User", "expires_in": 24}'
```

#### **Health and Capabilities**

**Health Check**:
```bash
curl http://localhost:8080/health
```

**Capabilities Discovery**:
```bash
curl http://localhost:8080/capabilities
```

### Integration Examples

#### **MCP Client Integration**

```json
{
  "mcpServers": {
    "pushover": {
      "command": "/path/to/pushover",
      "args": ["mcp", "-transport", "stdio"],
      "env": {
        "APP_KEY": "your_app_key",
        "RECIPIENT_KEY": "your_recipient_key"
      }
    }
  }
}
```

#### **HTTP API Integration**

```bash
# Send notification via HTTP
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt_token>" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "send_notification",
      "arguments": {
        "message": "Hello from MCP!",
        "title": "Test Notification",
        "priority": "1"
      }
    }
  }'
```

### Development and Testing

#### **Testing MCP Functionality**

**STDIO Mode Testing**:
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./bin/pushover mcp -transport stdio
```

**HTTP Mode Testing**:
```bash
# Start server in background
./bin/pushover mcp -transport http &

# Test capabilities
curl http://localhost:8080/capabilities

# Test notification
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"send_notification","arguments":{"message":"Test"}}}'
```

**Token Generation Testing**:
```bash
# Generate token and test authentication
PUSHOVER_AUTH_SECRET_KEY="test_secret" ./bin/pushover mcp -generate-token
```

#### **Configuration Testing**

```bash
# Test with environment variables and flags
PUSHOVER_HTTP_ADDRESS=":9090" \
PUSHOVER_HTTP_PATH="/api/mcp" \
PUSHOVER_AUTH_SECRET_KEY="test_secret" \
./bin/pushover mcp -transport http -auth-enabled
```

### Security Considerations

1. **JWT Authentication**: Use strong secret keys in production
2. **CORS Configuration**: Restrict origins in production environments
3. **HTTPS**: Use reverse proxy with SSL/TLS for production HTTP deployments
4. **Environment Variables**: Secure storage of API keys and secrets
5. **Input Validation**: All parameters are validated before processing

### Compatibility

- **Backwards Compatible**: Existing CLI functionality remains unchanged
- **Subcommand Structure**: MCP functionality accessed via `mcp` subcommand for clear separation
- **Dual Mode**: Same binary supports both CLI and MCP server modes
- **Configuration**: Environment variables work for both modes
- **Testing**: Existing test suite covers shared functionality

### Architecture Benefits

1. **Clean Separation**: MCP logic isolated from CLI functionality
2. **Authentication Ready**: JWT middleware for secure HTTP access
3. **Extensible**: Easy to add new tools following existing patterns
4. **Production Ready**: Health checks, CORS, graceful shutdown
5. **Well-Documented**: Comprehensive configuration and usage examples

## Claude Code Usage Guidelines

### Documentation Reference Patterns

When working with this codebase, Claude should follow these documentation usage patterns:

#### **Code References**
- Always include `file_path:line_number` when referencing specific functions or code sections
- Example: "The `CreateMessage()` function in main.go:245 handles message construction"
- Use exact function names and locations from the codebase

#### **Context Utilization**
- Reference this CLAUDE.md document for:
  - Architecture decisions and design patterns
  - Available commands and build processes
  - Environment configuration requirements
  - Testing strategies and patterns
  - MCP server capabilities and usage

#### **Task Planning**
- For complex changes, always reference the "Architecture Insights for Development" section
- Follow the established patterns for adding features, modifying configuration, or debugging
- Maintain consistency with the single-package architecture and interface-based design

#### **Command Execution Priority**
1. **First**: Use project-specific scripts (`./run_test.sh`, `./run_lint.sh`, `./run_format.sh`)
2. **Second**: Use Makefile targets (`make test`, `make lint`, `make build`)
3. **Third**: Use direct Go commands (`go test`, `go build`)

#### **Configuration Management**
- Always check environment variables section before suggesting new configuration
- Reference the MCP configuration section for server-related tasks
- Maintain backwards compatibility as outlined in the development guidelines

#### **Error Handling**
- Follow the established error propagation patterns (no `log.Fatal`)
- Reference the "Testing Strategy & Patterns" section for error condition testing
- Use the mock client patterns for testing error scenarios

### Efficient Documentation Usage

#### **Before Making Changes**
1. Review relevant sections of this documentation
2. Check existing patterns in the codebase
3. Verify environment variable requirements
4. Confirm testing approach

#### **During Development**
- Reference component descriptions for understanding existing functionality
- Use the dependency information for import decisions
- Follow the established coding conventions from the GOLANG.md guidelines

#### **After Implementation**
- Verify changes align with architecture principles
- Run appropriate commands based on the "Development Commands" section
- Confirm testing follows the established patterns

### Documentation Maintenance

When suggesting improvements to this documentation:
- Maintain the existing structure and formatting
- Add specific examples for new features
- Update command references when build processes change
- Keep the architecture insights current with code changes
