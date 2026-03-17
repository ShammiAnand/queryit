#!/usr/bin/env bash
set -euo pipefail

REPO="ShammiAnand/queryit"
BINARY="queryit"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── helpers ───────────────────────────────────────────────────────────────────

info() { printf '\033[1;34m  =>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  ok\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

need() { command -v "$1" &>/dev/null || die "required tool not found: $1 (please install it)"; }

# ── detect platform ───────────────────────────────────────────────────────────

detect_platform() {
  local os arch
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)

  case "$os" in
    linux)  os="linux"  ;;
    darwin) os="darwin" ;;
    *)      die "unsupported OS: $os" ;;
  esac

  case "$arch" in
    x86_64|amd64)   arch="amd64" ;;
    arm64|aarch64)  arch="arm64" ;;
    *)               die "unsupported architecture: $arch" ;;
  esac

  echo "${os}_${arch}"
}

# ── install binary ────────────────────────────────────────────────────────────

install_binary() {
  local src="$1"
  info "installing to $INSTALL_DIR/$BINARY"
  if [[ -w "$INSTALL_DIR" ]]; then
    install -m 755 "$src" "$INSTALL_DIR/$BINARY"
  else
    info "need sudo to write to $INSTALL_DIR"
    sudo install -m 755 "$src" "$INSTALL_DIR/$BINARY"
  fi
  ok "$BINARY installed to $INSTALL_DIR/$BINARY"
}

# ── main ──────────────────────────────────────────────────────────────────────

main() {
  need curl

  echo ""
  echo "  queryit installer"
  echo "  ─────────────────"
  echo ""

  info "fetching latest release..."
  local tag
  tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

  [[ -z "$tag" ]] && die "no releases found at github.com/${REPO} — try again later"

  local platform
  platform=$(detect_platform)

  local url="https://github.com/${REPO}/releases/download/${tag}/${BINARY}_${platform}.tar.gz"

  info "downloading $BINARY $tag ($platform)..."
  local tmp
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  if ! curl -fsSL "$url" -o "$tmp/${BINARY}.tar.gz"; then
    die "download failed: $url\nCheck https://github.com/${REPO}/releases for available builds."
  fi

  tar -xzf "$tmp/${BINARY}.tar.gz" -C "$tmp"

  # the tarball contains a binary named queryit_<os>_<arch>; rename it
  local extracted="$tmp/${BINARY}_${platform}"
  if [[ ! -f "$extracted" ]]; then
    # fallback: find any executable in the tmp dir
    extracted=$(find "$tmp" -maxdepth 1 -type f -perm /111 | head -1)
    [[ -z "$extracted" ]] && die "could not find binary in downloaded archive"
  fi

  install_binary "$extracted"

  echo ""
  info "verifying..."
  if command -v "$BINARY" &>/dev/null; then
    ok "$("$BINARY" version)"
  else
    info "$INSTALL_DIR is not in your PATH. Add this to your shell profile:"
    echo ""
    echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
  fi
}

main "$@"
