# Project Context: Pushover Notification Service

## Overview
A Go-based command-line utility and MCP server for sending Pushover notifications. The project emphasizes testability, modularity, and robust error handling.

## Architecture
- **Single-Package Design**: All Go files are in the `main` package.
- **Interface-Based Abstraction**: `PushoverClient` interface for mocking and testing.
- **Dependency Injection**: `Run()` function accepts the `PushoverClient` interface.
- **Configuration**: Loaded from environment variables (`.env` file) and overridden by CLI flags.
- **Authentication**: JWT-based authentication for the MCP server's HTTP transport, using the `github.com/golang-jwt/jwt/v5` library.

## Development Guidelines
- **Testing**: Use `make test` to run the test suite. Tests are table-driven and use mock objects.
- **Linting**: Use `make lint` to run `golangci-lint`.
- **Formatting**: Use `make fmt` to run `gofmt`.
- **Error Handling**: Propagate errors; do not use `log.Fatal`.

## Current Focus Areas
- The MCP server functionality is a key feature.
- The authentication system was recently migrated to the standard `github.com/golang-jwt/jwt/v5` library.

## Key Dependencies & Integrations
- **`github.com/gregdel/pushover`**: The official Pushover Go client.
- **`github.com/joho/godotenv`**: For loading `.env` files.
- **`github.com/stretchr/testify`**: For testing.
- **`github.com/golang-jwt/jwt/v5`**: For JWT-based authentication.
- **`github.com/mark3labs/mcp-go`**: For the Model Context Protocol server.

## AI Assistant Guidelines
- **Use Standard Libraries**: Prefer standard libraries like `github.com/golang-jwt/jwt/v5` over custom implementations.
- **Follow Existing Patterns**: Mimic the existing code style, including the use of interfaces, dependency injection, and table-driven tests.
- **Update Documentation**: When adding new features, update this `GEMINI.md` file accordingly.
- **Keep it Simple**: Avoid over-engineering solutions. The recent auth refactoring is a good example of this principle.

## Other instructions

@./CODANNA.md
@./GOLANG.md
@./USING-GODOC.md