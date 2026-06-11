#!/bin/bash
set -euo pipefail

GITHUB_REPO="${AIDE_GITHUB_REPO:-matheus-meneses/aide}"
RELEASE_URL="${AIDE_RELEASE_URL:-https://github.com/${GITHUB_REPO}/releases/latest/download}"
VERSION="${AIDE_VERSION:-latest}"
INSTALL_DIR="${HOME}/.local/bin"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin|linux) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

BINARY="aide_${OS}_${ARCH}"

if [ "$VERSION" != "latest" ]; then
    RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}"
fi

mkdir -p "$INSTALL_DIR"

if [ -n "${AIDE_LOCAL_BIN:-}" ]; then
    echo "Installing aide (${OS}/${ARCH}) from local binary ${AIDE_LOCAL_BIN}..."
    cp "${AIDE_LOCAL_BIN}" "${INSTALL_DIR}/aide"
else
    echo "Installing aide (${OS}/${ARCH})..."
    if token="$(gh auth token 2>/dev/null)" && [ -n "$token" ]; then
        curl -fL -H "Authorization: token ${token}" "${RELEASE_URL}/${BINARY}" -o "${INSTALL_DIR}/aide"
    else
        curl -fL "${RELEASE_URL}/${BINARY}" -o "${INSTALL_DIR}/aide"
    fi
fi
chmod +x "${INSTALL_DIR}/aide"

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

echo "aide installed to ${INSTALL_DIR}/aide"
echo ""
echo "Run 'aide init' to complete setup."
