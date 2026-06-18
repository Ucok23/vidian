#!/usr/bin/env bash
set -e

REPO="Ucok23/vidian"
BINARY="vidian"
INSTALL_DIR="/usr/local/bin"
TMP_DIR=$(mktemp -d)

# --- Colors ---
GREEN="\033[0;32m"
CYAN="\033[0;36m"
RED="\033[0;31m"
BOLD="\033[1m"
RESET="\033[0m"

print_step() { echo -e "${CYAN}→${RESET} $1"; }
print_ok()   { echo -e "${GREEN}✓${RESET} $1"; }
print_err()  { echo -e "${RED}✗ Error:${RESET} $1"; exit 1; }

echo ""
echo -e "${BOLD}  Vidian Installer${RESET}"
echo "  Lightweight read-only code viewer"
echo ""

# --- Detect OS and Architecture ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
  x86_64)       ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) print_err "Unsupported architecture: $ARCH" ;;
esac

case $OS in
  linux|darwin) ;;
  *) print_err "Unsupported OS: $OS. Please build from source." ;;
esac

ASSET_NAME="${BINARY}_${OS}_${ARCH}"

# --- Resolve version ---
if [ -z "$VIDIAN_VERSION" ]; then
  print_step "Fetching latest release..."
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    print_err "Could not determine latest version. Set VIDIAN_VERSION env var to override."
  fi
else
  VERSION="$VIDIAN_VERSION"
fi

print_step "Installing Vidian ${VERSION} (${OS}/${ARCH})..."

# --- Download ---
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}.tar.gz"
print_step "Downloading from: $DOWNLOAD_URL"

curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ASSET_NAME}.tar.gz" \
  || print_err "Download failed. Check https://github.com/${REPO}/releases for available versions."

tar -xzf "${TMP_DIR}/${ASSET_NAME}.tar.gz" -C "${TMP_DIR}"

# --- Install ---
print_step "Installing to ${INSTALL_DIR}/${BINARY}..."

if [ -w "$INSTALL_DIR" ]; then
  cp "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  chmod +x "${INSTALL_DIR}/${BINARY}"
else
  sudo cp "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  sudo chmod +x "${INSTALL_DIR}/${BINARY}"
fi

rm -rf "$TMP_DIR"

# --- Verify ---
if command -v "$BINARY" &>/dev/null; then
  print_ok "Vidian installed successfully!"
  echo ""
  echo -e "  ${BOLD}Usage:${RESET}"
  echo "    vidian .                    # open current directory"
  echo "    vidian ~/projects/my-app    # open any folder"
  echo ""
else
  echo ""
  echo -e "  ${RED}'${BINARY}' not found in PATH after install.${RESET}"
  echo "  Add ${INSTALL_DIR} to your PATH:"
  echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
  echo ""
fi
