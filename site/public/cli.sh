#!/bin/sh
#
# vStats CLI Installer
# https://vstats.zsoft.cc
#
# This script detects the current operating system and installs
# vStats CLI according to that OS's conventions.
#
# Environment variables:
#   VSTATS_VERSION: Pin to a specific version (e.g., "1.0.0")
#   VSTATS_INSTALL_DIR: Custom install directory (default: /usr/local/bin)
#
# Examples:
#   curl -fsSL https://vstats.zsoft.cc/cli.sh | sh
#   curl -fsSL https://vstats.zsoft.cc/cli.sh | VSTATS_VERSION=1.0.0 sh
#   wget -qO- https://vstats.zsoft.cc/cli.sh | sh
#

set -eu

# All the code is wrapped in a main function that gets called at the
# bottom of the file, so that a truncated partial download doesn't end
# up executing half a script.
main() {
    # Step 1: detect the current OS, version, and architecture
    OS=""
    ARCH=""
    PACKAGETYPE=""
    VERSION=""
    
    # Detect OS
    case "$(uname -s)" in
        Linux)
            OS="linux"
            ;;
        Darwin)
            OS="darwin"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            OS="windows"
            ;;
        FreeBSD)
            OS="freebsd"
            ;;
        *)
            echo "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armv7)
            ARCH="arm"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        *)
            echo "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    # Detect Linux distro and package type
    if [ "$OS" = "linux" ]; then
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            case "$ID" in
                ubuntu|debian|raspbian|linuxmint|pop|elementary|zorin|kali)
                    PACKAGETYPE="apt"
                    VERSION="$VERSION_ID"
                    ;;
                fedora)
                    PACKAGETYPE="dnf"
                    VERSION="$VERSION_ID"
                    ;;
                centos|rhel|rocky|almalinux|ol)
                    if [ "${VERSION_ID%%.*}" -ge 8 ]; then
                        PACKAGETYPE="dnf"
                    else
                        PACKAGETYPE="yum"
                    fi
                    VERSION="$VERSION_ID"
                    ;;
                amzn)
                    PACKAGETYPE="yum"
                    VERSION="$VERSION_ID"
                    ;;
                arch|manjaro|endeavouros)
                    PACKAGETYPE="pacman"
                    ;;
                opensuse*|sles)
                    PACKAGETYPE="zypper"
                    VERSION="$VERSION_ID"
                    ;;
                alpine)
                    PACKAGETYPE="apk"
                    VERSION="$VERSION_ID"
                    ;;
                *)
                    PACKAGETYPE="binary"
                    ;;
            esac
        else
            PACKAGETYPE="binary"
        fi
    elif [ "$OS" = "darwin" ]; then
        # Check if Homebrew is installed
        if command -v brew >/dev/null 2>&1; then
            PACKAGETYPE="brew"
        else
            PACKAGETYPE="binary"
        fi
    else
        PACKAGETYPE="binary"
    fi
    
    echo ""
    echo "╔═══════════════════════════════════════════════════╗"
    echo "║        vStats CLI - Installation Script           ║"
    echo "╚═══════════════════════════════════════════════════╝"
    echo ""
    echo "  Detected: $OS/$ARCH"
    echo "  Package type: $PACKAGETYPE"
    echo ""
    
    # Step 2: Install based on package type
    case "$PACKAGETYPE" in
        apt)
            install_apt
            ;;
        dnf|yum)
            install_rpm "$PACKAGETYPE"
            ;;
        brew)
            install_brew
            ;;
        pacman)
            install_pacman
            ;;
        apk)
            install_apk
            ;;
        *)
            install_binary
            ;;
    esac
    
    # Step 3: Verify installation
    echo ""
    if command -v vstats >/dev/null 2>&1; then
        echo "✓ vStats CLI installed successfully!"
        echo ""
        vstats version
        echo ""
        echo "Quick Start:"
        echo "  vstats login              # Login to vStats Cloud"
        echo "  vstats server list        # List your servers"
        echo "  vstats server create web1 # Create a new server"
        echo ""
        echo "Documentation: https://vstats.zsoft.cc/docs/cli"
    else
        echo "✗ Installation may have failed. Please check for errors above."
        exit 1
    fi
}

# Install via APT (Debian/Ubuntu)
install_apt() {
    echo "Installing vStats CLI via APT..."
    
    # Check for root/sudo
    SUDO=""
    if [ "$(id -u)" -ne 0 ]; then
        if command -v sudo >/dev/null 2>&1; then
            SUDO="sudo"
        else
            echo "Error: This script requires root privileges. Please run with sudo."
            exit 1
        fi
    fi
    
    # Install prerequisites
    $SUDO apt-get update -qq
    $SUDO apt-get install -y -qq curl gnupg apt-transport-https
    
    # Add GPG key
    echo "Adding vStats repository key..."
    curl -fsSL https://vstats.zsoft.cc/gpg | $SUDO gpg --dearmor -o /usr/share/keyrings/vstats-archive-keyring.gpg
    
    # Add repository
    echo "Adding vStats repository..."
    echo "deb [signed-by=/usr/share/keyrings/vstats-archive-keyring.gpg] https://repo.vstats.zsoft.cc/apt stable main" | \
        $SUDO tee /etc/apt/sources.list.d/vstats.list > /dev/null
    
    # Install
    echo "Installing vstats..."
    $SUDO apt-get update -qq
    $SUDO apt-get install -y vstats
}

# Install via DNF/YUM (Fedora/RHEL/CentOS)
install_rpm() {
    PKG_MGR="$1"
    echo "Installing vStats CLI via $PKG_MGR..."
    
    # Check for root/sudo
    SUDO=""
    if [ "$(id -u)" -ne 0 ]; then
        if command -v sudo >/dev/null 2>&1; then
            SUDO="sudo"
        else
            echo "Error: This script requires root privileges. Please run with sudo."
            exit 1
        fi
    fi
    
    # Add repository
    echo "Adding vStats repository..."
    $SUDO tee /etc/yum.repos.d/vstats.repo > /dev/null << 'EOF'
[vstats]
name=vStats Repository
baseurl=https://repo.vstats.zsoft.cc/yum/$basearch
enabled=1
gpgcheck=1
gpgkey=https://vstats.zsoft.cc/gpg
EOF

    # Install
    echo "Installing vstats..."
    $SUDO $PKG_MGR install -y vstats
}

# Install via Homebrew (macOS/Linux)
install_brew() {
    echo "Installing vStats CLI via Homebrew..."
    
    # Add tap and install
    brew tap zsai001/vstats https://github.com/zsai001/homebrew-vstats 2>/dev/null || true
    brew install vstats
}

# Install via Pacman (Arch Linux)
install_pacman() {
    echo "Installing vStats CLI via AUR..."
    
    # Check for AUR helper
    if command -v yay >/dev/null 2>&1; then
        yay -S --noconfirm vstats-bin
    elif command -v paru >/dev/null 2>&1; then
        paru -S --noconfirm vstats-bin
    else
        echo "No AUR helper found. Installing binary version..."
        install_binary
    fi
}

# Install via APK (Alpine Linux)
install_apk() {
    echo "Installing vStats CLI via APK..."
    
    # Check for root/sudo
    SUDO=""
    if [ "$(id -u)" -ne 0 ]; then
        if command -v sudo >/dev/null 2>&1; then
            SUDO="sudo"
        else
            echo "Error: This script requires root privileges. Please run with sudo."
            exit 1
        fi
    fi
    
    # For Alpine, we use binary installation
    # as setting up a custom repo is more complex
    install_binary
}

# Install binary directly
install_binary() {
    echo "Installing vStats CLI binary..."
    
    INSTALL_DIR="${VSTATS_INSTALL_DIR:-/usr/local/bin}"
    GITHUB_REPO="zsai001/vstats"
    
    # Check for root/sudo for default install dir
    SUDO=""
    if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ "$(id -u)" -ne 0 ]; then
        if command -v sudo >/dev/null 2>&1; then
            SUDO="sudo"
        else
            # Fall back to user directory
            INSTALL_DIR="$HOME/.local/bin"
            mkdir -p "$INSTALL_DIR"
            echo "Installing to $INSTALL_DIR (add to PATH if needed)"
        fi
    fi
    
    # Determine version
    if [ -n "${VSTATS_VERSION:-}" ]; then
        VERSION="v$VSTATS_VERSION"
    else
        echo "Fetching latest version..."
        VERSION=$(curl -fsSL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" 2>/dev/null | \
            grep '"tag_name"' | head -1 | cut -d'"' -f4)
        if [ -z "$VERSION" ]; then
            echo "Could not determine latest version. Please try again or specify VSTATS_VERSION."
            exit 1
        fi
    fi
    
    echo "Version: $VERSION"
    
    # Construct download URL
    BINARY_NAME="vstats-cli-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi
    
    DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/$BINARY_NAME"
    
    echo "Downloading from: $DOWNLOAD_URL"
    
    # Download
    TMP_FILE=$(mktemp)
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$DOWNLOAD_URL" -O "$TMP_FILE"
    else
        echo "Error: curl or wget is required"
        exit 1
    fi
    
    # Install
    chmod +x "$TMP_FILE"
    $SUDO mv "$TMP_FILE" "$INSTALL_DIR/vstats"
    
    # Add to PATH reminder for non-standard location
    if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
        if ! echo "$PATH" | grep -q "$HOME/.local/bin"; then
            echo ""
            echo "Note: Add $HOME/.local/bin to your PATH:"
            echo '  export PATH="$HOME/.local/bin:$PATH"'
            echo ""
            echo "Add this line to your ~/.bashrc or ~/.zshrc"
        fi
    fi
}

# Run main function
main "$@"
