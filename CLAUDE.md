# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a simple Go command-line utility for sending notifications via Pushover. The application is structured as a single-package CLI tool that wraps the Pushover API with configuration management and comprehensive testing.

## Architecture

- **Single Package Design**: All code is in the `main` package with clear separation of concerns through functions
- **Interface-based Testing**: Uses a `PushoverClient` interface to enable mocking in tests
- **Configuration Management**: Environment-based configuration with `.env` file support via `godotenv/autoload`
- **Error Handling**: Proper error propagation without using `log.Fatal` (follows project conventions)

## Key Components

- `Config` struct: Holds application configuration loaded from environment variables
- `PushoverClient` interface: Abstracts the Pushover API for testability
- `RealPushoverApp`: Production implementation of PushoverClient
- `MockPushoverClient`: Test implementation using testify/mock

## Common Development Commands

### Build
```bash
go build -o bin/push .
```

### Testing
```bash
./run_test.sh          # Run all tests with verbose output
go test -v ./...       # Direct test command
```

### Code Quality
```bash
./run_format.sh        # Format code with gofmt
./run_lint.sh          # Run golangci-lint with fixes
```

### Single Test Execution
```bash
go test -v -run TestFunctionName
```

## Environment Configuration

The application uses these environment variables:
- `APP_KEY`: Pushover application key
- `RECIPIENT_KEY`: Pushover recipient key  
- `DEVICE_NAME`: Target device (optional)

Configuration is loaded automatically via `godotenv/autoload` from a `.env` file.

## Dependencies

- `github.com/gregdel/pushover`: Official Pushover Go client
- `github.com/joho/godotenv`: Environment variable loading
- `github.com/stretchr/testify`: Testing framework with mocking

## Testing Strategy

The codebase follows a comprehensive testing approach:
- Unit tests for each function with table-driven tests
- Integration tests combining multiple components
- Mock-based testing for external API calls
- Environment variable testing with proper cleanup