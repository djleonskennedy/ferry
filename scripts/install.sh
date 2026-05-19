#!/usr/bin/env bash
#
# Build ferry with production-style ldflags (-trimpath, -s -w, version stamping)
# and install it into a directory on PATH. Idempotent.
#
# Usage:
#   ./scripts/install.sh                       # installs to $HOME/.local/bin
#   INSTALL_DIR=/usr/local/bin ./scripts/install.sh   # may require sudo
#   ./scripts/install.sh --no-path             # skip the PATH-rc edit
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
EDIT_PATH=1
for arg in "$@"; do
  case "$arg" in
    --no-path) EDIT_PATH=0 ;;
    -h|--help)
      sed -n '2,11p' "$0"
      exit 0
      ;;
    *)
      echo "unknown flag: $arg" >&2
      exit 2
      ;;
  esac
done

cd "$REPO_ROOT"

# --- preflight ----------------------------------------------------------------
if ! command -v go >/dev/null 2>&1; then
  echo "ferry install: go toolchain not found on PATH" >&2
  exit 1
fi

GOVER="$(go env GOVERSION 2>/dev/null || true)"
echo "→ go: ${GOVER:-unknown}"

# --- version stamping ---------------------------------------------------------
VERSION="$(git -C "$REPO_ROOT" describe --tags --always 2>/dev/null || echo dev)"
COMMIT="$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo none)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

LDFLAGS="-s -w \
  -X main.Version=${VERSION} \
  -X main.Commit=${COMMIT} \
  -X main.Date=${DATE}"

# --- build --------------------------------------------------------------------
echo "→ building ferry ${VERSION} (commit ${COMMIT})"
mkdir -p "$INSTALL_DIR"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
go build -trimpath -ldflags "$LDFLAGS" -o "$TMP/ferry" ./cmd/ferry

# --- install (atomic mv onto target) ------------------------------------------
DEST="$INSTALL_DIR/ferry"
if [ -e "$DEST" ] && ! [ -w "$DEST" ]; then
  echo "→ existing $DEST not writable; using sudo"
  sudo mv "$TMP/ferry" "$DEST"
elif ! [ -w "$INSTALL_DIR" ]; then
  echo "→ $INSTALL_DIR not writable; using sudo"
  sudo mv "$TMP/ferry" "$DEST"
else
  mv "$TMP/ferry" "$DEST"
fi
chmod 0755 "$DEST" 2>/dev/null || sudo chmod 0755 "$DEST"
echo "→ installed: $DEST"

# --- PATH check ---------------------------------------------------------------
case ":$PATH:" in
  *":$INSTALL_DIR:"*)
    echo "→ PATH already contains $INSTALL_DIR"
    ;;
  *)
    if [ "$EDIT_PATH" -eq 1 ]; then
      RC=""
      case "${SHELL##*/}" in
        zsh)  RC="$HOME/.zshrc" ;;
        bash) [ -f "$HOME/.bashrc" ]      && RC="$HOME/.bashrc"
              [ -z "$RC" ] && [ -f "$HOME/.bash_profile" ] && RC="$HOME/.bash_profile"
              [ -z "$RC" ] && RC="$HOME/.bashrc" ;;
        fish) RC="$HOME/.config/fish/config.fish" ;;
        *)    RC="$HOME/.profile" ;;
      esac
      LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
      [ "${SHELL##*/}" = "fish" ] && LINE="set -gx PATH $INSTALL_DIR \$PATH"

      mkdir -p "$(dirname "$RC")"
      touch "$RC"
      if ! grep -Fqs "$INSTALL_DIR" "$RC"; then
        printf '\n# Added by ferry install.sh\n%s\n' "$LINE" >> "$RC"
        echo "→ appended PATH entry to $RC"
        echo "  reload your shell:  source $RC"
      else
        echo "→ $RC already references $INSTALL_DIR (no changes)"
      fi
    else
      echo "→ $INSTALL_DIR is NOT on PATH (--no-path: skipping rc edit)"
      echo "  add this to your shell rc manually:"
      echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    ;;
esac

# --- verify -------------------------------------------------------------------
if command -v ferry >/dev/null 2>&1 && [ "$(command -v ferry)" = "$DEST" ]; then
  echo "→ $(ferry --version)"
else
  echo "→ installed binary: $($DEST --version)"
  echo "  (open a new shell or 'source' your rc to pick up ferry on PATH)"
fi
