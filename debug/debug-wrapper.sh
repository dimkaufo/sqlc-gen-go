#!/bin/bash

# Debug wrapper script for sqlc-gen-go plugin
# This script builds the plugin with debug symbols and starts a Delve server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Building debug-enabled sqlc-gen-go plugin..."

# Build with debug symbols
cd "$ROOT_DIR"
go build -gcflags="all=-N -l" -o bin/sqlc-gen-go-debug ./plugin

echo "Starting Delve server on port 2345..."
echo "The plugin will wait for debugger attachment before proceeding."
echo ""
echo "To debug:"
echo "1. Set breakpoints in VS Code/Cursor"
echo "2. Start the 'Debug with Delve Server' configuration"
echo "3. Run 'sqlc generate' in another terminal"
echo ""

# Replace the original binary with debug wrapper
mv bin/sqlc-gen-go bin/sqlc-gen-go-original 2>/dev/null || true

# Create a wrapper script that starts delve
cat > bin/sqlc-gen-go << 'EOF'
#!/bin/bash
exec dlv exec --headless --listen=:2345 --api-version=2 --accept-multiclient "$(dirname "$0")/sqlc-gen-go-debug" "$@"
EOF

chmod +x bin/sqlc-gen-go

echo "Debug wrapper installed. Run 'sqlc generate' to start debugging session."
echo "To restore original binary, run: mv bin/sqlc-gen-go-original bin/sqlc-gen-go"



