// Package workerd is intl's Intl outside a Cloudflare Worker: workerd, the Workers runtime, serving intl's
// js/intl-worker.mjs with the app's catalogs, set as intl.Runtime. On a Worker, Intl is the Worker's own; anywhere
// else (an ordinary Go server, go test, mise-intl) it is this one, so every place formats with the same engine.
//
// An app on an ordinary Go server starts it before serving, with the catalogs mise-intl gen embedded in its
// messages package:
//
//	stop, err := workerd.Start(locales.Messages)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer stop()
//
// workerd comes from npm, as mise-cloudflare's template gets wrangler: the repo's package.json pins it at Version (the
// template's app's does, mise-intl template), aube installs it (mise run setup), and intl runs the native workerd
// of the node_modules nearest the working directory, or of the one whose .bin is on the PATH (Path). INTL_WORKERD names
// a workerd of your own instead (a server's image built without node_modules). Each Intl call is a request to it on
// 127.0.0.1, about 0.1 ms. It ends with the process that started it, however that process ends (launch).
package workerd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	intljs "github.com/joeblew999/mise-intl/internal/js"
	"github.com/joeblew999/mise-intl/intl"
)

// CompatibilityDate is the Workers compatibility date the Intl Worker runs on.
const CompatibilityDate = "2026-09-11"

// Version is the workerd release intl is tested with: the template's package.json pins it (TestWorkerdPins).
const Version = "1.20261104.1"

// Path is the workerd intl runs: INTL_WORKERD when it names one; else the native workerd of npm's workerd package in
// the node_modules nearest the working directory (the repo's, where aube installs package.json's), or in the
// node_modules whose .bin has the workerd on the PATH (mise's [env] puts it there); else the workerd on the PATH.
func Path() (string, error) {
	if p := os.Getenv("INTL_WORKERD"); p != "" {
		return p, nil
	}
	if wd, err := os.Getwd(); err == nil {
		for dir := wd; ; dir = filepath.Dir(dir) {
			if p, ok := native(filepath.Join(dir, "node_modules")); ok {
				return p, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	p, err := exec.LookPath("workerd")
	if err != nil {
		return "", errors.New(noWorkerd)
	}
	if bin := filepath.Dir(p); filepath.Base(bin) == ".bin" {
		if n, ok := native(filepath.Dir(bin)); ok {
			return n, nil
		}
	}
	return p, nil
}

// native is the workerd binary of npm's workerd package in node_modules: its platform's package beside it
// (@cloudflare/workerd-darwin-arm64, …), where npm, pnpm and aube put it, or the package's own bin/workerd, which its
// install script makes the binary itself outside Windows.
func native(nodeModules string) (string, bool) {
	pkg, err := filepath.EvalSymlinks(filepath.Join(nodeModules, "workerd"))
	if err != nil {
		return "", false
	}
	arch := map[string]string{"amd64": "64", "arm64": "arm64"}[runtime.GOARCH]
	exe := map[bool]string{true: ".exe"}[runtime.GOOS == "windows"]
	platform := filepath.Join("@cloudflare", "workerd-"+runtime.GOOS+"-"+arch, "bin", "workerd"+exe)
	for _, p := range []string{
		filepath.Join(filepath.Dir(pkg), platform),
		filepath.Join(nodeModules, platform),
		filepath.Join(nodeModules, ".aube", "node_modules", platform),
	} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}
	if p := filepath.Join(pkg, "bin", "workerd"); runtime.GOOS != "windows" && isBinary(p) {
		return p, true
	}
	return "", false
}

// isBinary reports whether the file at p is a program of its own, not a script (a #! line).
func isBinary(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	head := make([]byte, 2)
	n, _ := io.ReadFull(f, head)
	return n == 2 && string(head) != "#!"
}

// noWorkerd says how to get workerd.
const noWorkerd = "workerd, the Workers runtime whose Intl intl runs on outside a Worker, isn't installed: the repo's package.json pins it (devDependencies, \"workerd\": \"" + Version + "\", as the template's app has it) and aube installs it (mise run setup, the template's mise.toml has node and aube), or set INTL_WORKERD to a workerd of your own"

var (
	base   string
	client = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 32}, Timeout: 10 * time.Second}
)

// Start starts workerd serving intl's JS with catalogs (an app's intl-messages.mjs: its messages package's
// Messages; "export default {};" for none) and sets intl.Runtime to it. stop ends it.
func Start(catalogs []byte) (stop func(), err error) {
	workerd, err := Path()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "intl-workerd-")
	if err != nil {
		return nil, err
	}
	if err := os.CopyFS(dir, intljs.FS); err != nil {
		return nil, err
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	addr := l.Addr().String()
	l.Close()
	files := map[string]string{
		"intl-messages.mjs": string(catalogs),
		"main.mjs": `import { goIntl } from "./intl-worker.mjs";
import * as intl from "./intl.mjs";
export default {
  async fetch(req) {
    if (req.method !== "POST") return new Response("ok");
    const { method, args, tree } = await req.json();
    if (method === "tree") return new Response(intl.message(args[0], tree, args.slice(1)));
    if (typeof goIntl[method] !== "function") return new Response("no method " + method, { status: 400 });
    return new Response(String(goIntl[method](...args)));
  },
};
`,
		"config.capnp": `using Workerd = import "/workerd/workerd.capnp";
const config :Workerd.Config = (
  services = [(name = "main", worker = (
    modules = [
      (name = "main.mjs", esModule = embed "main.mjs"),
      (name = "intl-worker.mjs", esModule = embed "intl-worker.mjs"),
      (name = "intl.mjs", esModule = embed "intl.mjs"),
      (name = "intl-messages.mjs", esModule = embed "intl-messages.mjs"),
    ],
    compatibilityDate = "` + CompatibilityDate + `"))],
  sockets = [(name = "http", address = "` + addr + `", http = (), service = "main")],
);
`,
	}
	for name, s := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			return nil, err
		}
	}
	var errs bytes.Buffer
	reapOrphans()
	c, err := launch(workerd, dir, io.MultiWriter(os.Stderr, &errs))
	if err != nil {
		return nil, err
	}
	// Who owns this workerd: launch ends it when this process ends, and a leftover (a system with no shell and no
	// parent-death signal) goes at the next Start's reapOrphans.
	os.WriteFile(filepath.Join(dir, "owner"), []byte(fmt.Sprintf("%d %d", os.Getpid(), c.pid())), 0o644)
	exited := c.exited
	stop = func() {
		c.stop()
		os.RemoveAll(dir)
	}
	base = "http://" + addr
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if resp, err := client.Get(base); err == nil {
			resp.Body.Close()
			break
		}
		select {
		case <-exited: // it didn't start: a mise shim with no version for this folder, a bad config
			os.RemoveAll(dir)
			if strings.Contains(errs.String(), "No version is set") {
				return nil, errors.New(noWorkerd)
			}
			return nil, fmt.Errorf("workerd exited: %s", strings.TrimSpace(errs.String()))
		default:
		}
		if time.Now().After(deadline) {
			stop()
			return nil, fmt.Errorf("workerd didn't answer on %s", addr)
		}
	}
	// As on a Worker, an Intl that can't answer gives "" (the value's machine form shows); the reason goes to stderr.
	intl.Runtime = func(method string, args ...string) string {
		s, err := post(map[string]any{"method": method, "args": args})
		if err != nil {
			fmt.Fprintln(os.Stderr, "intl/workerd:", method, err)
			return ""
		}
		return s
	}
	return stop, nil
}

// Tree formats a message tree as mise-intl gen writes it (js/intl.mjs message) in lang with args: for tests of
// the catalogs' compiler.
func Tree(lang string, tree json.RawMessage, args ...string) (string, error) {
	return post(map[string]any{"method": "tree", "args": append([]string{lang}, args...), "tree": tree})
}

func post(body map[string]any) (string, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	resp, err := client.Post(base, "application/json", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("workerd: %d %s", resp.StatusCode, out)
	}
	return string(out), nil
}

// reapOrphans stops the workerds of earlier processes that died without stopping them.
func reapOrphans() {
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), "intl-workerd-*"))
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "owner"))
		if err != nil {
			continue
		}
		var owner, worker int
		if _, err := fmt.Sscan(string(b), &owner, &worker); err != nil || alive(owner) {
			continue
		}
		if alive(worker) {
			killGroup(worker)
		}
		os.RemoveAll(d)
	}
}
