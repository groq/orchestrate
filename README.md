<div align="center">

<img width="1800" height="628" alt="Orchestrate_Header_1800x628" src="https://github.com/user-attachments/assets/5b04d89e-6342-412a-93fe-cbfb815214b1" />

<br>

### **Initialize, monitor, and orchestrate coding agents running in parallel within isolated git worktrees.**

![droid](https://img.shields.io/badge/droid-ff8c00?style=for-the-badge&logoColor=white)
![claude](https://img.shields.io/badge/claude-d2b48c?style=for-the-badge&logoColor=white)
![codex](https://img.shields.io/badge/codex-1e1e1e?style=for-the-badge&logoColor=white)

</div>

---

## 📦 Installation

```bash
# from the repo root
cargo install --path .
```

**Requirements:** macOS with iTerm2, Rust toolchain, and your preferred AI coding agents installed.

---

## ⚡ Quick Start

**1. Create `settings.yaml` in the Orchestrate data directory:**

The settings file goes in the platform-specific data directory:
- **macOS:** `~/.orchestrate/settings.yaml`
- **Linux:** `~/.local/share/orchestrate/settings.yaml`
- **Windows:** `%APPDATA%\Orchestrate\settings.yaml`

```yaml
default: default

presets:
  # Simple: just an agent
  default:
    - agent: claude

  # Complex: agent with dev server, tests, and shell
  fullstack:
    - agent: claude
      commands:
        - command: "./bin/go run ./cmd/server"
          title: "Dev Server"
        - command: "./bin/go test ./..."
          title: "Tests"
        - command: ""
          title: "Shell"

  # Multi-agent: compare different agents on the same task
  compare:
    - agent: droid
    - agent: claude
    - agent: codex

  # Best of n: use n to run multiple instances of an agent
  parallel:
    - agent: claude
      n: 3
```

**2. Option A: Run via TUI**

```bash
# Open TUI
orchestrate
```

<img width="1252" height="849" alt="Screenshot 2025-12-30 at 2 32 02 PM" src="https://github.com/user-attachments/assets/29155db0-6043-454c-9843-ff97dcd19f58" />

<img width="1255" height="840" alt="Screenshot 2025-12-30 at 2 32 39 PM" src="https://github.com/user-attachments/assets/afceac54-011c-41e8-a9ef-09eecd5c8c37" />

<br>
<br>

**3. Option B: Run via CLI:**

```bash
# Use default preset
orchestrate --repo groq/openbench --name fix-bug --prompt "Create a new eval that implements a subset of SimpleQA"

# Or use the fullstack preset
orchestrate --repo groq/openbench --name fix-bug --prompt "Create a new eval that implements a subset of SimpleQA" --preset fullstack
```

This clones/updates the repo from the main branch, creates isolated git worktrees, and launches agents/commands in separate iTerm2 panes.

<br>

<div align="center">

📦 **Clones/updates repo** → 📁 **Creates worktrees** → 🖥️ **Opens iTerm2 panes** → 🤖 **Launches agents** → ✨ **Parallel coding**

</div>

---

## 🔧 CLI Reference

| Flag | Description |
|------|-------------|
| `--repo` | **Required.** GitHub repo to clone (e.g., `groq/openbench`). Clones fresh or updates from main branch. |
| `--name` | **Required.** Branch name prefix for worktrees. Each branch gets a unique hex suffix. |
| `--prompt` | **Required.** The prompt to pass to each agent. |
| `--preset` | Use a preset from `settings.yaml`. Defaults to the config's default preset. |
| `--n` | Multiplier for agent worktrees. `--n 2` runs each agent twice. |

---

## 📖 Use Cases

### 🏃 Single Agent with Dev Environment — Run your app alongside the agent + spare terminal

One agent, but with your dev server running and an extra shell for manual testing:

```yaml
# settings.yaml
default: dev

presets:
  dev:
    - agent: codex
      commands:
        - command: "npm run dev"
          title: "App"
          color: "#00ff00"
        - command: ""
          title: "Terminal"
```

```bash
orchestrate --repo myorg/myapp --name feature-auth --prompt "Add OAuth2 login"
```

### 🔬 Evaluate Multiple Agents — Compare how droid/claude/codex solve the same problem

Give the same task to different agents and pick the best solution:

```yaml
presets:
  eval:
    - agent: droid
    - agent: claude
    - agent: codex
```

```bash
orchestrate --repo myorg/myapp --name eval-refactor --preset eval --prompt "Refactor the database layer"
# Compare branches: eval-refactor-a3f2, eval-refactor-b7c1, eval-refactor-d9e4
```

### 🚀 Parallel Execution at Scale — Use `--n` multiplier for maximum throughput

Run multiple instances of each agent when you need sheer volume:

```yaml
presets:
  heavy:
    - agent: claude
    - agent: codex
```

```bash
orchestrate --repo myorg/myapp --name big-task --preset heavy --n 3 --prompt "Add comprehensive test coverage"
# Creates 6 worktrees: 3 claude, 3 codex
```

### 🛠️ Custom Dev Workflows — Full engineering sessions (backend, frontend, tests, shell)

Set up complete development environments with multiple services and tools:

```yaml
presets:
  fullstack:
    - agent: claude
      commands:
        - command: "cd backend && cargo run"
          title: "Backend API"
          color: "#ff6600"
        - command: "cd frontend && npm run dev"
          title: "Frontend"
          color: "#00ccff"
        - command: ""
          title: "Shell"
    - agent: codex
      commands:
        - command: "npm run test:watch"
          title: "Tests"
          color: "#ffff00"
```

---

## 🤖 Supported Agents

| Agent | Description | Color |
|-------|-------------|-------|
| **droid** | Factory AI's coding agent | 🟠 Orange |
| **claude** | Anthropic's Claude CLI | 🟤 Sand |
| **codex** | OpenAI's Codex CLI | ⚫ Black |

Agents must be installed and available in your PATH.

### 🪵 Tracking shell commands in the activity log

Every agent shell defines a `track` helper when logging is enabled for that worktree. Use it to replace or rerun commands while keeping the activity panel up to date:

```bash
# inside the agent shell
track npm test
track ./bin/dev-server
```

`track` pipes stdout/stderr to the worktree's activity log (e.g. `~/.orchestrate/activity/<branch>.log`) so the TUI activity stream stays clean even when you stop an agent and run custom commands.

---

## ⚙️ Configuration

Create `settings.yaml` in the Orchestrate data directory:

```yaml
# settings.yaml
# Default preset when --preset is not specified
default: standard

presets:
  # Simple: just agents
  standard:
    - agent: droid
    - agent: codex
    - agent: codex

  # With commands: agents + terminals in their worktrees
  dev:
    - agent: codex
      commands:
        - command: "./bin/go run ./cmd/myapp"
          title: "App"
        - command: ""
          title: "Terminal"
```

### Command Options

| Option | Description |
|--------|-------------|
| `command` | Shell command to run (empty = just open terminal) |
| `title` | Custom title for the terminal tab |
| `color` | Hex color for the tab, e.g., `#ff8800` |

> 💡 Commands run in their parent agent's worktree and show the branch name in the title.

### TUI Navigation (Ratatui)

- `1/2/3/4` to jump to Worktrees / Launch / Settings / Presets
- `Tab` / `Shift+Tab` cycles views
- `?` toggles the help overlay
- Worktrees: `↑/↓/g/G` navigate, `Enter` focus existing window, `o` reopen, `d` details sidebar, `x` delete with confirm, `Ctrl+R` refresh, `Ctrl+P` toggle sidebar
- Launch: arrows or Tab to move, `Ctrl+Enter` or Launch button to start sessions, `←/→` cycle presets
- Settings: `↑/↓` select, `←/→` toggle values, `Enter` saves

---

## 🔄 How It Works

1. **Clone & Update** — Clones the specified GitHub repo (or updates if it already exists), always fetching the latest from main
2. **Git Worktrees** — Creates isolated worktrees, each with a unique branch based on main
3. **iTerm2 Integration** — Opens windows with up to 6 panes in a grid, color-coded by agent
4. **Parallel Execution** — Agents work simultaneously; compare branches and merge the best

**Data Location:**
- macOS: `~/.orchestrate/`
- Linux: `~/.local/share/orchestrate/` (or `$XDG_DATA_HOME/orchestrate`)
- Windows: `%APPDATA%\Orchestrate`

Inside this directory:
- `settings.yaml` — **Required.** Your presets configuration
- `repos/` — Cloned repositories
- `worktrees/` — Git worktrees for agent sessions

---

## 🙏 Acknowledgments

This app was developed by Benjamin Klieger at [Groq](https://groq.com).

The terminal UI is inspired by and borrows design patterns from [**gh-dash**](https://github.com/dlvhdr/gh-dash) by [@dlvhdr](https://github.com/dlvhdr) — an impressive GitHub CLI dashboard. Please check it out!

---

<div align="center">

**Built with Rust + Ratatui** • **Requires macOS + iTerm2**

</div>
