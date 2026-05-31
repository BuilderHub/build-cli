#!/usr/bin/env bash
#
# builderhub install script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/builderhub/build-cli/main/scripts/install.sh | bash
#   curl -fsSL ... | bash -s -- --version v0.1.0 --prefix ~/.local/bin
#
# Environment variables:
#   VERSION   - Specific version tag (e.g. v1.2.3). Defaults to latest.
#   PREFIX    - Install directory. Defaults to best writable location.
#   VERIFY    - Set to "1" to verify checksums (requires sha256sum or shasum).
#
set -euo pipefail

REPO="builderhub/build-cli"
BINARY="builderhub"
GITHUB="https://github.com"
GITHUB_API="https://api.github.com/repos/${REPO}"

# Colors (disabled if not tty)
if [ -t 1 ]; then
  BOLD="\033[1m"
  GREEN="\033[32m"
  YELLOW="\033[33m"
  RED="\033[31m"
  RESET="\033[0m"
else
  BOLD=""
  GREEN=""
  YELLOW=""
  RED=""
  RESET=""
fi

info()  { echo -e "${GREEN}==>${RESET} $1"; }
warn()  { echo -e "${YELLOW}==>${RESET} $1"; }
error() { echo -e "${RED}error:${RESET} $1" >&2; exit 1; }

usage() {
  cat <<EOF
builderhub installer

Usage:
  curl -fsSL https://raw.githubusercontent.com/builderhub/build-cli/main/scripts/install.sh | bash
  curl -fsSL ... | bash -s -- [options]

Options:
  --version <tag>     Install specific version (e.g. v0.5.0). Default: latest
  --prefix <dir>      Install directory. Default: auto-detect best location
  --verify            Verify checksums after download
  --help              Show this help

Environment:
  VERSION, PREFIX, VERIFY have the same effect as the flags above.
EOF
}

# Parse args
while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:-}"; shift 2 ;;
    --prefix)  PREFIX="${2:-}"; shift 2 ;;
    --verify)  VERIFY=1; shift ;;
    --help|-h) usage; exit 0 ;;
    *) error "unknown argument: $1 (use --help for usage)" ;;
  esac
done

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin|linux) ;;
  *) error "unsupported OS: $OS (only darwin and linux are supported)" ;;
esac

# Detect ARCH
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  armv7l|armv6l) error "32-bit ARM is not supported" ;;
  i386|i686)     error "32-bit x86 is not supported" ;;
  *)             error "unsupported architecture: $ARCH" ;;
esac

# --- Release resolution helpers (robust against goreleaser naming) ---

get_release_json() {
  local tag="$1"
  local url
  if [ -z "$tag" ] || [ "$tag" = "latest" ]; then
    url="${GITHUB_API}/releases/latest"
  else
    case "$tag" in
      v*) url="${GITHUB_API}/releases/tags/${tag}" ;;
      *)  url="${GITHUB_API}/releases/tags/v${tag}" ;;
    esac
  fi

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -H "Accept: application/vnd.github+json" "$url" 2>/dev/null || true
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- -H "Accept: application/vnd.github+json" "$url" 2>/dev/null || true
  fi
}

find_asset_name() {
  local json="$1"
  if command -v jq >/dev/null 2>&1; then
    echo "$json" | jq -r --arg os "$OS" --arg arch "$ARCH" '
      .assets[]
      | select(.name | contains($os) and contains($arch))
      | select(.name | endswith(".tar.gz") or endswith(".zip"))
      | .name
    ' | (grep '\.tar\.gz$' || cat) | head -1
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c '
import json, sys
data = json.loads(sys.stdin.read())
osys = sys.argv[1] if len(sys.argv) > 1 else ""
arch = sys.argv[2] if len(sys.argv) > 2 else ""
cands = []
for a in data.get("assets", []):
    n = a.get("name", "")
    if osys in n and arch in n and (n.endswith(".tar.gz") or n.endswith(".zip")):
        cands.append(n)
for c in cands:
    if c.endswith(".tar.gz"):
        print(c); sys.exit(0)
if cands:
    print(cands[0])
' "$OS" "$ARCH"
  else
    # Fallback: assume no "v" in filename (current goreleaser behavior)
    local ver
    ver=$(echo "$VERSION" | sed 's/^v//')
    echo "${BINARY}_${ver}_${OS}_${ARCH}.tar.gz"
  fi
}

find_download_url() {
  local json="$1" name="$2"
  if command -v jq >/dev/null 2>&1; then
    echo "$json" | jq -r --arg n "$name" '
      .assets[] | select(.name == $n) | .browser_download_url
    ' | head -1
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c '
import json, sys
data = json.loads(sys.stdin.read())
name = sys.argv[1]
for a in data.get("assets", []):
    if a.get("name") == name:
        print(a.get("browser_download_url", ""))
        sys.exit(0)
' "$name"
  fi
}

find_digest() {
  local json="$1" name="$2"
  if command -v jq >/dev/null 2>&1; then
    echo "$json" | jq -r --arg n "$name" '
      .assets[] | select(.name == $n) | .digest // ""
    ' | head -1
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c '
import json, sys
data = json.loads(sys.stdin.read())
name = sys.argv[1]
for a in data.get("assets", []):
    if a.get("name") == name:
        print(a.get("digest") or "")
        sys.exit(0)
' "$name"
  fi
}

# Resolve the release metadata first (this is the key fix)
DESIRED="${VERSION:-latest}"
info "Resolving release ${DESIRED}..."
RELEASE_JSON=$(get_release_json "$DESIRED")

if [ -z "$RELEASE_JSON" ] || echo "$RELEASE_JSON" | grep -q '"Not Found"'; then
  error "release ${DESIRED} not found (check https://github.com/${REPO}/releases)"
fi

VERSION=$(echo "$RELEASE_JSON" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4 || true)
[ -z "$VERSION" ] && error "could not determine tag_name from release JSON"

ASSET_NAME=$(find_asset_name "$RELEASE_JSON" | tr -d '\r\n')
[ -z "$ASSET_NAME" ] && error "no archive for ${OS}/${ARCH} found in release ${VERSION}"

DOWNLOAD_URL=$(find_download_url "$RELEASE_JSON" "$ASSET_NAME")
[ -z "$DOWNLOAD_URL" ] && DOWNLOAD_URL="${GITHUB}/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"

DIGEST=$(find_digest "$RELEASE_JSON" "$ASSET_NAME")

info "Installing ${BINARY} ${VERSION} for ${OS}/${ARCH}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

CHECKSUM_ASSET="checksums.txt"
CHECKSUM_URL="${GITHUB}/${REPO}/releases/download/${VERSION}/${CHECKSUM_ASSET}"

info "Downloading ${ASSET_NAME}..."

if command -v curl >/dev/null 2>&1; then
  curl -fsSL --retry 3 -o "${TMPDIR}/${ASSET_NAME}" "$DOWNLOAD_URL" \
    || error "download failed: $DOWNLOAD_URL"
  if [ "${VERIFY:-0}" = "1" ]; then
    curl -fsSL --retry 3 -o "${TMPDIR}/${CHECKSUM_ASSET}" "$CHECKSUM_URL" 2>/dev/null || true
  fi
elif command -v wget >/dev/null 2>&1; then
  wget -q --tries=3 -O "${TMPDIR}/${ASSET_NAME}" "$DOWNLOAD_URL" \
    || error "download failed: $DOWNLOAD_URL"
  if [ "${VERIFY:-0}" = "1" ]; then
    wget -q --tries=3 -O "${TMPDIR}/${CHECKSUM_ASSET}" "$CHECKSUM_URL" 2>/dev/null || true
  fi
else
  error "curl or wget is required"
fi

# Verify (prefer GitHub digest from release metadata, then checksums.txt)
if [ "${VERIFY:-0}" = "1" ]; then
  info "Verifying checksum..."
  verified=0
  if [ -n "${DIGEST:-}" ] && command -v sha256sum >/dev/null 2>&1; then
    # DIGEST is like "sha256:abc..."
    want=$(echo "$DIGEST" | sed 's/^sha256://')
    have=$(sha256sum "${TMPDIR}/${ASSET_NAME}" | awk '{print $1}')
    if [ "$want" = "$have" ]; then
      verified=1
      info "Checksum verified against GitHub release digest"
    else
      error "checksum mismatch (digest from GitHub)"
    fi
  elif [ -n "${DIGEST:-}" ] && command -v shasum >/dev/null 2>&1; then
    want=$(echo "$DIGEST" | sed 's/^sha256://')
    have=$(shasum -a 256 "${TMPDIR}/${ASSET_NAME}" | awk '{print $1}')
    if [ "$want" = "$have" ]; then
      verified=1
      info "Checksum verified against GitHub release digest"
    else
      error "checksum mismatch (digest from GitHub)"
    fi
  elif [ -f "${TMPDIR}/${CHECKSUM_ASSET}" ]; then
    pushd "$TMPDIR" >/dev/null
    if command -v sha256sum >/dev/null 2>&1; then
      if grep " ${ASSET_NAME}\$" "${CHECKSUM_ASSET}" | sha256sum -c - >/dev/null; then
        verified=1
        info "Checksum verified against checksums.txt"
      fi
    elif command -v shasum >/dev/null 2>&1; then
      if grep " ${ASSET_NAME}\$" "${CHECKSUM_ASSET}" | shasum -a 256 -c - >/dev/null; then
        verified=1
        info "Checksum verified against checksums.txt"
      fi
    fi
    popd >/dev/null
  fi

  if [ "$verified" != "1" ]; then
    if [ "${VERIFY:-0}" = "1" ]; then
      error "checksum verification failed (no sha256sum/shasum or no matching digest)"
    else
      warn "could not verify (no sha256sum/shasum available)"
    fi
  fi
fi

# Extract
info "Extracting..."
tar -xzf "${TMPDIR}/${ASSET_NAME}" -C "$TMPDIR" \
  || error "failed to extract archive"

BIN_PATH="${TMPDIR}/${BINARY}"
[ -f "$BIN_PATH" ] || error "binary not found in archive"

# macOS: remove quarantine attribute
if [ "$OS" = "darwin" ]; then
  if command -v xattr >/dev/null 2>&1; then
    xattr -dr com.apple.quarantine "$BIN_PATH" 2>/dev/null || true
  fi
fi

chmod +x "$BIN_PATH"

# Determine install prefix
if [ -z "${PREFIX:-}" ]; then
  # Prefer order:
  # 1. /usr/local/bin (if writable or sudo available)
  # 2. $HOME/.local/bin
  # 3. $HOME/bin
  if [ -w /usr/local/bin ] || ( [ ! -e /usr/local/bin ] && [ -w /usr/local ] ); then
    PREFIX="/usr/local/bin"
  elif [ -n "${HOME:-}" ]; then
    for d in "$HOME/.local/bin" "$HOME/bin"; do
      if [ -w "$d" ] 2>/dev/null || [ ! -e "$d" ] 2>/dev/null; then
        PREFIX="$d"
        break
      fi
    done
  fi
  [ -z "${PREFIX:-}" ] && PREFIX="/usr/local/bin"
fi

DEST="${PREFIX}/${BINARY}"

# Ensure prefix directory exists
if [ ! -d "$PREFIX" ]; then
  info "Creating directory ${PREFIX}"
  if ! mkdir -p "$PREFIX" 2>/dev/null; then
    if command -v sudo >/dev/null 2>&1; then
      sudo mkdir -p "$PREFIX" || error "failed to create $PREFIX"
    else
      error "cannot create $PREFIX (no write permission and sudo not available)"
    fi
  fi
fi

# Install binary
info "Installing to ${DEST}"
if ! mv "$BIN_PATH" "$DEST" 2>/dev/null; then
  if command -v sudo >/dev/null 2>&1; then
    sudo mv "$BIN_PATH" "$DEST" || error "failed to move binary to $DEST"
  else
    error "cannot write to $DEST (try --prefix or run with appropriate permissions)"
  fi
fi

# macOS quarantine on final location too (in case mv/sudo affected it)
if [ "$OS" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
  sudo xattr -dr com.apple.quarantine "$DEST" 2>/dev/null || xattr -dr com.apple.quarantine "$DEST" 2>/dev/null || true
fi

info "${BINARY} installed successfully!"

# Check PATH
INSTALLED_DIR=$(cd "$(dirname "$DEST")" && pwd -P 2>/dev/null || dirname "$DEST")
case ":$PATH:" in
  *":${INSTALLED_DIR}:"* | *":${PREFIX}:"* )
    echo
    echo -e "${GREEN}${BINARY} is ready.${RESET} Run: ${BOLD}${BINARY} version${RESET}"
    ;;
  *)
    echo
    warn "${PREFIX} is not in your PATH."
    echo
    echo "Add it to your shell configuration:"
    echo
    case "$(basename "${SHELL:-sh}")" in
      fish)
        echo "  fish_add_path ${PREFIX}"
        echo "  # or: set -U fish_user_paths ${PREFIX} \$fish_user_paths"
        ;;
      zsh)
        echo "  echo 'export PATH=\"${PREFIX}:\$PATH\"' >> ~/.zshrc"
        echo "  source ~/.zshrc"
        ;;
      bash)
        echo "  echo 'export PATH=\"${PREFIX}:\$PATH\"' >> ~/.bashrc"
        echo "  source ~/.bashrc"
        ;;
      *)
        echo "  export PATH=\"${PREFIX}:\$PATH\""
        ;;
    esac
    echo
    echo "Then run: ${BOLD}${BINARY} version${RESET}"
    ;;
esac

# Post-install hint
echo
echo "Next steps:"
echo "  ${BINARY} config set api-url https://api.builder-hub.dev"
echo "  ${BINARY} auth login"
