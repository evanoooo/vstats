# Homebrew Tap for vStats

This is the official [Homebrew](https://brew.sh/) tap for vStats CLI.

## Installation

### Quick Install

```bash
brew install zsai001/vstats/vstats
```

### Manual Install

```bash
# Add the tap
brew tap zsai001/vstats

# Install vStats CLI
brew install vstats
```

## Upgrade

```bash
brew upgrade vstats
```

## Uninstall

```bash
brew uninstall vstats
brew untap zsai001/vstats
```

## Available Formulas

| Formula | Description |
|---------|-------------|
| `vstats` | vStats CLI - Server monitoring management tool |

## Usage

After installation, you can use the `vstats` command:

```bash
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

If you encounter any issues with this tap, please report them at:
https://github.com/zsai001/vstats/issues

## License

MIT License - see the main repository for details.
