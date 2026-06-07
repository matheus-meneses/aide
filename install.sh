#!/bin/bash
set -euo pipefail

NEXUS_URL="${AIDE_NEXUS_URL:-https://nexus.sharedservices.local/repository/aide}"
VERSION="${AIDE_VERSION:-latest}"
INSTALL_DIR="${HOME}/.local/bin"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ "$OS" != "darwin" ]; then
    echo "Unsupported OS: $OS (only macOS is supported)"
    exit 1
fi

BINARY="aide-${OS}-${ARCH}"

if [ "$VERSION" = "latest" ]; then
    echo "Fetching latest version..."
    VERSION=$(curl -sfL "${NEXUS_URL}/VERSION" 2>/dev/null || echo "1.0.0")
fi

echo "Installing aide ${VERSION} (${OS}/${ARCH})..."

mkdir -p "$INSTALL_DIR"

curl -fL "${NEXUS_URL}/${VERSION}/${BINARY}" -o "${INSTALL_DIR}/aide"
chmod +x "${INSTALL_DIR}/aide"

mkdir -p "${HOME}/.aide"
echo "Downloading source registry..."
curl -fsSL "${NEXUS_URL}/${VERSION}/registry.yaml" -o "${HOME}/.aide/registry.yaml" 2>/dev/null || true

add_to_path() {
    local profile="$1"
    if [ -f "$profile" ]; then
        if ! grep -q '\.local/bin' "$profile" 2>/dev/null; then
            echo '' >> "$profile"
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$profile"
            echo "Updated $profile with PATH"
            return 0
        fi
        return 0
    fi
    return 1
}

if ! echo "$PATH" | grep -q "$HOME/.local/bin"; then
    added=false
    for profile in "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.bashrc" "$HOME/.profile"; do
        if add_to_path "$profile"; then
            added=true
            break
        fi
    done
    if [ "$added" = false ]; then
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.profile"
        echo "Created $HOME/.profile with PATH"
    fi
    export PATH="$HOME/.local/bin:$PATH"
fi

echo "aide ${VERSION} installed to ${INSTALL_DIR}/aide"
echo ""
echo "Run 'aide init' to complete setup."
