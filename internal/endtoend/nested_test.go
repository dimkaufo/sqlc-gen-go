package endtoend

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// -update rewrites the checked-in generated code instead of diffing against it.
var update = flag.Bool("update", false, "rewrite generated code under nested/gen")

// TestNestedGenerate regenerates nested/ with the plugin built from this checkout and
// fails on any drift from nested/gen. Hand-written *_test.go files there are left alone.
func TestNestedGenerate(t *testing.T) {
	tmp := t.TempDir()

	binDir := filepath.Join(tmp, "bin")
	run(t, "../..", nil, "go", "build", "-o", filepath.Join(binDir, "sqlc-gen-go"), "./plugin")

	work := filepath.Join(tmp, "nested")
	for _, p := range []string{"sqlc.yaml", "schema.sql", "query"} {
		copyTree(t, filepath.Join("nested", p), filepath.Join(work, p))
	}

	sqlc := []string{"tool", "sqlc", "generate", "-f", filepath.Join(work, "sqlc.yaml")}
	run(t, ".", []string{"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH")}, "go", sqlc...)

	got := readGenerated(t, filepath.Join(work, "gen"))
	if *update {
		writeGenerated(t, "nested/gen", got)
		return
	}
	want := readGenerated(t, "nested/gen")

	for name, g := range got {
		w, ok := want[name]
		switch {
		case !ok:
			t.Errorf("%s: generated but not checked in (run go test -run TestNestedGenerate -update)", name)
		case !bytes.Equal(g, w):
			t.Errorf("%s: drifted from checked-in output (run go test -run TestNestedGenerate -update)", name)
		}
	}
	for name := range want {
		if _, ok := got[name]; !ok {
			t.Errorf("%s: checked in but no longer generated", name)
		}
	}
}

func run(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(), "GOTOOLCHAIN=auto"), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

// readGenerated maps relative path to content for every generated (non-test) file under root.
func readGenerated(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[rel], err = os.ReadFile(path)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// writeGenerated replaces every generated file under root, keeping hand-written tests.
func writeGenerated(t *testing.T, root string, files map[string][]byte) {
	t.Helper()
	for name := range readGenerated(t, root) {
		if err := os.Remove(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
