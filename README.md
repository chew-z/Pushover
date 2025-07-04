# Pushover

A versatile Go application that serves as both a command-line utility and an MCP (Model Context Protocol) server for sending push notifications via Pushover. Features comprehensive configuration options, JWT authentication, and dual transport modes.

## Features

### CLI Mode
- Send notifications with custom messages, titles, and priorities
- Target specific devices with custom sounds and expiration times
- Extensive command-line options with backwards compatibility
- Environment variable configuration with `.env` file support

### MCP Server Mode
- **STDIO Transport**: Perfect for AI system integrations
- **HTTP Transport**: RESTful API with CORS support
- **JWT Authentication**: Secure token-based access control
- **Health Endpoints**: Monitoring and capabilities discovery
- **Dual Mode**: Same binary supports both CLI and MCP server functionality

## Installation

### Build from Source
```bash
git clone <repository-url>
cd Pushover
go build -o bin/pushover .
```

### Using Makefile
```bash
make build    # Build with version injection
make install  # Install system-wide
make clean    # Clean build artifacts
```

## Configuration

### Required Environment Variables
```env
APP_KEY=your_pushover_app_key_here
RECIPIENT_KEY=your_pushover_recipient_key_here
```

### Optional Environment Variables
```env
# Pushover Settings
DEVICE_NAME=iPhone                    # Target device
DEFAULT_TITLE=Notification            # Default message title
PUSHOVER_PRIORITY=0                   # Priority (-2 to 2)
PUSHOVER_SOUND=vibrate                # Notification sound
PUSHOVER_EXPIRE=180                   # Expire time in seconds

# MCP HTTP Server Settings
PUSHOVER_HTTP_ADDRESS=:8080           # Server bind address
PUSHOVER_HTTP_PATH=/mcp               # Endpoint path
PUSHOVER_HTTP_STATELESS=false         # Session management
PUSHOVER_HTTP_HEARTBEAT=30s           # Heartbeat interval
PUSHOVER_HTTP_TIMEOUT=30s             # Request timeout
PUSHOVER_HTTP_CORS_ENABLED=true       # Enable CORS
PUSHOVER_HTTP_CORS_ORIGINS=*          # Allowed origins (comma-separated)

# Authentication Settings
PUSHOVER_AUTH_ENABLED=false           # Enable JWT authentication
PUSHOVER_AUTH_SECRET_KEY=your_secret  # JWT signing secret
```

### Configuration File
Create a `.env` file in your project directory:
```env
APP_KEY=your_app_key_here
RECIPIENT_KEY=your_recipient_key_here
DEVICE_NAME=iPhone
DEFAULT_TITLE=Notification
PUSHOVER_PRIORITY=0
PUSHOVER_SOUND=pushover
PUSHOVER_EXPIRE=180
```

## Usage

### Getting Help

```bash
# Main help - shows both CLI usage and MCP subcommand
./bin/pushover -h

# MCP-specific help - shows all MCP server options
./bin/pushover mcp -h
```

### CLI Mode

#### Basic Usage
```bash
# Using flags
./bin/pushover -m "Your task has completed" -t "Task Notification"

# Using positional arguments (backwards compatible)
./bin/pushover "Your task has completed" "Task Notification"
```

#### Advanced Usage
```bash
# High priority with custom sound
./bin/pushover -m "Critical alert" -t "System Alert" -p 1 -s siren

# Send to specific device with expiration
./bin/pushover -m "Build completed" -d "iPhone" -e 3600

# Emergency notification (requires acknowledgment)
./bin/pushover -m "Server down!" -t "EMERGENCY" -p 2 -e 300
```

#### Command Line Options
```bash
-m, -message    Message to send (required)
-t, -title      Message title
-p, -priority   Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)
-s, -sound      Sound name (see sounds list below)
-e, -expire     Expire time in seconds (for emergency messages)
-d, -device     Target device name
-h, -help       Show help
-version        Show version
```

#### Available Sounds
`pushover`, `bike`, `bugle`, `cashregister`, `classical`, `cosmic`, `falling`, `gamelan`, `incoming`, `intermission`, `magic`, `mechanical`, `pianobar`, `siren`, `spacealarm`, `tugboat`, `alien`, `climb`, `persistent`, `echo`, `updown`, `vibrate`, `none`

### MCP Server Mode

#### Getting Started
```bash
# Show MCP-specific help
./bin/pushover mcp -h

# Start MCP server (STDIO transport - default)
./bin/pushover mcp

# Start HTTP transport server
./bin/pushover mcp -transport http
```

#### Command Line Flags
```bash
-transport string        Transport mode: 'stdio' or 'http' (default: stdio)
-auth-enabled           Enable JWT authentication for HTTP transport
-generate-token         Generate a JWT token and exit
-token-expiration int   Token expiration in hours (default: 744 = 31 days)
-token-role string      Role for token generation (default: admin)
-token-user-id string   User ID for token generation (default: user1)
-token-username string  Username for token generation (default: admin)
```

#### STDIO Transport (Default)
Perfect for AI systems and command-line integrations:
```bash
./bin/pushover mcp
./bin/pushover mcp -transport stdio
```

#### HTTP Transport
RESTful API with optional authentication:
```bash
# Basic HTTP server
./bin/pushover mcp -transport http

# With JWT authentication
./bin/pushover mcp -transport http -auth-enabled
```

#### Authentication

**Generate Token via Command Line:**
```bash
# Basic token generation
PUSHOVER_AUTH_SECRET_KEY="your-secret-key" \
./bin/pushover mcp -generate-token

# Custom token with specific parameters
PUSHOVER_AUTH_SECRET_KEY="your-secret-key" \
./bin/pushover mcp -generate-token \
  -token-user-id="api_user" \
  -token-username="API User" \
  -token-role="admin" \
  -token-expiration=24
```

**Use Token in HTTP Requests:**
```bash
Authorization: Bearer <jwt_token>
```

**Generate Token via HTTP API:**
```bash
curl -X POST http://localhost:8080/generate-token \
  -H "Content-Type: application/json" \
  -d '{"user_id": "test_user", "username": "Test User", "expires_in": 24}'
```

#### HTTP Endpoints

**Health Check:**
```bash
curl http://localhost:8080/health
```

**Capabilities Discovery:**
```bash
curl http://localhost:8080/capabilities
```

**Send Notification:**
```bash
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

## MCP Integration

### MCP Client Configuration
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

### Available MCP Tools

#### `send_notification`
Send push notifications via Pushover

**Parameters:**
- `message` (required): Notification message text (max 1024 chars)
- `title` (optional): Notification title
- `priority` (optional): Priority level (-2 to 2)
- `device` (optional): Target device name  
- `sound` (optional): Notification sound name
- `expire` (optional): Expiration time for emergency messages

## Development

### Build System
```bash
make build      # Build binary with version injection
make test       # Run all tests
make lint       # Run golangci-lint
make fmt        # Format code
make clean      # Clean build artifacts
make install    # Install binary system-wide
```

### Testing
```bash
# Run all tests with verbose output
./run_test.sh

# Run specific tests
go test -v ./...
go test -v -run TestFunctionName
```

### Code Quality
```bash
# Format code
./run_format.sh

# Run linter with fixes
./run_lint.sh
```

### Testing MCP Functionality

**STDIO Mode Testing:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./bin/pushover mcp -transport stdio
```

**HTTP Mode Testing:**
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

## Getting Pushover Keys

1. Create an account at [pushover.net](https://pushover.net)
2. Register an application to get your `APP_KEY`
3. Find your `RECIPIENT_KEY` in your user dashboard
4. Optional: Register devices to get specific device names

## Architecture

### Design Principles
- **Single Package Architecture**: All functionality in `main` package with clear separation
- **Interface-Based Design**: `PushoverClient` interface enables comprehensive mocking
- **Dependency Injection**: Flexible testing and configuration
- **Subcommand Structure**: Clean separation between CLI and MCP modes via `mcp` subcommand
- **Dual Mode Operation**: Same binary supports CLI and MCP server modes
- **Security First**: JWT authentication and input validation

### Key Components
- **`main.go`**: Core application logic and MCP server implementation
- **`config.go`**: Configuration management for both CLI and MCP modes
- **`auth.go`**: JWT-based authentication middleware
- **`http_server.go`**: HTTP transport server with health endpoints
- **`main_test.go`**: Comprehensive test suite with mocking

## Security Considerations

1. **JWT Authentication**: Use strong secret keys in production
2. **CORS Configuration**: Restrict origins in production environments  
3. **HTTPS**: Use reverse proxy with SSL/TLS for production HTTP deployments
4. **Environment Variables**: Secure storage of API keys and secrets
5. **Input Validation**: All parameters are validated before processing

## Dependencies

- **[github.com/gregdel/pushover](https://github.com/gregdel/pushover)**: Official Pushover Go client
- **[github.com/joho/godotenv](https://github.com/joho/godotenv)**: Environment variable loading
- **[github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)**: Model Context Protocol implementation
- **[github.com/stretchr/testify](https://github.com/stretchr/testify)**: Testing framework (dev dependency)

## License

[Add your license information here]

## Contributing

[Add contributing guidelines here]