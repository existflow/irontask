# IronTask

<p align="center">
  <img src="docs/demo.gif" alt="IronTask Demo" width="600">
</p>

**IronTask** is a modern, privacy-focused terminal-based todo application designed for developers who live in the CLI. It features a beautiful TUI, project management, and end-to-end encrypted synchronization across devices.

## Features

- **🚀 TUI & CLI**: Interactive terminal UI (vim-like bindings) + scriptable CLI commands.
- **🔒 End-to-End Encryption**: Tasks are encrypted before they leave your device. The server sees only blobs.
- **🔄 Auto-Sync**: Seamlessly syncs changes in the background while you work.
- **📂 Projects & Contexts**: Organize tasks into projects.
- **⚡ Fast**: Built with Go and SQLite.

## Installation

### From Source

```bash
go install github.com/tphuc/irontask/cmd/irontask@latest
```

### Docker (Sync Server)

To self-host the sync server:

```bash
git clone https://github.com/tphuc/irontask.git
cd irontask
docker-compose up -d
```

## Quick Start

1. **Start the App**
   Run `task` to open the interactive UI.

2. **Keybindings (TUI)**
   - `a`: Add task
   - `x` or `Enter`: Mark done
   - `d`: Delete
   - `e`: Edit
   - `1-4`: Set priority (P1-P4)
   - `/`: Search/Filter
   - `Tab`: Switch between Project List and Task List
   - `j/k` or `Up/Down`: Navigate
   - `q`: Quit

3. **CLI Commands**
   ```bash
   task add "Buy milk"
   task list
   task done <id>
   ```

## Configuration

Configuration is optional. Create `~/.irontask/config.yaml`:

```yaml
editor: vim          # Default editor
confirm_delete: true # Ask before deleting in CLI
```

## Sync Setup

IronTask allows you to sync tasks between devices encrypted.

1. **Register** (on first device):
   ```bash
   task sync register
   # Prompts for email, password
   ```

2. **Login** (on other devices):
   ```bash
   task sync login
   # Prompts for email, password
   ```

3. **Encryption Key**:
   During registration/login, a unique encryption key is derived from your password. **If you lose your password, your data cannot be recovered.**

4. **Status**:
   Check sync status:
   ```bash
   task sync status
   ```

## Shell Completion

Generate completion script for your shell (bash, zsh, fish, powershell).

**Zsh Example**:
```bash
echo "autoload -U compinit; compinit" >> ~/.zshrc
task completion zsh > "${fpath[1]}/_task"
# Restart shell
```

## License

MIT
