# tact

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey.svg)](https://github.com/Fabio-RibeiroB/tact)

**Tmux AI Control Tower** — A terminal dashboard for monitoring and managing multiple AI coding sessions running in tmux panes.

![Dashboard Preview](docs/imgs/dashboard.png)

## Features

- **Multi-session Monitoring**: Track multiple AI coding sessions simultaneously across tmux panes
- **Real-time Status Detection**: Automatically detects session states (idle, working, needs attention, disconnected)
- **Cost Tracking**: Monitor token usage and costs for Claude Code sessions
- **Context Window Awareness**: See context window utilization at a glance
- **Desktop Notifications**: Get alerts when sessions need attention
- **Shared Todo Management**: Create and manage project-level todos synced across sessions
- **Remote Control**: Send inputs to sessions without switching panes

### Supported AI Tools

| Tool | Detection | Status | Context | Cost |
|------|-----------|--------|---------|------|
| Claude Code | Full | Full | Full | Full |
| Kiro CLI | Full | Full | Partial | — |
| Codex | Full | Full | Partial | — |
| Opencode | Full | Full | Partial | — |

## Installation

### From Source

```bash
git clone https://github.com/Fabio-RibeiroB/tact.git
cd tact/go
make install
```

### Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/Fabio-RibeiroB/tact/main/go/install.sh | bash
```

### Requirements

- Go 1.22 or later
- tmux 3.0+

## Usage

### Launch the Dashboard

```bash
tact
```

![Sessions View](docs/imgs/sessions.png)

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate sessions/todos |
| `Tab` | Switch focus panel |
| `Enter` | Switch to selected pane |
| `r` | Refresh session discovery |
| `n` | Toggle notifications |
| `i` | Enter insert mode (send keys to session) |
| `Esc` | Exit insert mode |
| `q` / `Ctrl+C` | Quit |

#### Session Interaction (when session needs attention)

| Key | Action |
|-----|--------|
| `y` | Confirm/continue (sends Enter) |
| `a` | Auto-approve tool use |
| `!` | Cancel current operation (sends Escape) |

#### Todo Panel

| Key | Action |
|-----|--------|
| `i` | Add new todo |
| `Enter` | Mark todo as done |
| `d` / `x` | Delete todo |
| `Esc` | Exit insert mode |

### CLI Commands

```bash
# Manage todos from command line
tact todo list
tact todo add "Implement feature X" -p myproject -t feature,backend
tact todo done abc12345
tact todo start abc12345
tact todo rm abc12345
```

## Screenshots

### Main Dashboard

![Main Dashboard](docs/imgs/dashboard.png)

The main dashboard shows:
- **Left panel**: Session list with status indicators and todo list
- **Right panel**: Detailed view of selected session including task summary, cost, and context

### Session States

| Status | Icon | Description |
|--------|------|-------------|
| Idle | `●` | Session waiting for input |
| Working | `◉` | Session actively processing |
| Needs Attention | `◉` | Session awaiting user response |
| Disconnected | `○` | Pane no longer accessible |

## Architecture

```
tact/
├── go/
│   ├── cmd/tact/           # CLI entry point
│   ├── internal/
│   │   ├── model/          # Domain types (SessionInfo, TodoItem, etc.)
│   │   ├── parser/         # JSONL parsing, status detection
│   │   ├── tmux/           # Tmux pane discovery and capture
│   │   ├── notify/         # Desktop notifications
│   │   ├── todo/           # Shared todo store with sync
│   │   └── tui/            # Bubble Tea terminal UI
│   ├── go.mod
│   └── Makefile
└── docs/
    └── imgs/               # Screenshots and diagrams
```

### Data Storage

Session data and todos are stored in `~/.local/share/tact/`:
- `sessions/<session-id>.jsonl` — Parsed session data
- `todos/<project-slug>.json` — Project-specific todo lists

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes following conventional commits
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Terminal UI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Style definitions
- [Cobra](https://github.com/spf13/cobra) — CLI framework