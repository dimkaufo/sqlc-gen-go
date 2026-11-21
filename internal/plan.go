package golang

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"

	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

// planMarker opens a query comment that picks how Postgres plans the query: "-- plan: custom".
const planMarker = "plan:"

// queryExecMode reads "-- plan: custom" from a query's comments. Such a query runs as pgx's
// unnamed statement (QueryExecModeCacheDescribe), which Postgres plans for each call's
// parameters, instead of a named prepared statement whose cached generic plan ignores them.
func queryExecMode(name, cmd string, comments []string, driver opts.SQLDriver) (string, error) {
	for _, comment := range comments {
		value, ok := strings.CutPrefix(strings.TrimSpace(comment), planMarker)
		if !ok {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 || strings.TrimRight(fields[0], ":,;.") != "custom" {
			return "", fmt.Errorf("query %s: unknown %q, want \"-- plan: custom\"", name, strings.TrimSpace(comment))
		}
		if driver != opts.SQLDriverPGXV5 {
			return "", fmt.Errorf("query %s: \"-- plan: custom\" needs sql_package pgx/v5", name)
		}
		if cmd == metadata.CmdCopyFrom || strings.HasPrefix(cmd, ":batch") {
			return "", fmt.Errorf("query %s: \"-- plan: custom\" does not apply to %s", name, cmd)
		}
		return "pgx.QueryExecModeCacheDescribe", nil
	}
	return "", nil
}

// CallParams is the argument list after the SQL, led by the exec mode when the query pins one.
func (q Query) CallParams() string {
	params := q.Arg.Params()
	switch {
	case q.ExecMode == "":
		return params
	case params == "":
		return q.ExecMode
	case strings.HasPrefix(params, "\n"):
		return "\n" + q.ExecMode + "," + params
	default:
		return q.ExecMode + ", " + params
	}
}
