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
- **Testing**: Use `make test` to run the test suite. Tests are table-driven and use mock objects. MCP schema validation is tested end-to-end via `client.NewInProcessClient` in `TestMCPSchemaValidation` (requires `Start` → `Initialize` → `CallTool` sequence).
- **Linting**: Use `make lint` or `./run_lint.sh` to run `golangci-lint`. Config is in `.golangci.yml` (v2 format); pre-existing `errcheck` patterns for stdout writes and JSON encoding are excluded there.
- **Formatting**: Use `make fmt` to run `gofmt`.
- **Error Handling**: Propagate errors; do not use `log.Fatal`.

## Current Focus Areas
- The MCP server functionality is a key feature.
- The authentication system was recently migrated to the standard `github.com/golang-jwt/jwt/v5` library.
- MCP tool schema validation is enabled via `server.WithInputSchemaValidation()`: `priority` and `expire` are integer parameters with `Min`/`Max` constraints; `message` enforces `MaxLength(1024)` (character-based). Handlers no longer perform manual range checks or `strconv` parsing for these fields.

## Key Dependencies & Integrations
- **`github.com/gregdel/pushover`**: The official Pushover Go client.
- **`github.com/joho/godotenv`**: For loading `.env` files.
- **`github.com/stretchr/testify`**: For testing.
- **`github.com/golang-jwt/jwt/v5`**: For JWT-based authentication.
- **`github.com/mark3labs/mcp-go`**: For the Model Context Protocol server.

## AI Assistant Guidelines
- **Use Standard Libraries**: Prefer standard libraries like `github.com/golang-jwt/jwt/v5` over custom implementations.
- **Follow Existing Patterns**: Mimic the existing code style, including the use of interfaces, dependency injection, and table-driven tests.
- **Update Documentation**: When adding new features, update this `CLAUDE.md` file accordingly.
- **Keep it Simple**: Avoid over-engineering solutions. The recent auth refactoring and schema validation migration are good examples of this principle.
- **MCP Schema vs Handler Validation**: Put input constraints (type, range, length) in the tool schema using `mcp.WithInteger`/`mcp.MaxLength`/`mcp.Min`/`mcp.Max`. Do not duplicate those checks in handler code.

## Other instructions

@./CODANNA.md
@./GOLANG.md
@./USING-GODOC.md