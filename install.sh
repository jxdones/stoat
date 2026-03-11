#!/usr/bin/env sh
# Install Stoat — run with: curl -fsSL https://raw.githubusercontent.com/jxdones/stoat/main/install.sh | sh
# Options: pass a version (e.g. v0.2.1) as first argument. Set BINDIR to choose install location when using go install (default: $HOME/go/bin).

set -e

VERSION="${1:-latest}"
BINDIR="${BINDIR:-$HOME/go/bin}"
REPO="github.com/jxdones/stoat"

if ! command -v go >/dev/null 2>&1; then
  echo "Stoat install requires Go. Install Go from https://go.dev/dl/ then run this script again."
  exit 1
fi

echo "Installing stoat@${VERSION}..."
go install "${REPO}@${VERSION}"

if [ -x "${BINDIR}/stoat" ]; then
  echo "Stoat installed to ${BINDIR}/stoat"
  if ! echo ":${PATH}:" | grep -q ":${BINDIR}:"; then
    echo "Add ${BINDIR} to your PATH, e.g.:"
    echo "  export PATH=\"\${PATH}:${BINDIR}\""
    echo "Add the above to your shell profile (~/.bashrc, ~/.zshrc, etc.) to make it permanent."
  fi
else
  echo "Install completed. Ensure ${BINDIR} (or \$GOBIN) is in your PATH."
fi
