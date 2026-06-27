# Vidian 🔍

A lightweight, beautiful **read-only code viewer** that runs as a single Go binary and opens in your browser instantly. Built for quickly inspecting code, reading READMEs, and reviewing commit history — without the weight of a full IDE.

```bash
vidian .           # open current folder
vidian ~/projects  # open any folder
```

---

## Why Vidian?

When an AI generates code, or you need to quickly check a README, review a diff, or browse a commit — opening VS Code or a full IDE is often overkill. Vidian fills that gap:

- **Instant**: Opens a browser tab in under a second
- **Lightweight**: Single binary, `< 15 MB RAM` usage
- **Zero config**: No extensions, no language servers, no workspace setup
- **Read-only**: Safe to point at any directory — no accidental edits

---

## Features

- 📁 **File Explorer** — Tree view with expand/collapse, color-coded file icons
- 📝 **Monaco Editor** — The same editor engine as VS Code, syntax highlighting for 100+ languages
- 🔍 **Global Search** — Full-text content search across all files
- ⚡ **Quick Open** — `Ctrl+P` to jump to any file instantly
- 🖼️ **Image Preview** — View images inline, binary file metadata cards
- 📄 **Markdown Preview** — Side-by-side rendered markdown
- 🌿 **Git Integration**:
  - Browse commit history with full details in the main editor area
  - Side-by-side diff viewer for any changed file in a commit
  - View uncommitted changes (working tree vs HEAD)
  - Switch branches from the Git sidebar

---

## Installation

Three ways to install — pick what fits your workflow.

---

### Method 1: One-liner script *(Linux / macOS / Git Bash)*

Downloads a pre-built binary for your OS and architecture:

```bash
curl -sSL https://raw.githubusercontent.com/Ucok23/vidian/master/install.sh | bash
```

**Supports:** Linux (amd64, arm64), macOS (amd64, arm64), Windows via Git Bash (amd64, arm64)

To pin a specific version:
```bash
VIDIAN_VERSION=v1.0.0 curl -sSL https://raw.githubusercontent.com/Ucok23/vidian/master/install.sh | bash
```

On Git Bash the binary is placed in `~/bin` — make sure that's on your `PATH`.

---

### Method 2: PowerShell *(Windows)*

```powershell
irm https://raw.githubusercontent.com/Ucok23/vidian/master/install.ps1 | iex
```

Installs to `%LOCALAPPDATA%\Programs\Vidian` and adds it to your user `PATH` automatically. To pin a version:

```powershell
.\install.ps1 -Version v1.0.0
```

> The binary is unsigned, so Windows SmartScreen may warn on first launch — choose **More info → Run anyway**.

---

### Method 3: Build from source *(Linux / macOS — for contributors)*

Clone and use the Makefile:

```bash
git clone https://github.com/Ucok23/vidian.git
cd vidian
make install
```

This builds the frontend + Go binary and copies it to `/usr/local/bin/vidian`.

Other useful Makefile targets:

```bash
make help        # Show all available targets
make build       # Build binary only (frontend must already be built)
make all         # Build frontend + binary (no install)
make uninstall   # Remove from /usr/local/bin
make clean       # Remove build artifacts
```

---

## Usage

```bash
vidian .                      # open current directory
vidian /path/to/project       # open a specific folder
vidian . -port 9000           # custom port (default: 8080)
```

Then open **[http://localhost:8080](http://localhost:8080)** in your browser.

### Flags

| Flag | Default | Description |
|:---|:---|:---|
| `-dir` | `.` | Path to workspace directory |
| `-port` | `8080` | HTTP port to listen on |
| `-dev` | `false` | Serve frontend from disk (for development) |

---

## Keyboard Shortcuts

| Shortcut | Action |
|:---|:---|
| `Ctrl + P` | Quick Open — search and jump to any file |
| `Ctrl + B` | Toggle sidebar visibility |
| `Ctrl + Shift + F` | Focus global search |
| `Esc` | Close Quick Open palette |
| `↑` / `↓` | Navigate items in Quick Open |
| `Enter` | Open selected file |

---

## Architecture

```mermaid
graph TD
    Client[Browser: Svelte 5 + Monaco Editor] -->|HTTP| Server[Go HTTP Server]
    Server -->|embed.FS| Assets[Compiled Frontend Assets]
    Server -->|Safe Path Read| FS[Local Filesystem]
    Server -->|git CLI| Git[Git]
```

The entire app ships as a **single self-contained binary** — the Svelte + Monaco frontend is compiled and embedded at build time via Go's `embed` package. No Node.js, no npm, no external dependencies at runtime.

---

## Development

Run the frontend dev server and Go backend separately for hot-reload:

```bash
# Terminal 1 — Svelte with HMR
cd frontend && npm run dev

# Terminal 2 — Go backend in dev mode
go run ./cmd/vidian/main.go -dir . -dev -port 8080
```

### Tests

```bash
make visual-test
```

Builds the frontend, compiles the backend, starts the server, and runs the Playwright (`@playwright/test`) visual suite inside Docker across the file explorer, Monaco editor, Git panel, commit viewer, diff editor, and the GitLens features. Each run writes a timestamped folder under `tests/visual/results/` containing a custom dark-themed `index.html`, Playwright's native HTML report, and per-test traces, HD video, and screenshots.

---

## License

MIT
