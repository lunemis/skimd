# skimd

**Skim through markdown docs without leaving the terminal.**

AI tools generate piles of markdown — plans, specs, changelogs, meeting notes. skimd lets you browse and read them in a TUI, right where you work.

[한국어](README.md)

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

<!-- TODO: demo GIF — browse folders, hover preview, enter reader, search, outline -->
<!-- ![Demo](assets/demo.gif) -->

## How it works

1. Open skimd in a directory full of markdown files
2. Browse folders, hover to preview any document instantly
3. Press `Enter` to read in detail — search, outline, section jump
4. Close and return to your session

<!-- TODO: screenshot — browser + preview panel side by side -->
<!-- ![Screenshot](assets/screenshot.png) -->

## Features

### Hover preview
Move your cursor over any markdown file — the preview panel renders it immediately. No need to open anything.

<!-- TODO: GIF — cursor moving over files, preview changing -->
<!-- ![Hover preview](assets/hover-preview.gif) -->

### Reader mode
Press `Enter` to enter a full reader view with search (`/`), outline (`o`), section jumping (`[`/`]`), and adjustable width (`-`/`=`).

### Outline view
Toggle between full outline (navigate + jump), side outline (passive position marker), or hidden.

### Search
`/` to search in browser (filename filter) or reader (full-text search). `n`/`N` to cycle matches.

### tmux popup integration
One keybinding to pop up skimd over your current session. Works great alongside [mux](https://github.com/lunemis/mux).

```bash
# Add to ~/.tmux.conf (or run: skimd --print-tmux-binding)
bind v display-popup -E -w 92% -h 88% -d "#{pane_current_path}" "skimd ."
```

<!-- TODO: GIF — tmux popup appearing over a Claude session -->
<!-- ![Popup](assets/popup.gif) -->

### More
- **Auto-reload**: Detects file changes and re-renders
- **Position restore**: Remembers scroll position when switching between files
- **Zen mode**: `z` to hide the browser panel
- **Adaptive width**: `-`/`=` to adjust reader width
- **File filter**: `a` to toggle between markdown-only and all files

## Quick Start

```bash
# One-line interactive installer (recommended)
curl -sSL https://raw.githubusercontent.com/lunemis/skimd/main/install.sh | bash

# Homebrew
brew install lunemis/tap/skimd

# Go
go install github.com/lunemis/skimd/cmd/skimd@latest

# From source
git clone https://github.com/lunemis/skimd.git && cd skimd && make install
```

Then:

```bash
skimd                    # browse current directory
skimd /path/to/docs      # browse specific directory
skimd README.md           # open a file directly
```

## Keybindings

### Browser

| Key | Action |
|---|---|
| `j` / `k` | Move down / up |
| `Enter` | Open directory or file |
| `h` / `←` / `Backspace` | Go to parent directory |
| `/` | Filter files |
| `a` | Toggle markdown-only / all files |
| `r` | Refresh |
| `q` | Quit |

### Reader

| Key | Action |
|---|---|
| `j` / `k` | Scroll down / up |
| `PgUp` / `PgDn` | Page scroll |
| `g` / `G` | Top / bottom |
| `/` | Search |
| `n` / `N` | Next / previous match |
| `o` | Cycle outline: full → side → hidden |
| `[` / `]` | Previous / next heading |
| `-` / `=` | Shrink / expand width |
| `z` | Zen mode (hide browser) |
| `Esc` / `h` | Back to browser |

## Requirements

- Go 1.24+ (build only)
- tmux 3.2+ (popup mode, optional)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
