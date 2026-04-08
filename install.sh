#!/usr/bin/env bash
set -euo pipefail

REPO="lunemis/skimd"
BINARY="skimd"
INSTALL_DIR="/usr/local/bin"

# Parse flags
for arg in "$@"; do
    case "$arg" in
        --help|-h)
            echo "Usage: install.sh"
            exit 0
            ;;
    esac
done

# --- Helpers ---

info()  { printf '\033[1;34m→\033[0m %s\n' "$*"; }
ok()    { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
skip()  { printf '\033[1;33m⊘\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m⚠\033[0m %s\n' "$*"; }
fail()  { printf '\033[1;31m✗\033[0m %s\n' "$*"; exit 1; }

ask() {
    local prompt="$1"
    local default="${2:-Y}"
    local yn
    if [ "$default" = "Y" ]; then
        printf '\033[1m%s\033[0m [Y/n] ' "$prompt"
    else
        printf '\033[1m%s\033[0m [y/N] ' "$prompt"
    fi
    read -r yn </dev/tty || yn=""
    yn="${yn:-$default}"
    case "$yn" in
        [Yy]*) return 0 ;;
        *) return 1 ;;
    esac
}

detect_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    case "$os" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      fail "Unsupported OS: $os" ;;
    esac

    case "$arch" in
        x86_64|amd64)   ARCH="amd64" ;;
        arm64|aarch64)  ARCH="arm64" ;;
        *)              fail "Unsupported architecture: $arch" ;;
    esac
}

latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | head -1 \
        | sed -E 's/.*"([^"]+)".*/\1/'
}

# --- Steps ---

install_binary() {
    info "Installing ${BINARY}..."

    # Try go install first
    if command -v go &>/dev/null; then
        if ask "  Go detected. Use 'go install'?"; then
            go install "github.com/${REPO}/cmd/${BINARY}@latest"
            ok "${BINARY} installed via go install"
            return
        fi
    fi

    # Fall back to GitHub Release download
    detect_platform
    local version
    version="$(latest_version)" || fail "Could not determine latest version"
    local name="${BINARY}_${version#v}_${OS}_${ARCH}"
    local url="https://github.com/${REPO}/releases/download/${version}/${name}.tar.gz"

    info "Downloading ${BINARY} ${version} (${OS}/${ARCH})..."
    local tmp
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT

    curl -fsSL "$url" -o "${tmp}/${name}.tar.gz" \
        || fail "Download failed. Check https://github.com/${REPO}/releases"
    tar -xzf "${tmp}/${name}.tar.gz" -C "$tmp"

    # Determine install path
    if [ -w "$INSTALL_DIR" ]; then
        install -m 755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
        ok "${BINARY} installed to ${INSTALL_DIR}/${BINARY}"
    else
        local local_bin="${HOME}/.local/bin"
        mkdir -p "$local_bin"
        install -m 755 "${tmp}/${BINARY}" "${local_bin}/${BINARY}"
        ok "${BINARY} installed to ${local_bin}/${BINARY}"
        if ! echo "$PATH" | grep -q "$local_bin"; then
            warn "Add ${local_bin} to your PATH"
        fi
    fi
}

setup_keybind() {
    info "Setting up tmux keybinding..."

    local conf="${HOME}/.tmux.conf"
    local binary_path
    binary_path="$(command -v "$BINARY" 2>/dev/null || echo "$BINARY")"
    local line="bind v display-popup -E -w 92% -h 88% -d \"#{pane_current_path}\" \"${binary_path} .\"  # skimd popup keybinding"

    if [ -f "$conf" ] && grep -q "skimd" "$conf"; then
        ok "Keybinding already exists in ${conf}"
    else
        echo "$line" >> "$conf"
        ok "Keybinding added to ${conf}: prefix + v"
        info "Reload tmux config: tmux source-file ${conf}"
    fi
}

# --- Main ---

echo ""
echo "  ⚡ skimd installer"
echo ""

# Step 1: Install binary
if ask "[1/2] Install ${BINARY} binary?"; then
    install_binary
else
    skip "${BINARY} binary installation skipped"
fi
echo ""

# Step 2: tmux keybinding
if ask "[2/2] Configure tmux keybinding (prefix+v)?"; then
    setup_keybind
else
    skip "Keybinding setup skipped"
fi

echo ""
echo "  Done! Run 'skimd' or press prefix+v in tmux."
echo ""
