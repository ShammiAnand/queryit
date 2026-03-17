#!/usr/bin/env bash
set -euo pipefail

REPO="shammianand/queryit"
BINARY="queryit"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── helpers ──────────────────────────────────────────────────────────────────

info() { printf '\033[1;34m  =>\033[0m %s\n' "$*"; }
ok() { printf '\033[1;32m  ok\033[0m %s\n' "$*"; }
die() {
  printf '\033[1;31merror:\033[0m %s\n' "$*" >&2
  exit 1
}

need() {
  command -v "$1" &>/dev/null || die "required tool not found: $1"
}

# ── detect OS / arch ─────────────────────────────────────────────────────────

detect_platform() {
  local os arch
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)

  case "$os" in
  linux) os="linux" ;;
  darwin) os="darwin" ;;
  *) die "unsupported OS: $os" ;;
  esac

  case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) die "unsupported architecture: $arch" ;;
  esac

  echo "${os}_${arch}"
}

# ── install from source (fallback) ───────────────────────────────────────────

install_from_source() {
  need go
  info "building from source..."
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  git clone --depth=1 "https://github.com/${REPO}.git" "$tmp/queryit" 2>/dev/null
  cd "$tmp/queryit"
  go build -ldflags "-s -w" -o "$tmp/$BINARY" .
  install_binary "$tmp/$BINARY"
}

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

# ── install from GitHub release ──────────────────────────────────────────────

install_from_release() {
  need curl

  info "fetching latest release tag..."
  local tag
  tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
    grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/')

  [[ -z "$tag" ]] && return 1 # no release yet -- fall through to source

  local platform
  platform=$(detect_platform)
  local url="https://github.com/${REPO}/releases/download/${tag}/${BINARY}_${platform}.tar.gz"

  info "downloading $tag for $platform..."
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  if ! curl -fsSL "$url" -o "$tmp/${BINARY}.tar.gz"; then
    info "no pre-built binary for $platform -- building from source"
    return 1
  fi

  tar -xzf "$tmp/${BINARY}.tar.gz" -C "$tmp"
  install_binary "$tmp/$BINARY"
}

# ── main ─────────────────────────────────────────────────────────────────────

main() {
  echo ""
  echo "  queryit installer"
  echo "  ─────────────────"
  echo ""

  # prefer release binary, fall back to source build
  if ! install_from_release 2>/dev/null; then
    install_from_source
  fi

  echo ""
  info "verifying installation..."
  if command -v "$BINARY" &>/dev/null; then
    ok "$($BINARY version)"
  else
    info "installation succeeded but $INSTALL_DIR is not in your PATH."
    info "add this to your shell profile:"
    echo ""
    echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
  fi
}

main "$@"
