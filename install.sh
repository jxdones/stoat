#!/usr/bin/env sh
# Install Stoat — run with: curl -fsSL https://raw.githubusercontent.com/jxdones/stoat/main/install.sh | sh
# Options: pass a version (e.g. v0.5.2) as first argument. Set BINDIR to choose install location (default: /usr/local/bin).

set -e

VERSION="${1:-latest}"
BINDIR="${BINDIR:-/usr/local/bin}"
REPO="jxdones/stoat"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  darwin) ;;
  linux)  ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Resolve "latest" to a concrete tag so we can build the versioned archive name
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  if [ -z "$VERSION" ]; then
    echo "Failed to resolve latest version from GitHub API."
    exit 1
  fi
fi

# goreleaser uses the version without the leading 'v' in the archive name
SEMVER="${VERSION#v}"
ARCHIVE="stoat_${SEMVER}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "Installing stoat ${VERSION} (${OS}/${ARCH})..."

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o /tmp/stoat.tar.gz
elif command -v wget >/dev/null 2>&1; then
  wget -qO /tmp/stoat.tar.gz "$URL"
else
  echo "curl or wget is required to install stoat."
  exit 1
fi

tar -xzf /tmp/stoat.tar.gz -C /tmp stoat
rm /tmp/stoat.tar.gz

if [ -w "$BINDIR" ]; then
  mv /tmp/stoat "${BINDIR}/stoat"
else
  echo "Installing to ${BINDIR} requires sudo..."
  sudo mv /tmp/stoat "${BINDIR}/stoat"
fi

echo "Stoat installed to ${BINDIR}/stoat"
if ! echo ":${PATH}:" | grep -q ":${BINDIR}:"; then
  echo "Add ${BINDIR} to your PATH, e.g.:"
  echo "  export PATH=\"\${PATH}:${BINDIR}\""
  echo "Add the above to your shell profile (~/.bashrc, ~/.zshrc, etc.) to make it permanent."
fi
