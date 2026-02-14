#!/bin/sh
set -eu

# Install or update rig by downloading the latest release from GitHub.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ryan-rushton/rig/main/install.sh | sh
#   INSTALL_DIR=/usr/local/bin sh install.sh

REPO="ryan-rushton/rig"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

main() {
    detect_platform
    fetch_latest_tag
    download_and_install
    verify_install
    check_path
}

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) die "Unsupported architecture: $ARCH" ;;
    esac

    case "$OS" in
        darwin|linux) ;;
        *) die "Unsupported OS: $OS" ;;
    esac

    printf "Platform: %s/%s\n" "$OS" "$ARCH"
}

fetch_latest_tag() {
    printf "Fetching latest release...\n"
    TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | head -1 \
        | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

    if [ -z "$TAG" ]; then
        die "Could not determine latest release tag"
    fi

    printf "Latest version: %s\n" "$TAG"
}

download_and_install() {
    ARCHIVE="rig_${OS}_${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"
    TMP=$(mktemp -d)

    printf "Downloading %s...\n" "$URL"
    curl -fsSL "$URL" -o "${TMP}/${ARCHIVE}" || die "Download failed. Check that the release exists."

    tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP"

    mkdir -p "$INSTALL_DIR"
    mv "${TMP}/rig" "${INSTALL_DIR}/rig"
    chmod +x "${INSTALL_DIR}/rig"

    rm -rf "$TMP"

    printf "Installed to %s/rig\n" "$INSTALL_DIR"
}

verify_install() {
    VERSION=$("${INSTALL_DIR}/rig" --version 2>&1 || true)
    printf "%s\n" "$VERSION"
}

check_path() {
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            printf "\n"
            printf "WARNING: %s is not in your PATH.\n" "$INSTALL_DIR"
            printf "Add it by appending this to your shell profile (~/.zshrc or ~/.bashrc):\n"
            printf "\n"
            printf "  export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR"
            printf "\n"
            ;;
    esac
}

die() {
    printf "Error: %s\n" "$1" >&2
    exit 1
}

main
