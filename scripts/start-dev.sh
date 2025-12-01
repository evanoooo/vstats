#!/bin/bash
#
# Development server start script
# Rebuilds frontend and backend, then starts the server
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SERVER_DIR="$PROJECT_ROOT/server"
WEB_DIR="$PROJECT_ROOT/web"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${BLUE}       vStats Development Server        ${NC}"
echo -e "${BLUE}════════════════════════════════════════${NC}"
echo ""

# Build frontend
echo -e "${YELLOW}[1/2]${NC} Building frontend..."
cd "$WEB_DIR"
if [ ! -d "node_modules" ]; then
    npm install
fi
npm run build
echo -e "${GREEN}✓ Frontend built${NC}"
echo ""

# Build backend
echo -e "${YELLOW}[2/2]${NC} Building backend..."
cd "$SERVER_DIR"
cargo build --release
echo -e "${GREEN}✓ Backend built${NC}"
echo ""

# Set environment variables
export RUST_LOG=info
export VSTATS_PORT=3001
export VSTATS_WEB_DIR="$WEB_DIR/dist"

SERVER_BINARY="$SERVER_DIR/target/release/xprob-server"

echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${GREEN}Server: http://localhost:3001${NC}"
echo -e "${GREEN}Web:    $VSTATS_WEB_DIR${NC}"
echo -e "${GREEN}Pass:   admin${NC}"
echo -e "${BLUE}════════════════════════════════════════${NC}"
echo ""

# Change to server directory and run
cd "$SERVER_DIR"
"$SERVER_BINARY"
