# System DNS Configuration

The `reghostd` daemon automatically configures your system's DNS resolver when it starts and cleans it up when it exits.

## How It Works

### macOS

On macOS, reghost creates a resolver configuration file:
- **File**: `/etc/resolver/reghost`
- **Effect**: DNS queries are sent to the reghost DNS server, which matches them against your configured regex patterns
- **Cleanup**: File is automatically removed when daemon exits

Example configuration created:
```
nameserver 127.x.x.x
port 53
```

### Linux

On Linux, reghost modifies the system resolver configuration:
- **File**: `/etc/resolv.conf`
- **Effect**: Adds reghost DNS server as the first nameserver
- **Cleanup**: Original configuration is automatically restored when daemon exits

## Automatic Behavior

When you start `reghostd`:
```bash
sudo ./bin/reghostd
```

The daemon will:
1. Find and bind to a random loopback IP (127.0.0.0/8)
2. **Automatically configure system DNS resolver**
3. Start DNS server on port 53
4. Log the configuration changes

Example log output:
```
[2025-10-29 12:34:56] [INFO] DNS server bound to: 127.0.0.5:53
[2025-10-29 12:34:56] [INFO] Configuring macOS resolver...
[2025-10-29 12:34:56] [INFO] ✓ Created /etc/resolver/reghost
[2025-10-29 12:34:56] [INFO] ✓ DNS cache flushed
[2025-10-29 12:34:56] [INFO] DNS resolver configured - queries will be matched against your regex patterns
```

When you stop the daemon (Ctrl+C or SIGTERM):
```
[2025-10-29 12:35:30] [INFO] Received signal: interrupt
[2025-10-29 12:35:30] [INFO] Shutting down gracefully...
[2025-10-29 12:35:30] [INFO] Removing macOS resolver config: /etc/resolver/reghost
[2025-10-29 12:35:30] [INFO] ✓ Removed /etc/resolver/reghost
[2025-10-29 12:35:30] [INFO] Released loopback alias: 127.0.0.5
```

## Testing DNS Resolution

### macOS

After starting the daemon, test with domains matching your configured regex patterns:

```bash
# Example: if your config has domain: '^[a-zA-Z0-9-]+\.myhost\.$'
dig x.myhost
ping test.myhost

# Using curl (if you have a record configured)
curl http://web.myhost
```

### Linux

After starting the daemon, test with any domain:

```bash
# Using dig
dig example.com

# Using nslookup
nslookup myapp.local

# Check resolv.conf
cat /etc/resolv.conf
```

## Manual Configuration (Alternative)

If automatic configuration fails or you prefer manual setup, you can use the `setup-resolver` tool:

```bash
# Get the bind IP from logs
tail /var/log/reghost.log | grep "bound to"

# Configure manually
sudo ./bin/setup-resolver 127.x.x.x reghost
```

## Cleanup on Crash

If the daemon crashes or is killed with `SIGKILL` (kill -9), the configuration may not be cleaned up automatically. To manually clean up:

### macOS
```bash
sudo rm -f /etc/resolver/reghost
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder
```

### Linux
```bash
# Remove the nameserver entry from /etc/resolv.conf
sudo nano /etc/resolv.conf
# Or restore from backup if you made one
sudo cp /etc/resolv.conf.backup /etc/resolv.conf
```

## Important Notes

1. **Root Required**: System DNS configuration requires root privileges
2. **Graceful Shutdown**: Always use Ctrl+C or `sudo pkill reghostd` (SIGTERM) for proper cleanup
3. **Regex Matching**: The DNS server matches incoming queries against the regex patterns defined in your configuration file
4. **DNS Cache**: The daemon automatically flushes DNS caches when configuring/cleaning up
5. **systemd-resolved**: On Linux systems using systemd-resolved, you may need to stop the service first:
   ```bash
   sudo systemctl stop systemd-resolved
   ```

## Troubleshooting

### Configuration Not Working

Check logs:
```bash
sudo tail -f /var/log/reghost.log
```

Look for messages like:
- "Failed to configure system resolver" - Check permissions
- "Configuring macOS resolver" or "Configuring Linux resolver" - Success

### Manual Verification

**macOS:**
```bash
ls -la /etc/resolver/
cat /etc/resolver/reghost
```

**Linux:**
```bash
head -5 /etc/resolv.conf
```

### DNS Not Resolving

1. Verify daemon is running: `ps aux | grep reghostd`
2. Check bind IP: `sudo lsof -i :53`
3. Test directly: `dig @127.x.x.x -p 53 test.reghost`
4. Check configuration exists: See "Manual Verification" above
