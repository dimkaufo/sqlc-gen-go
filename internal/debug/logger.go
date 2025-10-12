package debug

import (
	"fmt"
	"log"
	"os"
)

var (
	debugEnabled = os.Getenv("SQLC_DEBUG") != ""
	debugLogger  *log.Logger
)

func init() {
	if debugEnabled {
		// Create debug log file
		file, err := os.OpenFile("/tmp/sqlc-gen-go-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to stderr if file creation fails
			debugLogger = log.New(os.Stderr, "[SQLC-DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
		} else {
			debugLogger = log.New(file, "[SQLC-DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
		}
	}
}

// Printf writes debug output to log file or stderr when SQLC_DEBUG env var is set
// This is safe to use in protobuf plugins as it never writes to stdout
func Printf(format string, args ...interface{}) {
	if debugEnabled && debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}

// Println writes debug output to log file or stderr when SQLC_DEBUG env var is set
func Println(args ...interface{}) {
	if debugEnabled && debugLogger != nil {
		debugLogger.Println(args...)
	}
}

// Errorf writes to stderr regardless of debug setting (for important warnings/errors)
func Errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[SQLC-ERROR] "+format+"\n", args...)
}

// Warnf writes to stderr when debug is enabled
func Warnf(format string, args ...interface{}) {
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[SQLC-WARN] "+format+"\n", args...)
	}
}
