# Scoop Bucket for vStats

This is the official [Scoop](https://scoop.sh/) bucket for vStats CLI on Windows.

## Installation

### Quick Install

```powershell
scoop bucket add vstats https://github.com/zsai001/scoop-vstats
scoop install vstats
```

### First Time Setup

If you don't have Scoop installed:

```powershell
# Install Scoop (run in PowerShell)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
Invoke-RestMethod -Uri https://get.scoop.sh | Invoke-Expression

# Then install vStats
scoop bucket add vstats https://github.com/zsai001/scoop-vstats
scoop install vstats
```

## Update

```powershell
scoop update vstats
```

## Uninstall

```powershell
scoop uninstall vstats
scoop bucket rm vstats
```

## Available Apps

| App | Description |
|-----|-------------|
| `vstats` | vStats CLI - Server monitoring management tool |

## Usage

After installation, you can use the `vstats` command:

```powershell
# Login to vStats Cloud
vstats login

# List your servers
vstats server list

# Create a new server
vstats server create web1

# View server metrics
vstats server metrics web1

# Get help
vstats --help
```

## Documentation

- [vStats Documentation](https://vstats.zsoft.cc/docs)
- [CLI Reference](https://vstats.zsoft.cc/docs/cli)

## Issues

If you encounter any issues with this bucket, please report them at:
https://github.com/zsai001/vstats/issues

## License

MIT License
