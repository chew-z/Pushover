# Pushover

A simple command-line utility for sending notifications via Pushover.

## Installation

```bash
go get -u github.com/yourusername/Pushover
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

- `APP_KEY`: Your Pushover application key
- `RECIPIENT_KEY`: Your Pushover recipient key
- `DEVICE_NAME`: The device to send to (leave empty for all devices)

You can also use a `.env` file to store these values. The application will automatically load them.

## Testing

Run the tests with:

```bash
go test ./...
```

## Features

- Send notifications with custom messages and titles
- Target specific devices 
- Configure notification priority, expiration, and sound
- Uses environment variables for configuration
