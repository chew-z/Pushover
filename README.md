# Pushover

A simple command-line utility for sending notifications via Pushover.

## Installation

### Build from source
```bash
go build -o bin/push .
```

### Install the binary
```bash
# Copy the binary to your PATH
cp bin/push /usr/local/bin/push
```

## Usage

```bash
push <message> [<title>]
```

Example:
```bash
push "Your task has completed" "Task Notification"
```

## Configuration

The application can be configured using environment variables:

- `APP_KEY`: Your Pushover application key (required)
- `RECIPIENT_KEY`: Your Pushover recipient key (required)
- `DEVICE_NAME`: The device to send to (optional, leave empty for all devices)

You can also use a `.env` file to store these values. The application will automatically load them.

### Getting Pushover Keys

1. Create an account at [pushover.net](https://pushover.net)
2. Register an application to get your `APP_KEY`
3. Find your `RECIPIENT_KEY` in your user dashboard

## Development

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

## Features

- Send notifications with custom messages and titles
- Target specific devices 
- Configure notification priority, expiration, and sound
- Uses environment variables for configuration
- Comprehensive test coverage with mocking
- Clean architecture with interface-based design
