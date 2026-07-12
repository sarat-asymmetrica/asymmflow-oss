#!/bin/bash
set -e

echo "============================================"
echo "  AsymmFlow - Build Release"
echo "============================================"
echo ""

# Detect architecture
ARCH=$(uname -m)
OS=$(uname -s)

echo "Platform: $OS / $ARCH"
echo ""

# Determine Wails platform string
case "$OS" in
    Darwin)
        if [ "$ARCH" = "arm64" ]; then
            PLATFORM="darwin/arm64"
        else
            PLATFORM="darwin/amd64"
        fi
        ;;
    Linux)
        PLATFORM="linux/amd64"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        PLATFORM="windows/amd64"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Target platform: $PLATFORM"
echo ""

# Check prerequisites
echo "Checking prerequisites..."
command -v go >/dev/null 2>&1 || { echo "ERROR: Go not found. Install with: brew install go"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "ERROR: Node.js not found. Install with: brew install node"; exit 1; }
command -v wails >/dev/null 2>&1 || { echo "ERROR: Wails not found. Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"; exit 1; }
echo "All prerequisites found."
echo ""

# Frontend build
echo "Building frontend..."
cd frontend
if command -v pnpm >/dev/null 2>&1; then
    pnpm install
elif command -v npm >/dev/null 2>&1; then
    npm install
fi
cd ..

# Native build
echo ""
echo "Building for $PLATFORM..."
CGO_ENABLED=1 wails build -clean -platform "$PLATFORM"
echo "Native build complete!"

# Windows cross-compile (optional)
if [ "$1" = "--cross-windows" ]; then
    if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
        echo ""
        echo "Cross-compiling for Windows (amd64)..."
        CGO_ENABLED=1 \
            CC=x86_64-w64-mingw32-gcc \
            CXX=x86_64-w64-mingw32-g++ \
            GOOS=windows GOARCH=amd64 \
            wails build -clean -platform windows/amd64
        echo "Windows cross-compile complete!"
    else
        echo ""
        echo "WARNING: mingw-w64 not found. Install with: brew install mingw-w64"
        echo "Skipping Windows cross-compilation."
    fi
fi

echo ""
echo "============================================"
echo "  Build complete! Output: build/bin/"
echo "============================================"
ls -la build/bin/ 2>/dev/null || echo "(build/bin/ directory not found - check for errors above)"
