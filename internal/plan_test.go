package golang

import (
	"strings"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"

	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

func TestQueryExecMode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cmd      string
		comments []string
		driver   opts.SQLDriver
		want     string
		wantErr  string
	}{
		{name: "no marker", cmd: ":many", comments: []string{" a note"}, driver: opts.SQLDriverPGXV5},
		{name: "custom", cmd: ":many", comments: []string{" why", " plan: custom: optional filters"}, driver: opts.SQLDriverPGXV5, want: "pgx.QueryExecModeCacheDescribe"},
		{name: "unknown plan", cmd: ":one", comments: []string{" plan: generic"}, driver: opts.SQLDriverPGXV5, wantErr: "unknown"},
		{name: "not pgx v5", cmd: ":many", comments: []string{" plan: custom"}, driver: opts.SQLDriverLibPQ, wantErr: "pgx/v5"},
		{name: "batch", cmd: ":batchmany", comments: []string{" plan: custom"}, driver: opts.SQLDriverPGXV5, wantErr: "does not apply"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := queryExecMode("Q", tc.cmd, tc.comments, tc.driver)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestQueryCallParams(t *testing.T) {
	one := QueryValue{Name: "id", Typ: "int64", SQLDriver: opts.SQLDriverPGXV5, Column: &plugin.Column{}}
	for _, tc := range []struct {
		name string
		q    Query
		want string
	}{
		{name: "default mode", q: Query{}, want: ""},
		{name: "mode alone", q: Query{ExecMode: "pgx.QueryExecModeCacheDescribe"}, want: "pgx.QueryExecModeCacheDescribe"},
		{name: "mode then params", q: Query{ExecMode: "M", Arg: one}, want: "M, id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.q.CallParams(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
