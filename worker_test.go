package e2e

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joeblew999/mise-check/walk"
	"github.com/joeblew999/mise-intl/intl/workerd"
)

// TestWorker is the template's app on a Cloudflare Worker, in local workerd, the runtime Cloudflare runs (mise run
// template:worker): Go built for GOOS=js GOARCH=wasm with workers-go's glue (workers-assets-gen, as charter builds a Go
// Worker), intl's Worker modules (mise-intl assets --worker), and an entry module that gives Go the Worker's own Intl,
// answering / and /de/ with what the Go server answers. No intl daemon, no pitchfork: a Worker has its own Intl.
func TestWorker(t *testing.T) {
	walk.OnlyByName(t)
	build := t.TempDir()
	run := func(dir string, env []string, name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir, cmd.Env = dir, append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
		}
	}
	run(".", nil, "go", "tool", "workers-assets-gen", "-mode=go", "-o", build)
	run(".", []string{"GOOS=js", "GOARCH=wasm"}, "go", "build", "-o", filepath.Join(build, "app.wasm"), "./templates/app")
	run("templates", nil, "go", "run", "../cmd/mise-intl", "assets", "--worker", build)
	for name, s := range map[string]string{
		"entry.mjs": "import { goIntl } from \"./intl-worker.mjs\";\nglobalThis.goIntl = goIntl;\nexport { default } from \"./worker.mjs\";\n",
		"config.capnp": `using Workerd = import "/workerd/workerd.capnp";
const config :Workerd.Config = (
  services = [(name = "main", worker = (
    modules = [
      (name = "entry.mjs", esModule = embed "entry.mjs"),
      (name = "worker.mjs", esModule = embed "worker.mjs"),
      (name = "runtime.mjs", esModule = embed "runtime.mjs"),
      (name = "wasm_exec.js", esModule = embed "wasm_exec.js"),
      (name = "app.wasm", wasm = embed "app.wasm"),
      (name = "intl.mjs", esModule = embed "intl.mjs"),
      (name = "intl-worker.mjs", esModule = embed "intl-worker.mjs"),
      (name = "intl-messages.mjs", esModule = embed "intl-messages.mjs"),
    ],
    compatibilityDate = "2026-09-11"))],
  sockets = [(name = "http", address = "127.0.0.1:8080", http = (), service = "main")],
);
`,
	} {
		if err := os.WriteFile(filepath.Join(build, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Chdir("templates") // its node_modules has the workerd it pins
	bin, err := workerd.Path()
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + l.Addr().String()
	l.Close()
	cmd := exec.Command(bin, "serve", filepath.Join(build, "config.capnp"), "--socket-addr", "http="+strings.TrimPrefix(base, "http://"))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })

	get := func(path string) (string, error) {
		resp, err := http.Get(base + path)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		return string(b), err
	}
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		if _, err := get("/"); err == nil {
			break
		} else if time.Now().After(deadline) {
			t.Fatalf("the Worker didn't answer on %s: %v", base, err)
		}
	}
	for path, want := range map[string][]string{
		"/":    {`lang="en"`, "July 4, 2026", "€1,234.50", "Go, mise, and intl", "Japan", "This page is in 2 languages."},
		"/de/": {`lang="de"`, "4. Juli 2026", "1.234,50\u00a0€", "Go, mise und intl", "vor 3 Tagen", "Diese Seite gibt es in 2 Sprachen."},
	} {
		body, err := get(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range want {
			if !strings.Contains(body, s) {
				t.Fatalf("the Worker at %s has no %q:\n%s", path, s, body)
			}
		}
		t.Logf("PASS the template's app on a Worker, in local workerd, at %s: %s", path, strings.Join(want[1:3], ", "))
	}
}
