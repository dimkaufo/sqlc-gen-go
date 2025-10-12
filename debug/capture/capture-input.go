package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sqlc-dev/plugin-sdk-go/codegen"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"

	golang "github.com/sqlc-dev/sqlc-gen-go/internal"
)

// findProjectRoot walks up the directory tree to find the project root
// by looking for go.mod file
func findProjectRoot(startDir string) string {
	dir := startDir
	for {
		// Check if go.mod exists in current directory
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, fallback to start directory
			return startDir
		}
		dir = parent
	}
}

// This program captures the actual input that sqlc sends to the plugin
// and also runs the real generation so sqlc doesn't fail
func main() {
	// Use the standard codegen.Run approach but intercept the request
	codegen.Run(func(ctx context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
		return captureAndGenerate(ctx, req)
	})
}

func captureAndGenerate(ctx context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	// Get the directory where the debug files should be saved
	var debugDir string

	// Check if we can use os.Executable (not available in WASM)
	if executable, err := os.Executable(); err == nil {
		// Native binary: use executable path to find project root
		// The executable is in bin/ directory, so go up one level to get project root
		execDir := filepath.Dir(executable)
		rootDir := filepath.Dir(execDir)
		debugDir = filepath.Join(rootDir, "debug", "capture", "output")
		fmt.Fprintf(os.Stderr, "🔧 Using native path: %s\n", debugDir)
	} else {
		// WASM or other environment: try to find project root dynamically
		// First check if SQLC_GEN_GO_DEBUG_DIR environment variable is set
		if envDebugDir := os.Getenv("SQLC_GEN_GO_DEBUG_DIR"); envDebugDir != "" {
			debugDir = envDebugDir
			fmt.Fprintf(os.Stderr, "🔧 Using env var SQLC_GEN_GO_DEBUG_DIR: %s\n", debugDir)
		} else {
			// Try to find project root by looking for go.mod file
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to get working directory: %v\n", err)
				debugDir = "./debug/capture/output" // fallback to relative path
			} else {
				// Look for go.mod to identify project root
				projectRoot := findProjectRoot(cwd)
				debugDir = filepath.Join(projectRoot, "debug", "capture", "output")
			}
			fmt.Fprintf(os.Stderr, "🔧 Using dynamic WASM path: %s\n", debugDir)
		}
		fmt.Fprintf(os.Stderr, "🔧 os.Executable error was: %v\n", err)
	}

	if err := os.MkdirAll(debugDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create debug directory %s: %v\n", debugDir, err)
	}

	// Pretty-print the parsed input
	prettyInput, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to marshal pretty input: %v\n", err)
	} else {
		prettyPath := filepath.Join(debugDir, "captured_input.json")
		fmt.Fprintf(os.Stderr, "🔧 Attempting to save to: %s\n", prettyPath)
		if err := os.WriteFile(prettyPath, prettyInput, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to save pretty input to %s: %v\n", prettyPath, err)
		} else {
			fmt.Fprintf(os.Stderr, "✅ Captured input saved to %s\n", prettyPath)
		}
	}

	// Print summary to stderr so it doesn't interfere with sqlc
	fmt.Fprintf(os.Stderr, "📊 Captured request summary:\n")
	if req.Settings != nil {
		fmt.Fprintf(os.Stderr, "- Version: %s\n", req.Settings.Version)
		fmt.Fprintf(os.Stderr, "- Engine: %s\n", req.Settings.Engine)
	}
	fmt.Fprintf(os.Stderr, "- Queries: %d\n", len(req.Queries))
	if req.Catalog != nil {
		totalTables := 0
		for _, schema := range req.Catalog.Schemas {
			totalTables += len(schema.Tables)
		}
		fmt.Fprintf(os.Stderr, "- Schemas: %d (with %d total tables)\n", len(req.Catalog.Schemas), totalTables)
	}

	if len(req.Queries) > 0 {
		fmt.Fprintf(os.Stderr, "- Query names: ")
		for i, q := range req.Queries {
			if i > 0 {
				fmt.Fprintf(os.Stderr, ", ")
			}
			fmt.Fprintf(os.Stderr, "%s", q.Name)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Check for nested configuration
	if len(req.PluginOptions) > 0 {
		var options map[string]interface{}
		if err := json.Unmarshal(req.PluginOptions, &options); err == nil {
			if nested, ok := options["nested"]; ok {
				fmt.Fprintf(os.Stderr, "- Nested config: ✅ Found\n")
				if nestedSlice, ok := nested.([]interface{}); ok {
					fmt.Fprintf(os.Stderr, "- Nested queries: %d\n", len(nestedSlice))
				}
			} else {
				fmt.Fprintf(os.Stderr, "- Nested config: ❌ Not found\n")
			}
		}
	}

	// Now actually run the generation
	resp, err := golang.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %v", err)
	}

	fmt.Fprintf(os.Stderr, "🎯 Generated %d files - ready for debugging!\n", len(resp.Files))

	return resp, nil
}
