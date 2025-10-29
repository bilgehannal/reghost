# reghost

A lightweight DNS server daemon with dynamic configuration management for local development. reghost allows you to route custom domains to specific IP addresses with regex pattern support and hot configuration reloading.

## Features

- 🚀 **Dynamic DNS Resolution**: Route domains to specific IPs using exact matches or regex patterns
- 🔄 **Hot Reload**: Automatically reloads configuration when the config file changes
- 🎯 **Multiple Record Sets**: Define multiple record sets and switch between them easily
- 🔐 **Safe Configuration Updates**: Atomic file writes prevent corruption
- 📝 **Comprehensive Logging**: Rotating logs with size and time-based retention
- 🛠️ **CLI Management**: Easy-to-use CLI tool for configuration management
- 🎲 **Smart IP Binding**: Automatically binds to random loopback IP (127.0.0.0/8) with safe cleanup
- 🔧 **Automatic DNS Setup**: Configures system DNS resolver on startup and cleans up on exit
- ⚡ **Lightweight**: Minimal resource usage, written in Go

## Installation

### Quick Install (Recommended)

Install the latest release with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/bilgehannal/reghost/main/install.sh | sudo sh
```

Or manually download from [releases](https://github.com/bilgehannal/reghost/releases/latest):

```bash
# macOS ARM64 (Apple Silicon)
curl -L https://github.com/bilgehannal/reghost/releases/latest/download/reghost-darwin-arm64.tar.gz | sudo tar -xz -C /usr/local/bin && sudo chmod 6755 /usr/local/bin/reghost{d,ctl} && sudo chgrp admin /usr/local/bin/reghost{d,ctl}

# macOS AMD64 (Intel)
curl -L https://github.com/bilgehannal/reghost/releases/latest/download/reghost-darwin-amd64.tar.gz | sudo tar -xz -C /usr/local/bin && sudo chmod 6755 /usr/local/bin/reghost{d,ctl} && sudo chgrp admin /usr/local/bin/reghost{d,ctl}

# Linux AMD64
curl -L https://github.com/bilgehannal/reghost/releases/latest/download/reghost-linux-amd64.tar.gz | sudo tar -xz -C /usr/local/bin && sudo chmod 6755 /usr/local/bin/reghost{d,ctl} && sudo chgrp root /usr/local/bin/reghost{d,ctl}

# Linux ARM64
curl -L https://github.com/bilgehannal/reghost/releases/latest/download/reghost-linux-arm64.tar.gz | sudo tar -xz -C /usr/local/bin && sudo chmod 6755 /usr/local/bin/reghost{d,ctl} && sudo chgrp root /usr/local/bin/reghost{d,ctl}
```

### Prerequisites

- Root/sudo access (required for DNS server on port 53)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/bilgehannal/reghost.git
cd reghost

# Download dependencies
make deps

# Build binaries
make build

# Install (requires sudo)
make install
```

This will:
- Build `reghostd` (daemon) and `reghostctl` (CLI tool)
- Install binaries to `/usr/local/bin`
- Set appropriate permissions (setuid for daemon)

## Quick Start

### 1. Start the Daemon

```bash
# The daemon needs to run as root
reghostd
```

The daemon will:
- Create a default config at `/etc/reghost.yml` if it doesn't exist
- Bind to a random loopback IP in 127.0.0.0/8 range
- **Automatically configure system DNS resolver** (macOS: `/etc/resolver/reghost`, Linux: `/etc/resolv.conf`)
- Start listening on port 53 (UDP and TCP)
- Watch the config file for changes
- **Automatically cleanup DNS configuration on exit** (SIGTERM/SIGINT)

### 2. Configure DNS Records

```bash
# View current configuration
reghostctl show

# Add a new record to the default set
reghostctl add-record default \
  --domain "myapp.local" \
  --ip "192.168.1.100"

# Add a regex pattern
reghostctl add-record default \
  --domain "^[a-z]+\.dev\.$" \
  --ip "10.0.0.1"

# Create a new record set
reghostctl create-set production

# Add records to the new set
reghostctl add-record production \
  --domain "api.myapp.com" \
  --ip "10.113.241.216"

# Switch active record set
reghostctl set-active production

# List all record sets
reghostctl list
```

## Configuration

### Configuration File

Location: `/etc/reghost.yml`

Example configuration:

```yaml
activeRecord: record1
records:
  record1:
    - domain: '^[a-zA-Z0-9-]+\.myhost\.$'
      ip: 10.113.241.216
    - domain: 'myhost'
      ip: 10.113.241.216
  record2:
    - domain: '^[a-zA-Z0-9-]+\.myhost\.$'
      ip: 10.113.241.217
  production:
    - domain: 'api.example.com'
      ip: 192.168.1.100
    - domain: 'web.example.com'
      ip: 192.168.1.101
```

### Domain Patterns

- **Exact Match**: `myapp.local` - matches exactly "myapp.local"
- **Regex Pattern**: `^[a-z]+\.dev\.$` - matches any lowercase letters followed by .dev.

### Default Configuration

If no config file exists, reghost creates a default configuration:

```yaml
activeRecord: default
records:
  default:
    - domain: 'reghost.local'
      ip: 127.0.0.1
```

## CLI Commands

### Show Configuration

```bash
reghostctl show
```

### List Record Sets

```bash
reghostctl list
```

### Set Active Record Set

```bash
reghostctl set-active <record-set-name>
```

### Add Record

```bash
reghostctl add-record <record-set> \
  --domain "example.local" \
  --ip "192.168.1.1"
```

### Remove Record

```bash
# Remove record at index 0 from the set
reghostctl remove-record <record-set> --index 0
```

### Create Record Set

```bash
reghostctl create-set <record-set-name>
```

### Delete Record Set

```bash
# Note: Cannot delete the active record set
reghostctl delete-set <record-set-name>
```

## System DNS Configuration

The daemon **automatically configures** your system's DNS resolver:

- **macOS**: Creates `/etc/resolver/reghost` to route `*.reghost` domains
- **Linux**: Adds nameserver to `/etc/resolv.conf` as first entry

On daemon shutdown (Ctrl+C or SIGTERM), the configuration is **automatically cleaned up**.

See [DNS_CONFIGURATION.md](DNS_CONFIGURATION.md) for detailed information.

## Logging

Logs are written to `/var/log/reghost.log` with automatic rotation:

- **Max Size**: 5MB per file
- **Retention**: 7 days
- **Max Files**: 7 backup files

Log format:
```
[2025-10-29 12:34:56] [INFO] DNS Query: myapp.local. (type: A)
[2025-10-29 12:34:56] [INFO] Match found: myapp.local. -> 192.168.1.100
```

## Architecture

### Project Structure

```
reghost/
├── cmd/
│   ├── reghostd/          # Daemon entrypoint
│   └── reghostctl/        # CLI entrypoint
├── internal/
│   ├── config/            # Configuration management
│   ├── dns/               # DNS server implementation
│   ├── watcher/           # File watcher for hot reload
│   ├── cli/               # CLI commands
│   └── utils/             # Utilities (logger, etc.)
├── pkg/
│   └── reghost/           # Core resolver logic
├── test/                  # Tests
├── Makefile
└── README.md
```

### How It Works

1. **Daemon Startup**: 
   - Loads configuration from `/etc/reghost.yml`
   - Binds to a random IP in 127.0.0.0/8 range
   - Starts DNS servers (UDP/TCP on port 53)
   - Begins watching config file

2. **DNS Resolution**:
   - Receives DNS query
   - Looks up domain in active record set
   - Returns IP if matched (exact or regex)
   - Returns NXDOMAIN if no match

3. **Hot Reload**:
   - File watcher detects config changes
   - Reloads and validates configuration
   - Updates in-memory cache
   - No daemon restart needed

4. **Signal Handling**:
   - Gracefully handles SIGTERM/SIGINT
   - Cleans up system DNS resolver configuration
   - Releases loopback IP alias
   - Cleanly shuts down DNS servers

## Development

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Linting

```bash
# Install golangci-lint
brew install golangci-lint

# Run linter
make lint
```

### Building

```bash
# Build both daemon and CLI
make build

# Build daemon only
make build-daemon

# Build CLI only
make build-cli
```

## System Requirements

- **OS**: macOS or Linux
- **Privileges**: Root access required for:
  - Binding to port 53
  - Managing loopback IP aliases
  - Writing to /etc and /var/log

## Troubleshooting

### Daemon won't start

```bash
# Check if running as root
reghostd

# Check if port 53 is already in use
sudo lsof -i :53

# Check logs
sudo tail -f /var/log/reghost.log
```

### DNS not resolving

```bash
# Test DNS directly
dig @127.x.x.x -p 53 myapp.local

# Check active record set
reghostctl show

# Verify domain is in active record set
reghostctl list
```

### Config not reloading

```bash
# Check file watcher
sudo tail -f /var/log/reghost.log

# Manually validate config
reghostctl show
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Author

Bilgehan Nal ([@bilgehannal](https://github.com/bilgehannal))

## Related Projects

- [miekg/dns](https://github.com/miekg/dns) - DNS library in Go
- [fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) - File system notifications
- [spf13/cobra](https://github.com/spf13/cobra) - CLI framework

## Changelog

### v1.0.0 (2025-10-29)
- Initial release
- Dynamic DNS resolution with regex support
- Hot configuration reloading
- CLI management tool
- Rotating logs
- Safe loopback IP management