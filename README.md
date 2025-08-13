# devmod

**Manage development environments with tmux layouts and project profiles.**

`devmod` automates your dev workspace setup by launching predefined `tmux` layouts with multiple panes for your backend, frontend, Git UI, logs, or any other tools — all with a single command.

## Features

- Define project profiles in YAML (`~/.config/devmod/config.yml` and per-project `.devmod.yml`)
- Launch a full tmux session with one command: `devmod up`
- Automatically split panes and set working directories/commands
- Works with any project structure (backend/frontend/tests/logs/etc.)
- Detects missing tools before launching

## Example Config

```yaml
version: 1
sessionName: your-project-name
profiles:
  full:
    layout:
      columns:
        - rows: ["be", "fe"]
        - rows: ["git"]
    panes:
      be:
        name: "BE"
        dir: "/path/to/backend"
        cmd: "dotnet watch run"
      fe:
        name: "FE"
        dir: "/path/to/frontend"
        cmd: "npm start"
      git:
        name: "Git"
        dir: "/path/to/project"
        cmd: "lazygit"
```

## Usage

```bash
# Start default profile
devmod up

# Start a specific profile
devmod up --profile full
```

On first run, devmod will:

1. Load global and project-specific configs
2. Verify required tools are installed (tmux, lazygit, etc.)
3. Create and attach to the tmux session

## Installation

### Homebrew (macOS)

```bash
brew tap sirwanafifi/devmod
brew install devmod
```

### From source

```bash
go install github.com/sirwanafifi/devmod@latest
```

## License

MIT License © 2025 Sirwan Afifi
