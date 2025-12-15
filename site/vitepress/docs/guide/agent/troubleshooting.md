# Agent Troubleshooting

Common issues and solutions for the vStats agent.

## Connection Issues

### Agent can't connect to server

**Symptoms**: Server doesn't appear in dashboard, agent logs show connection errors.

**Solutions**:

1. **Verify server URL**
   ```bash
   curl http://YOUR_SERVER:3001/health
   ```
   Should return: `{"status":"ok"}`

2. **Check network connectivity**
   ```bash
   ping YOUR_SERVER
   telnet YOUR_SERVER 3001
   ```

3. **Check firewall rules**
   ```bash
   # Server side
   sudo ufw status
   sudo firewall-cmd --list-ports
   ```

4. **Verify token is correct**
   - Get fresh token from dashboard admin panel
   - Check for extra spaces or newlines in token

### Connection drops frequently

**Solutions**:

1. Check network stability
2. Review server logs for errors
3. Increase WebSocket timeout if needed

## Metric Issues

### Missing CPU metrics

**Solutions**:

1. **Linux**: Check if `/proc/stat` is readable
   ```bash
   cat /proc/stat
   ```

2. **Permission issues**: Agent may need elevated privileges

### Missing memory metrics

**Solutions**:

1. **Linux**: Check `/proc/meminfo`
   ```bash
   cat /proc/meminfo
   ```

2. **macOS**: Ensure agent has system permissions

### Missing disk metrics

**Solutions**:

1. Check if disks are mounted correctly
   ```bash
   df -h
   lsblk
   ```

2. Some virtual filesystems may not report correctly

### Missing network metrics

**Solutions**:

1. **Linux**: Check `/proc/net/dev`
   ```bash
   cat /proc/net/dev
   ```

2. Check if network interfaces are up
   ```bash
   ip link show
   ```

### GPU metrics not showing

**Requirements**:
- NVIDIA GPU
- NVIDIA drivers installed
- `nvidia-smi` in PATH

**Check**:
```bash
nvidia-smi
which nvidia-smi
```

## Service Issues

### Agent won't start

**Linux**:
```bash
# Check status
systemctl status vstats-agent

# Check logs
journalctl -u vstats-agent -n 100

# Try running manually
/opt/vstats-agent/vstats-agent --config /etc/vstats-agent/config.json
```

**macOS**:
```bash
# Check if loaded
launchctl list | grep vstats

# Check logs
tail -100 ~/.vstats-agent/agent.log

# Try running manually
~/.vstats-agent/vstats-agent
```

**Windows**:
```powershell
# Check service status
Get-Service vstats-agent

# Check event logs
Get-EventLog -LogName Application -Source vstats-agent -Newest 50

# Try running manually
& "C:\Program Files\vstats-agent\vstats-agent.exe"
```

### High resource usage

**High CPU**:
1. Check agent version (upgrade if outdated)
2. Increase collection interval
3. Disable unused metrics (GPU, Docker)

**High memory**:
1. Usually indicates a leak - restart agent
2. Report issue if persistent

### Agent crashes

1. Check system logs for crash details
2. Run agent in foreground to see error output
3. Check for permission issues
4. Verify all required system calls are allowed (container environments)

## Log Analysis

### Enable debug logging

```json
{
  "log_level": "debug"
}
```

Restart agent after changing log level.

### Log locations

| OS | Location |
|----|----------|
| Linux | `journalctl -u vstats-agent` |
| macOS | `~/.vstats-agent/agent.log` |
| Windows | Event Viewer → Application |

### Common log messages

| Message | Meaning |
|---------|---------|
| `connected to server` | Successfully connected |
| `connection refused` | Server not reachable |
| `authentication failed` | Invalid token |
| `websocket closed` | Connection dropped |
| `metrics sent` | Data successfully sent |

## Getting Help

If you can't resolve the issue:

1. Gather information:
   - OS and version
   - Agent version (`vstats-agent --version`)
   - Error logs
   - Network configuration

2. Join our [Telegram Group](https://t.me/zsai010_group/10)

3. Open an issue on [GitHub](https://github.com/zsai001/vstats/issues)

