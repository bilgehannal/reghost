# Getting Started with reghost

## Quick Setup

### 1. Build the Project

```bash
cd /Users/bilgehan.nal/dev/personal/reghost
make build
```

This will create two binaries in the `bin/` directory:
- `reghostd` - The DNS server daemon
- `reghostctl` - The configuration management CLI tool

### 2. Install (Optional)

To install system-wide with proper permissions:

```bash
make install
```

This requires `sudo` and will:
- Copy binaries to `/usr/local/bin`
- Set setuid bit on `reghostd` for root privileges

### 3. Run the Daemon

```bash
# If installed system-wide
sudo reghostd

# Or run from bin directory
sudo ./bin/reghostd
```

The daemon will:
- Create `/etc/reghost.yml` with default configuration if it doesn't exist
- Bind to a random IP in 127.0.0.0/8 range
- **Automatically configure system DNS** (macOS: `/etc/resolver/reghost`, Linux: `/etc/resolv.conf`)
- Start DNS server on port 53 (UDP and TCP)
- Begin watching config file for changes
- Log to `/var/log/reghost.log`

**Note**: DNS configuration is automatically cleaned up when you stop the daemon (Ctrl+C).

### 4. Configure DNS Records

```bash
# View current configuration
sudo ./bin/reghostctl show

# Add a record to the default set
sudo ./bin/reghostctl add-record default \
  --domain "myapp.local" \
  --ip "192.168.1.100"

# Add a regex pattern (note: escape special chars in shell)
sudo ./bin/reghostctl add-record default \
  --domain "^[a-z]+\\.dev\\.$$" \
  --ip "10.0.0.1"
```

### 5. Test DNS Resolution

With automatic DNS configuration, you can query directly:

```bash
# macOS - query *.reghost domains
dig myapp.reghost
host api.reghost

# Linux - query any configured domain
dig myapp.local
nslookup test.reghost

# Or query directly with the bind IP
tail /var/log/reghost.log | grep "bound to"
dig @127.x.x.x myapp.local
```

## Example Configuration

After running, check `/etc/reghost.yml`:

```yaml
activeRecord: default
records:
  default:
    - domain: 'reghost.local'
      ip: 127.0.0.1
```

## Development Commands

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Clean build artifacts
make clean

# See all available commands
make help
```

## Stopping the Daemon

Press `Ctrl+C` or send SIGTERM:

```bash
sudo pkill reghostd
```

The daemon will gracefully:
- Clean up DNS resolver configuration
- Release the loopback IP alias
- Stop DNS servers

**Important**: Don't use `kill -9` (SIGKILL) as it prevents cleanup.

## Troubleshooting

### Permission Denied

```bash
# Make sure you're running as root
sudo ./bin/reghostd
```

### Port 53 Already in Use

```bash
# Check what's using port 53
sudo lsof -i :53

# Stop conflicting service (e.g., systemd-resolved on Linux)
sudo systemctl stop systemd-resolved
```

### Check Logs

```bash
# View real-time logs
sudo tail -f /var/log/reghost.log

# View all logs
sudo cat /var/log/reghost.log
```

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Set up multiple record sets for different environments
- Configure your system's DNS resolver to query the reghost server
- Set up as a systemd service (Linux) or launchd (macOS)
