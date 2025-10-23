#!/usr/bin/env bash

set -e

# Sloth Kubernetes Installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/chalkan3/sloth-kubernetes/main/install.sh | bash
#   ./install.sh                    # Install latest version
#   ./install.sh v1.0.0             # Install specific version

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
REPO="chalkan3/sloth-kubernetes"
BINARY_NAME="sloth-kubernetes"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Functions
print_header() {
    echo ""
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}  ${BOLD}🦥 Sloth Kubernetes Installer${NC}                          ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Detect OS and Architecture
detect_platform() {
    local os=""
    local arch=""

    # Detect OS
    case "$(uname -s)" in
        Linux*)
            os="Linux"
            ;;
        Darwin*)
            os="Darwin"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            os="Windows"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect Architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="x86_64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        armv7l)
            arch="armv7"
            ;;
        i386|i686)
            arch="i386"
            ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get latest release version
get_latest_version() {
    local version=""

    print_info "Fetching latest version..."

    # Try using GitHub API
    if command -v curl >/dev/null 2>&1; then
        version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    if [ -z "$version" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi

    echo "$version"
}

# Download and install binary
install_binary() {
    local version="$1"
    local platform="$2"
    local os=$(echo "$platform" | cut -d'_' -f1)
    local arch=$(echo "$platform" | cut -d'_' -f2)

    # Construct download URL
    local archive_ext="tar.gz"
    if [ "$os" = "Windows" ]; then
        archive_ext="zip"
    fi

    local archive_name="${BINARY_NAME}_${version}_${os}_${arch}.${archive_ext}"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"

    print_info "Downloading ${BINARY_NAME} ${version} for ${os} ${arch}..."
    print_info "URL: ${download_url}"

    # Create temporary directory
    local tmp_dir=$(mktemp -d)
    trap "rm -rf ${tmp_dir}" EXIT

    # Download archive
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL -o "${tmp_dir}/${archive_name}" "${download_url}"; then
            print_error "Failed to download ${archive_name}"
            print_info "Please check if the release exists: https://github.com/${REPO}/releases"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q -O "${tmp_dir}/${archive_name}" "${download_url}"; then
            print_error "Failed to download ${archive_name}"
            print_info "Please check if the release exists: https://github.com/${REPO}/releases"
            exit 1
        fi
    fi

    print_success "Downloaded successfully"

    # Download and verify checksum
    print_info "Verifying checksum..."
    local checksum_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "${tmp_dir}/checksums.txt" "${checksum_url}" 2>/dev/null || true
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "${tmp_dir}/checksums.txt" "${checksum_url}" 2>/dev/null || true
    fi

    if [ -f "${tmp_dir}/checksums.txt" ]; then
        cd "${tmp_dir}"
        if command -v shasum >/dev/null 2>&1; then
            if grep "${archive_name}" checksums.txt | shasum -a 256 -c - >/dev/null 2>&1; then
                print_success "Checksum verified"
            else
                print_warning "Checksum verification failed, but continuing..."
            fi
        elif command -v sha256sum >/dev/null 2>&1; then
            if grep "${archive_name}" checksums.txt | sha256sum -c - >/dev/null 2>&1; then
                print_success "Checksum verified"
            else
                print_warning "Checksum verification failed, but continuing..."
            fi
        else
            print_warning "No checksum tool found (shasum/sha256sum), skipping verification"
        fi
    else
        print_warning "Checksums not available, skipping verification"
    fi

    # Extract archive
    print_info "Extracting archive..."
    cd "${tmp_dir}"

    if [ "$archive_ext" = "tar.gz" ]; then
        tar -xzf "${archive_name}"
    elif [ "$archive_ext" = "zip" ]; then
        unzip -q "${archive_name}"
    fi

    if [ ! -f "${BINARY_NAME}" ]; then
        print_error "Binary not found in archive"
        exit 1
    fi

    print_success "Extracted successfully"

    # Install binary
    print_info "Installing to ${INSTALL_DIR}..."

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        print_warning "Requires sudo to install to ${INSTALL_DIR}"
        sudo mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    print_success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Verify installation
verify_installation() {
    print_info "Verifying installation..."

    if command -v "${BINARY_NAME}" >/dev/null 2>&1; then
        local installed_version=$("${BINARY_NAME}" --version 2>/dev/null || echo "unknown")
        print_success "${BINARY_NAME} installed successfully!"
        print_info "Version: ${installed_version}"
        return 0
    else
        print_error "Installation verification failed"
        print_warning "Binary installed but not found in PATH"
        print_info "Please add ${INSTALL_DIR} to your PATH:"
        echo ""
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        return 1
    fi
}

# Print usage information
print_usage() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}  ${BOLD}✨ Installation Complete!${NC}                               ${GREEN}║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}Quick Start:${NC}"
    echo ""
    echo -e "  ${CYAN}1.${NC} Get help and see available commands:"
    echo -e "     ${BLUE}${BINARY_NAME} --help${NC}"
    echo ""
    echo -e "  ${CYAN}2.${NC} Deploy a Kubernetes cluster:"
    echo -e "     ${BLUE}${BINARY_NAME} deploy production${NC}"
    echo ""
    echo -e "  ${CYAN}3.${NC} Join the VPN to access your cluster:"
    echo -e "     ${BLUE}${BINARY_NAME} vpn join production --install${NC}"
    echo ""
    echo -e "  ${CYAN}4.${NC} View VPN peers:"
    echo -e "     ${BLUE}${BINARY_NAME} vpn peers production${NC}"
    echo ""
    echo -e "${BOLD}Other Useful Commands:${NC}"
    echo -e "  ${BLUE}${BINARY_NAME} stacks list${NC}              # List all stacks"
    echo -e "  ${BLUE}${BINARY_NAME} nodes list <stack>${NC}       # List cluster nodes"
    echo -e "  ${BLUE}${BINARY_NAME} destroy <stack>${NC}          # Destroy a cluster"
    echo ""
    echo -e "${BOLD}Documentation:${NC}"
    echo -e "  📚 README: ${BLUE}https://github.com/${REPO}${NC}"
    echo -e "  📝 Examples: ${BLUE}https://github.com/${REPO}/tree/main/examples${NC}"
    echo ""
}

# Main installation flow
main() {
    local requested_version="$1"

    print_header

    # Check if already installed
    if command -v "${BINARY_NAME}" >/dev/null 2>&1; then
        local current_version=$("${BINARY_NAME}" --version 2>/dev/null | head -n1 || echo "unknown")
        print_warning "${BINARY_NAME} is already installed"
        print_info "Current version: ${current_version}"
        echo ""

        # Only prompt if running interactively
        if [ -t 0 ]; then
            read -p "Do you want to reinstall/upgrade? (y/N): " -n 1 -r
            echo ""
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                print_info "Installation cancelled"
                exit 0
            fi
        else
            print_info "Running in non-interactive mode, proceeding with installation..."
        fi
    fi

    # Detect platform
    local platform=$(detect_platform)
    print_success "Detected platform: $platform"

    # Get version to install
    local version
    if [ -n "$requested_version" ]; then
        version="$requested_version"
        print_success "Installing requested version: $version"
    else
        version=$(get_latest_version)
        print_success "Installing latest version: $version"
    fi

    # Install binary
    install_binary "$version" "$platform"

    # Verify installation
    if verify_installation; then
        print_usage
    fi
}

# Run main function
main "$@"
