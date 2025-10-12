# Debugging sqlc-gen-go Plugin

This directory contains utilities for debugging the sqlc-gen-go plugin. Since sqlc spawns plugins as separate processes, normal debugging approaches don't work directly.

## Available Debugging Methods

### Method 1: Debug with Real Data (Recommended)

To debug with the actual data that sqlc sends (this will hit all your nested processing breakpoints):

1. **Capture real input** (one command):

   ```bash
   # Use default example directory
   ./debug/capture/capture-real-data.sh
   
   # Or specify your own sqlc project directory
   ./debug/capture/capture-real-data.sh /path/to/your/sqlc/project
   ```

2. **Debug with captured input**:

   ```bash
   # Run the debug harness with real data
   go run debug/main.go
   ```

   Or use the **"Debug Plugin with Real Data"** configuration in VS Code (recommended)

### Method 2: Remote Debugging with Delve

For debugging the actual plugin process spawned by sqlc:

1. **Setup debug environment**:

   ```bash
   ./debug/debug-wrapper.sh
   ```

2. **Start the debugger**: Select "Debug with Delve Server" in VS Code/Cursor
3. **Run sqlc generate** in another terminal:

   ```bash
   cd example/sqlcin && sqlc generate
   ```

The plugin will wait for debugger attachment before proceeding.

## VS Code/Cursor Tasks

Use these tasks from the Command Palette (`Ctrl/Cmd+Shift+P` → "Tasks: Run Task"):

- **Build Debug Binary**: Builds plugin with debug symbols
- **Setup Debug Environment**: Installs debug wrapper for remote debugging
- **Capture Plugin Input**: Captures actual input from sqlc for analysis

## Debug Configurations

Available in the Run and Debug panel:

- **Debug Plugin with Real Data**: Debug with captured real data (recommended)
- **Debug with Delve Server**: Attach to remote debugging session
- **Attach to Running Plugin**: Attach to any running plugin process

## Files

- `main.go`: Debug harness for real captured data
- `debug-wrapper.sh`: Script to setup remote debugging
- `capture/`: Capture tools and data
  - `capture-input.go`: Tool to capture real plugin input
  - `capture-real-data.sh`: Script to capture real data (one command)
  - `output/`: Capture output directory
    - `capture-input`: Compiled capture tool binary
    - `captured_input.json`: Captured plugin input (created when using Method 2)
- `output/`: Directory containing all generated files

## Tips

1. **Start with Method 1** using real captured data for most debugging
2. **Use Method 2** only when you need to debug the actual plugin lifecycle
4. **Set breakpoints** in `internal/nested.go` around line 260+ for nested query logic
5. **Inspect variables** like `groupConfig`, `nestedStructs`, and generated field mappings
