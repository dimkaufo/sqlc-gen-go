#!/bin/bash

# Script to capture real sqlc input data for debugging
# Usage: ./capture-real-data.sh [directory]
# Example: ./capture-real-data.sh /path/to/your/sqlc/project

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
ORIGINAL_PWD="$(pwd)"

# Show help if requested
if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
    echo "Usage: $0 [directory]"
    echo ""
    echo "Captures real sqlc input data for debugging the plugin."
    echo ""
    echo "Arguments:"
    echo "  directory    Path to directory containing sqlc.yaml (default: example/sqlcin)"
    echo ""
    echo "Examples:"
    echo "  $0                                                              # Use default"
    echo "  $0 /path/to/your/sqlc/project                                   # Absolute path"
    echo "  $0 ../../hired-rocks/platform-backend/tools/db-generator/platform  # Relative path"
    echo ""
    exit 0
fi

# Parse arguments
SQLC_DIR="${1:-example/sqlcin}"

# Convert to absolute path if it's relative and doesn't start with /
if [[ "$SQLC_DIR" != /* ]] && [[ "$SQLC_DIR" != ~* ]]; then
    # If it's a relative path, make it relative to the original working directory
    SQLC_DIR="$ORIGINAL_PWD/$SQLC_DIR"
fi

# Resolve the absolute path to handle .. and . components
SQLC_DIR="$(cd "$SQLC_DIR" 2>/dev/null && pwd || echo "$SQLC_DIR")"

echo "🔧 Capturing real sqlc input data from: $SQLC_DIR"

# Validate the directory exists and has sqlc.yaml
if [ ! -d "$SQLC_DIR" ]; then
    echo "❌ Directory does not exist: $SQLC_DIR"
    exit 1
fi

if [ ! -f "$SQLC_DIR/sqlc.yaml" ]; then
    echo "❌ No sqlc.yaml found in: $SQLC_DIR"
    echo "   Make sure you're pointing to a directory with a sqlc.yaml file"
    exit 1
fi

# Build the capture tool
echo "1. Building capture tool..."
cd "$ROOT_DIR"
mkdir -p debug/capture/output
go build -o debug/capture/output/capture-input debug/capture/capture-input.go

# Backup original plugins (both native and WASM)
echo "2. Backing up original plugins..."
if [ -f bin/sqlc-gen-go ]; then
    cp bin/sqlc-gen-go bin/sqlc-gen-go-backup
else
    echo "   Building original native plugin first..."
    go build -o bin/sqlc-gen-go ./plugin
    cp bin/sqlc-gen-go bin/sqlc-gen-go-backup
fi

if [ -f bin/sqlc-gen-go.wasm ]; then
    cp bin/sqlc-gen-go.wasm bin/sqlc-gen-go.wasm-backup
else
    echo "   Building original WASM plugin first..."
    GOOS=wasip1 GOARCH=wasm go build -o bin/sqlc-gen-go.wasm ./plugin
    cp bin/sqlc-gen-go.wasm bin/sqlc-gen-go.wasm-backup
fi

# Replace with capture tool (both native and WASM)
echo "3. Installing capture tool..."
cp debug/capture/output/capture-input bin/sqlc-gen-go
GOOS=wasip1 GOARCH=wasm go build -o bin/sqlc-gen-go.wasm debug/capture/capture-input.go

# Run sqlc generate to capture input
echo "4. Running sqlc generate to capture data..."
cd "$SQLC_DIR"
sqlc generate

# Restore original plugins (both native and WASM)
echo "5. Restoring original plugins..."
cd "$ROOT_DIR"
cp bin/sqlc-gen-go-backup bin/sqlc-gen-go
if [ -f bin/sqlc-gen-go.wasm-backup ]; then
    cp bin/sqlc-gen-go.wasm-backup bin/sqlc-gen-go.wasm
fi

# Check if data was captured
if [ -f debug/capture/output/captured_input.json ]; then
    echo "✅ Successfully captured real input data from: $SQLC_DIR"
    echo "📁 Captured data saved to: debug/capture/output/captured_input.json"
    echo ""
    echo "🎯 Now you can debug with real data:"
    echo "   go run debug/main.go"
    echo ""
    echo "Or use the 'Debug Plugin with Real Data' configuration in VS Code"
else
    echo "❌ Failed to capture input data"
    exit 1
fi