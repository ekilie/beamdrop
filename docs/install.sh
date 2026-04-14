#!/bin/sh
# install.sh — Installer for BeamDrop
# https://github.com/ekilie/beamdrop
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ekilie/beamdrop/main/docs/install.sh | sh
#
# Or download and inspect first (recommended):
#   curl -fsSL https://raw.githubusercontent.com/ekilie/beamdrop/main/docs/install.sh -o install.sh
#   less install.sh
#   sh install.sh
#
# Options (environment variables):
#   BEAMDROP_VERSION   - Specific version to install (default: latest)
#   BEAMDROP_INSTALL   - Installation directory      (default: /usr/local/bin)

set -eu

# ── Configuration ─────────────────────────────────────────────────────────────
REPO="ekilie/beamdrop"
BINARY_NAME="beamdrop"
INSTALL_DIR="${BEAMDROP_INSTALL:-/usr/local/bin}"
VERSION="${BEAMDROP_VERSION:-latest}"
GITHUB_BASE="https://github.com/${REPO}"
CHECKSUM_FILE="checksums.txt"

# ── Helpers ───────────────────────────────────────────────────────────────────
info()  { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn()  { printf '\033[1;33mWARN:\033[0m %s\n' "$1"; }
error() { printf '\033[1;31mERROR:\033[0m %s\n' "$1" >&2; exit 1; }

need_cmd() {
    if ! command -v "$1" > /dev/null 2>&1; then
        error "Required command '$1' not found. Please install it and try again."
    fi
}

# ── Detect OS ─────────────────────────────────────────────────────────────────
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux"  ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported operating system: $(uname -s). Use Windows zip from the releases page." ;;
    esac
}

# ── Detect Architecture ──────────────────────────────────────────────────────
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)       echo "amd64" ;;
        aarch64|arm64)     echo "arm64" ;;
        *)                  error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# ── Resolve version tag ──────────────────────────────────────────────────────
resolve_version() {
    if [ "$VERSION" = "latest" ]; then
        # Follow the redirect from /releases/latest to get the tag
        VERSION=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "${GITHUB_BASE}/releases/latest" | sed 's|.*/||')
        if [ -z "$VERSION" ]; then
            error "Could not determine latest version. Set BEAMDROP_VERSION explicitly."
        fi
        info "Latest version: ${VERSION}"
    else
        info "Requested version: ${VERSION}"
    fi
}

# ── Download helper (curl or wget) ───────────────────────────────────────────
download() {
    url="$1"
    dest="$2"
    if command -v curl > /dev/null 2>&1; then
        curl -fSL --retry 3 --retry-delay 2 -o "$dest" "$url"
    elif command -v wget > /dev/null 2>&1; then
        wget -q -O "$dest" "$url"
    else
        error "Neither curl nor wget found. Please install one and try again."
    fi
}

# ── Verify checksum ──────────────────────────────────────────────────────────
verify_checksum() {
    archive_path="$1"
    archive_name="$2"
    checksums_path="$3"

    # Extract expected checksum for this archive
    expected=$(grep "  ${archive_name}\$" "$checksums_path" | awk '{print $1}')
    if [ -z "$expected" ]; then
        # Try alternate format (single space separator)
        expected=$(grep " ${archive_name}\$" "$checksums_path" | awk '{print $1}')
    fi

    if [ -z "$expected" ]; then
        warn "No checksum found for ${archive_name} in checksums.txt — skipping verification."
        return 0
    fi

    # Compute actual checksum
    if command -v sha256sum > /dev/null 2>&1; then
        actual=$(sha256sum "$archive_path" | awk '{print $1}')
    elif command -v shasum > /dev/null 2>&1; then
        actual=$(shasum -a 256 "$archive_path" | awk '{print $1}')
    else
        warn "Neither sha256sum nor shasum found — skipping checksum verification."
        return 0
    fi

    if [ "$expected" != "$actual" ]; then
        error "Checksum mismatch for ${archive_name}!
  Expected: ${expected}
  Got:      ${actual}
This could indicate a corrupted download or a tampered file. Aborting."
    fi

    info "Checksum verified (SHA-256)."
}

# ── Main ──────────────────────────────────────────────────────────────────────
main() {
    info "BeamDrop Installer"
    echo ""

    # Check for required commands
    need_cmd uname
    need_cmd tar

    OS=$(detect_os)
    ARCH=$(detect_arch)
    info "Detected platform: ${OS}/${ARCH}"

    # We need curl or wget
    if ! command -v curl > /dev/null 2>&1 && ! command -v wget > /dev/null 2>&1; then
        error "Neither curl nor wget found. Please install one and try again."
    fi

    resolve_version

    ARCHIVE_NAME="${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
    DOWNLOAD_URL="${GITHUB_BASE}/releases/download/${VERSION}/${ARCHIVE_NAME}"
    CHECKSUMS_URL="${GITHUB_BASE}/releases/download/${VERSION}/${CHECKSUM_FILE}"

    # Create a temporary directory and ensure cleanup
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

    # Download checksums
    info "Downloading checksums..."
    download "$CHECKSUMS_URL" "${TMP_DIR}/${CHECKSUM_FILE}" || warn "Could not download checksums.txt"

    # Download archive
    info "Downloading ${ARCHIVE_NAME}..."
    download "$DOWNLOAD_URL" "${TMP_DIR}/${ARCHIVE_NAME}"

    # Verify checksum
    if [ -f "${TMP_DIR}/${CHECKSUM_FILE}" ]; then
        verify_checksum "${TMP_DIR}/${ARCHIVE_NAME}" "${ARCHIVE_NAME}" "${TMP_DIR}/${CHECKSUM_FILE}"
    else
        warn "Checksum file not available — skipping verification."
    fi

    # Extract
    info "Extracting..."
    tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "${TMP_DIR}"

    # Verify the binary exists after extraction
    if [ ! -f "${TMP_DIR}/${BINARY_NAME}" ]; then
        error "Expected binary '${BINARY_NAME}' not found in archive."
    fi

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        info "Elevated permissions required to install to ${INSTALL_DIR}"
        sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    # Verify installation
    if command -v "$BINARY_NAME" > /dev/null 2>&1; then
        echo ""
        info "BeamDrop ${VERSION} installed successfully!"
        info "Location: ${INSTALL_DIR}/${BINARY_NAME}"
        echo ""
        echo "  Get started:"
        echo "    beamdrop              # serve current directory on :7777"
        echo "    beamdrop -p secret    # with password protection"
        echo "    beamdrop --help       # all options"
        echo ""
    else
        echo ""
        info "BeamDrop ${VERSION} installed to ${INSTALL_DIR}/${BINARY_NAME}"
        warn "${INSTALL_DIR} may not be in your PATH. Add it with:"
        echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
    fi
}

main "$@"
