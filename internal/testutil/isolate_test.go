package testutil

import (
	"flag"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/config"
)

const isolationProbeArg = "isolation-probe"

func TestIsolatedMain_HidesCallerConfigAndEnv(t *testing.T) {
	if flag.Arg(0) == isolationProbeArg {
		f, err := config.Read()
		if err != nil {
			t.Fatal(err)
		}
		if f.DefaultAccount != "" {
			t.Errorf("config.Read() saw the caller's config: default_account = %q", f.DefaultAccount)
		}
		for _, kv := range os.Environ() {
			if strings.HasPrefix(kv, "GCCLI_") {
				t.Errorf("the caller's %s reached the test", kv)
			}
		}
		return
	}

	caller := t.TempDir()
	cfg := filepath.Join(caller, "gccli", "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"default_account": "caller@example.com"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(caller, "Library", "Application Support"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(cfg), filepath.Join(caller, "Library", "Application Support", "gccli")); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestIsolatedMain_HidesCallerConfigAndEnv$", isolationProbeArg)
	cmd.Env = append(os.Environ(),
		"HOME="+caller,
		"XDG_CONFIG_HOME="+caller,
		"GCCLI_ACCOUNT=caller@example.com",
		"GCCLI_KEYRING_BACKEND=secret-service",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("the test process saw the caller's config or GCCLI_* variables: %v\n%s", err, out)
	}
}

func TestIsolatedMain_RunsInEveryPackageThatLinksConfig(t *testing.T) {
	const module = "github.com/bpauli/gccli/"
	const config = "internal/config"
	root := filepath.Join("..", "..")
	prodImports := map[string][]string{}
	testImports := map[string][]string{}
	isolated := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, src, parser.ImportsOnly)
		if err != nil {
			return err
		}
		dir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		dir = filepath.ToSlash(dir)
		imports := prodImports
		if strings.HasSuffix(path, "_test.go") {
			imports = testImports
			if _, ok := testImports[dir]; !ok {
				testImports[dir] = nil
			}
			if strings.Contains(string(src), "IsolatedMain(m)") {
				isolated[dir] = true
			}
		}
		for _, imp := range f.Imports {
			if p := strings.Trim(imp.Path.Value, `"`); strings.HasPrefix(p, module) {
				imports[dir] = append(imports[dir], strings.TrimPrefix(p, module))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for dir, direct := range testImports {
		seen := map[string]bool{dir: true}
		queue := append(append([]string{}, prodImports[dir]...), direct...)
		for len(queue) > 0 {
			next := queue[0]
			queue = queue[1:]
			if !seen[next] {
				seen[next] = true
				queue = append(queue, prodImports[next]...)
			}
		}
		if seen[config] && !isolated[dir] {
			t.Errorf("the tests of %s link %s but no TestMain calls testutil.IsolatedMain(m)", dir, config)
		}
	}
	for _, dir := range []string{"internal/cmd", config, "internal/secrets", "internal/testutil"} {
		if _, ok := testImports[dir]; !ok {
			t.Errorf("the walk found no tests in %s", dir)
		}
	}
}
