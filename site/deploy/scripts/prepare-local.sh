#!/bin/bash
# ===========================================
# Prepare Local Deployment
# Copies frontend dist files to deploy directory
# ===========================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_SOURCE="$DEPLOY_DIR/../dist"
DIST_TARGET="$DEPLOY_DIR/dist"

echo "======================================"
echo "Preparing Local Deployment"
echo "======================================"
echo "Source: $DIST_SOURCE"
echo "Target: $DIST_TARGET"
echo ""

# 检查源目录是否存在
if [ ! -d "$DIST_SOURCE" ]; then
    echo "❌ Error: Source directory not found: $DIST_SOURCE"
    echo "   Please build the frontend first: cd site && npm run build"
    exit 1
fi

# 检查源目录是否为空
if [ -z "$(ls -A "$DIST_SOURCE" 2>/dev/null)" ]; then
    echo "❌ Error: Source directory is empty: $DIST_SOURCE"
    echo "   Please build the frontend first: cd site && npm run build"
    exit 1
fi

# 创建目标目录
mkdir -p "$DIST_TARGET"

# 复制文件
echo ">> Copying frontend files..."
cp -r "$DIST_SOURCE"/* "$DIST_TARGET/"

echo "✅ Frontend files copied successfully"
echo ""
echo "You can now run: docker compose up -d"
