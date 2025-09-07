# Loopback Manager

A loopback IP address management tool for Docker Compose in GitHub organization repositories

## Features

- Automatic IP assignment (starting from 127.0.0.10)
- IP duplication check
- Automatic detection of unassigned repositories
- Automatic generation and update of .env files
- Persistent configuration

## Installation

### Using Go Install
If you have Go environment set up (Go 1.19 or later), you can install directly:
```bash
go install github.com/takah/loopback-manager@latest
```

### Binary Download
```bash
# Run installation script
curl -sf https://raw.githubusercontent.com/takah/loopback-manager/main/scripts/install.sh | bash
```

### Build from Source
```bash
git clone https://github.com/takah/loopback-manager.git
cd loopback-manager
go build -o loopback-manager
```

## Usage

```bash
# List all repositories
loopback-manager list

# List with JSON output (for scripting)
loopback-manager list --json

# Scan for unassigned repositories
loopback-manager scan

# Scan with JSON output
loopback-manager scan --json

# Manually assign IP
loopback-manager assign myorg/myrepo

# Assign with specific IP
loopback-manager assign myorg/myrepo --ip 127.0.0.50

# Auto-assign IP to all unassigned repositories (dry-run by default)
loopback-manager auto-assign

# Execute the auto-assignment (actually make changes)
loopback-manager auto-assign --execute

# Check for duplicates
loopback-manager check

# Remove IP assignment
loopback-manager remove myorg/myrepo
```

## Configuration

Default configuration file: `~/.config/loopback-manager/config.yaml`

```yaml
base_dir: "~/github"
ip_range:
  base: "127.0.0"
  start: 10
  end: 254
```

Environment variable configuration:
- `GITHUB_BASE_DIR`: Base directory for GitHub repositories

## License

MIT License
