<div align="center">

# 🎭 Orchestrate

**Run AI coding agents, custom dev environments, or both — each in their own git worktree**

<br>

![droid](https://img.shields.io/badge/droid-ff8c00?style=for-the-badge&logoColor=white)
![claude](https://img.shields.io/badge/claude-d2b48c?style=for-the-badge&logoColor=white)
![codex](https://img.shields.io/badge/codex-1e1e1e?style=for-the-badge&logoColor=white)

</div>

---

## ⚡ Quick Start

**1. Create `settings.orchestrate.yaml` in the Orchestrate data directory:**

The settings file goes in the platform-specific data directory:
- **macOS:** `~/Library/Application Support/Orchestrate/settings.orchestrate.yaml`
- **Linux:** `~/.local/share/orchestrate/settings.orchestrate.yaml`
- **Windows:** `%APPDATA%\Orchestrate\settings.orchestrate.yaml`

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
        - command: "npm run dev"
          title: "Dev Server"
        - command: "npm run test:watch"
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

**2. Run with a GitHub repo:**

```bash
# Use default preset
orchestrate --repo groq/orion --name fix-bug --prompt "Fix the login timeout issue"

# Or use the fullstack preset
orchestrate --repo groq/orion --name fix-bug --prompt "Fix the login timeout issue" --preset fullstack
```

This clones/updates the repo from the main branch, creates isolated git worktrees, and launches agents/commands in separate iTerm2 panes.

<br>

<div align="center">

📦 **Clones/updates repo** → 📁 **Creates worktrees** → 🖥️ **Opens iTerm2 panes** → 🤖 **Launches agents** → ✨ **Parallel coding**

</div>

---

## 📦 Installation

```bash
go install github.com/groq/orchestrate@latest
```

**Requirements:** macOS with iTerm2, Go 1.21+, and your preferred AI coding agents installed.

---

## 🔧 CLI Reference

| Flag | Description |
|------|-------------|
| `--repo` | **Required.** GitHub repo to clone (e.g., `groq/openbench`). Clones fresh or updates from main branch. |
| `--name` | **Required.** Branch name prefix for worktrees. Each branch gets a unique hex suffix. |
| `--prompt` | **Required.** The prompt to pass to each agent. |
| `--preset` | Use a preset from `settings.orchestrate.yaml`. Defaults to the config's default preset. |
| `--n` | Multiplier for agent windows. `--n 2` runs each agent twice. |

---

## 📖 Use Cases

### 🏃 Single Agent with Dev Environment — Run your app alongside the agent + spare terminal

One agent, but with your dev server running and an extra shell for manual testing:

```yaml
# settings.orchestrate.yaml
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

---

## ⚙️ Configuration

Create `settings.orchestrate.yaml` in the Orchestrate data directory:

```yaml
# settings.orchestrate.yaml
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

---

## 🔄 How It Works

1. **Clone & Update** — Clones the specified GitHub repo (or updates if it already exists), always fetching the latest from main
2. **Git Worktrees** — Creates isolated worktrees, each with a unique branch based on main
3. **iTerm2 Integration** — Opens windows with up to 6 panes in a grid, color-coded by agent
4. **Parallel Execution** — Agents work simultaneously; compare branches and merge the best

**Data Location:**
- macOS: `~/Library/Application Support/Orchestrate/`
- Linux: `~/.local/share/orchestrate/` (or `$XDG_DATA_HOME/orchestrate`)
- Windows: `%APPDATA%\Orchestrate`

Inside this directory:
- `settings.orchestrate.yaml` — **Required.** Your presets configuration
- `repos/` — Cloned repositories
- `worktrees/` — Git worktrees for agent sessions

**Example Output:**

```
⚙️  Settings: ~/Library/Application Support/Orchestrate/settings.orchestrate.yaml
📦 Repo: groq/openbench
🔄 Fetching latest from main branch...
📂 Local path: ~/Library/Application Support/Orchestrate/repos/groq-openbench
🌿 Base branch: main
💬 Prompt: Fix the authentication bug
✅ Created worktree: .../worktrees/groq-openbench-fix-auth-a3f2 (branch: fix-auth-a3f2, agent: codex)
   🖥️  Command: App (branch: fix-auth-a3f2)
   🖥️  Command: Terminal (branch: fix-auth-a3f2)

✨ Started 3 session(s) in 1 window(s)!
```

---

## 📁 Project Structure

```
orchestrate/
├── main.go           # CLI entry point
├── config/           # YAML configuration loading
├── git/              # Git worktree operations
├── agents/           # Agent parsing and colors
├── terminal/         # iTerm2 window management
└── util/             # Utilities
```

---

<div align="center">

**Built with Go** • **Requires macOS + iTerm2**

</div>
