#!/bin/bash

# vStats Debug Data Generator
# Usage:
#   ./run.sh                    # Run both history and realtime mode (prompts for token)
#   ./run.sh history            # Generate 2h of history data only
#   ./run.sh realtime           # Run realtime agents only
#   ./run.sh cleanup            # Remove all debug servers
#   ./run.sh build              # Build Linux amd64 executable
#   ./run.sh --help             # Show all options

set -e

cd "$(dirname "$0")"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

# Download dependencies
echo "📦 Checking dependencies..."
go mod tidy 2>/dev/null || go mod download

# Default values
SERVER_URL="${VSTATS_SERVER_URL:-http://localhost:3001}"
SERVER_COUNT="${VSTATS_SERVER_COUNT:-100}"
HISTORY_HOURS="${VSTATS_HISTORY_HOURS:-2}"
INTERVAL="${VSTATS_INTERVAL:-3}"
AGG_INTERVAL="${VSTATS_AGG_INTERVAL:-60}"

# Parse mode from first argument
MODE="both"
CLEANUP=""
if [ "$1" == "build" ]; then
    echo "🔨 Building Linux executable..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o debug-data-linux-amd64 main.go
    echo "✅ Built: debug-data-linux-amd64"
    exit 0
elif [ "$1" == "history" ]; then
    MODE="history"
    shift
elif [ "$1" == "realtime" ]; then
    MODE="realtime"
    shift
elif [ "$1" == "cleanup" ]; then
    CLEANUP="--cleanup"
    shift
fi

echo "🚀 Starting vStats Debug Data Generator (New Protocol)"
echo "   Server URL: $SERVER_URL"
echo "   Server Count: $SERVER_COUNT"
echo "   Mode: $MODE"
echo "   Metrics Interval: ${INTERVAL}s"
echo "   Aggregation Sync: ${AGG_INTERVAL}s"
echo ""

# Run the generator (will prompt for token)
go run main.go \
    --server "$SERVER_URL" \
    --count "$SERVER_COUNT" \
    --hours "$HISTORY_HOURS" \
    --interval "$INTERVAL" \
    --agg-interval "$AGG_INTERVAL" \
    --mode "$MODE" \
    $CLEANUP \
    "$@"
