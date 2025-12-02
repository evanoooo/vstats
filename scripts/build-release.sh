#!/bin/bash
#
# Build release binaries for vStats (agent and server) using Go
# Creates portable static binaries
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
RELEASE_DIR="$PROJECT_ROOT/releases"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info() { echo -e "${CYAN}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Check if Go is installed
if ! command -v go &> /dev/null; then
    error "Go is not installed. Please install Go 1.22 or later."
fi

# Detect current platform
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
    
    info "Building on: $OS-$ARCH"
}

# Build agent
build_agent() {
    local goos="$1"
    local goarch="$2"
    local output_name="$3"
    local version="${4:-dev}"
    
    info "Building agent for $goos/$goarch..."
    
    cd "$PROJECT_ROOT/agent-go"
    
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
        -ldflags "-X main.AgentVersion=$version" \
        -trimpath \
        -a -installsuffix cgo \
        -o "$RELEASE_DIR/$output_name" \
        .
    
    if [ -f "$RELEASE_DIR/$output_name" ]; then
        chmod +x "$RELEASE_DIR/$output_name"
        success "Built: $output_name ($(du -h "$RELEASE_DIR/$output_name" | cut -f1))"
    else
        error "Build failed for $goos/$goarch"
    fi
}

# Build server
build_server() {
    local goos="$1"
    local goarch="$2"
    local output_name="$3"
    local version="${4:-dev}"
    
    info "Building server for $goos/$goarch..."
    
    cd "$PROJECT_ROOT/server-go"
    
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
        -ldflags "-X main.ServerVersion=$version" \
        -trimpath \
        -a -installsuffix cgo \
        -o "$RELEASE_DIR/$output_name" \
        .
    
    if [ -f "$RELEASE_DIR/$output_name" ]; then
        chmod +x "$RELEASE_DIR/$output_name"
        success "Built: $output_name ($(du -h "$RELEASE_DIR/$output_name" | cut -f1))"
    else
        error "Build failed for $goos/$goarch"
    fi
}

# Build all targets for current platform
build_native() {
    detect_platform
    mkdir -p "$RELEASE_DIR"
    
    VERSION="${1:-dev}"
    
    if [[ "$OS" == "linux" ]]; then
        build_agent "linux" "$ARCH" "vstats-agent-linux-$ARCH" "$VERSION"
        build_server "linux" "$ARCH" "vstats-server-linux-$ARCH" "$VERSION"
    elif [[ "$OS" == "darwin" ]]; then
        build_agent "darwin" "$ARCH" "vstats-agent-darwin-$ARCH" "$VERSION"
        build_server "darwin" "$ARCH" "vstats-server-darwin-$ARCH" "$VERSION"
    elif [[ "$OS" == "freebsd" ]]; then
        build_agent "freebsd" "$ARCH" "vstats-agent-freebsd-$ARCH" "$VERSION"
        build_server "freebsd" "$ARCH" "vstats-server-freebsd-$ARCH" "$VERSION"
    else
        error "Unsupported OS: $OS"
    fi
}

# Build for specific target
build_target() {
    local target="$1"
    local version="${2:-dev}"
    
    mkdir -p "$RELEASE_DIR"
    
    case "$target" in
        linux-amd64)
            build_agent "linux" "amd64" "vstats-agent-linux-amd64" "$version"
            build_server "linux" "amd64" "vstats-server-linux-amd64" "$version"
            ;;
        linux-arm64)
            build_agent "linux" "arm64" "vstats-agent-linux-arm64" "$version"
            build_server "linux" "arm64" "vstats-server-linux-arm64" "$version"
            ;;
        darwin-amd64)
            build_agent "darwin" "amd64" "vstats-agent-darwin-amd64" "$version"
            build_server "darwin" "amd64" "vstats-server-darwin-amd64" "$version"
            ;;
        darwin-arm64)
            build_agent "darwin" "arm64" "vstats-agent-darwin-arm64" "$version"
            build_server "darwin" "arm64" "vstats-server-darwin-arm64" "$version"
            ;;
        windows-amd64)
            build_agent "windows" "amd64" "vstats-agent-windows-amd64.exe" "$version"
            build_server "windows" "amd64" "vstats-server-windows-amd64.exe" "$version"
            ;;
        freebsd-amd64)
            build_agent "freebsd" "amd64" "vstats-agent-freebsd-amd64" "$version"
            build_server "freebsd" "amd64" "vstats-server-freebsd-amd64" "$version"
            ;;
        freebsd-arm64)
            build_agent "freebsd" "arm64" "vstats-agent-freebsd-arm64" "$version"
            build_server "freebsd" "arm64" "vstats-server-freebsd-arm64" "$version"
            ;;
        *)
            error "Unknown target: $target"
            ;;
    esac
}

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS] [VERSION]"
    echo ""
    echo "Options:"
    echo "  --native        Build for current platform only (default)"
    echo "  --target TARGET Build for specific target"
    echo "  --clean         Clean build artifacts"
    echo ""
    echo "Targets:"
    echo "  linux-amd64     Linux x86_64"
    echo "  linux-arm64     Linux ARM64"
    echo "  darwin-amd64    macOS Intel"
    echo "  darwin-arm64    macOS Apple Silicon"
    echo "  windows-amd64   Windows x86_64"
    echo "  freebsd-amd64   FreeBSD x86_64"
    echo "  freebsd-arm64   FreeBSD ARM64"
    echo ""
    echo "Version:"
    echo "  Optional version string (default: dev)"
    echo ""
    echo "Output directory: $RELEASE_DIR"
    echo ""
    echo "Note: For CI/CD, use GitHub Actions workflow instead"
}

# Clean build artifacts
clean() {
    info "Cleaning build artifacts..."
    rm -rf "$RELEASE_DIR"
    success "Cleaned"
}

# Main
main() {
    case "$1" in
        --help|-h)
            usage
            exit 0
            ;;
        --clean)
            clean
            exit 0
            ;;
        --target)
            if [ -z "$2" ]; then
                error "Target required. Use --help for usage."
            fi
            build_target "$2" "$3"
            ;;
        --native|"")
            build_native "$1"
            ;;
        *)
            # Try as version
            if [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || [[ "$1" == "dev" ]]; then
                build_native "$1"
            else
                error "Unknown option: $1. Use --help for usage."
            fi
            ;;
    esac
    
    echo ""
    info "Release binaries in: $RELEASE_DIR"
    ls -la "$RELEASE_DIR/" 2>/dev/null || true
}

main "$@"
