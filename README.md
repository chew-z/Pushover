# Pushover

A robust command-line utility for sending notifications via Pushover with extensive configuration options.

## Installation

### Build from source
```bash
go build -o bin/pushover .
```

### Install the binary
```bash
# Copy the binary to your PATH
cp bin/pushover /usr/local/bin/pushover
```

## Usage

### Basic Usage
```bash
# Using flags
pushover -m "Your task has completed" -t "Task Notification"

# Using positional arguments (backwards compatible)
pushover "Your task has completed" "Task Notification"
```

### Advanced Usage
```bash
# Set high priority and custom sound
pushover -m "Critical alert" -t "System Alert" -p 1 -s siren

# Send to specific device with custom expire time
pushover -m "Build completed" -d "iPhone" -e 3600

# Show help
pushover -h

# Show version
pushover -version
```

## Configuration

### Environment Variables

Required:
- `APP_KEY`: Your Pushover application key
- `RECIPIENT_KEY`: Your Pushover recipient key

Optional:
- `DEVICE_NAME`: Default device to send to
- `DEFAULT_TITLE`: Default message title when none specified
- `PUSHOVER_PRIORITY`: Default priority (-2 to 2)
- `PUSHOVER_SOUND`: Default sound name
- `PUSHOVER_EXPIRE`: Default expire time in seconds

### Command Line Options

- `-m, -message`: Message to send (required)
- `-t, -title`: Message title
- `-p, -priority`: Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)
- `-s, -sound`: Sound name (e.g., pushover, bike, bugle, cashregister, classical, cosmic, falling, gamelan, incoming, intermission, magic, mechanical, pianobar, siren, spacealarm, tugboat, alien, climb, persistent, echo, updown, vibrate, none)
- `-e, -expire`: Expire time in seconds
- `-d, -device`: Device name
- `-h, -help`: Show help
- `-version`: Show version

### Priority Levels
- `-2`: Lowest priority (no notification)
- `-1`: Low priority (quiet notification)
- `0`: Normal priority (default)
- `1`: High priority (bypass quiet hours)
- `2`: Emergency priority (requires acknowledgment)

### Configuration File

You can also use a `.env` file to store configuration values:

```env
APP_KEY=your_app_key_here
RECIPIENT_KEY=your_recipient_key_here
DEVICE_NAME=iPhone
DEFAULT_TITLE=Notification
PUSHOVER_PRIORITY=0
PUSHOVER_SOUND=pushover
PUSHOVER_EXPIRE=180
```

### Getting Pushover Keys

1. Create an account at [pushover.net](https://pushover.net)
2. Register an application to get your `APP_KEY`
3. Find your `RECIPIENT_KEY` in your user dashboard

## Development

### Build
```bash
go build -o bin/pushover .
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
