// Package workerd is intl's Intl outside a Cloudflare Worker: workerd, the Workers runtime, serving intl's
// js/intl-worker.mjs with the app's catalogs, set as intl.Runtime. On a Worker, Intl is the Worker's own; anywhere
// else (an ordinary Go server, go test, mise-intl) it is this one, so every place formats with the same engine.
//
// An app on an ordinary Go server starts it before serving:
//
//	if err := workerd.Start(); err != nil {
//		log.Fatal(err)
//	}
//
// workerd runs as the repo's intl daemon, which pitchfork runs (pitchfork.toml, the template's app has it): workerd
// serving the files mise-intl gen writes beside the catalogs (intl-workerd/), started by the first Start when it isn't
// running (pitchfork start intl), restarted when gen rewrites them, stopped with pitchfork stop intl. INTL_WORKERD_URL
// names a workerd of your own instead (a server's, deployed with it). workerd comes from npm, as mise-cloudflare's
// template gets wrangler: the repo's package.json pins it at Version, aube installs it (mise run setup), and the daemon
// runs its native binary (Path). Each Intl call is a request to it on 127.0.0.1, about 0.1 ms.
package workerd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joeblew999/mise-intl/intl"
)

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

// noDaemon says how to get the intl daemon.
const noDaemon = "intl's Intl outside a Worker is the repo's intl daemon: pitchfork runs it (the template's mise.toml pins pitchfork, and its pitchfork.toml has the daemon, [daemons.intl]; run through mise: mise run serve), or set INTL_WORKERD_URL to a workerd of your own"

var (
	base   string
	client = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 32}, Timeout: 10 * time.Second}
)

// Start sets intl.Runtime to workerd's Intl: INTL_WORKERD_URL's, or else the repo's intl daemon, started first when it
// isn't running (pitchfork start intl), in the folder INTL_PITCHFORK_DIR names (this repo's own tests: the template's)
// or else the working directory's pitchfork.toml.
func Start() error {
	if base = strings.TrimSuffix(os.Getenv("INTL_WORKERD_URL"), "/"); base == "" {
		var err error
		if base, err = daemon(); err != nil {
			return err
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
	return nil
}

// daemon starts the repo's intl daemon when it isn't running, and returns its URL, on the port pitchfork gave it.
func daemon() (string, error) {
	pitchfork := func(args ...string) ([]byte, error) {
		cmd := exec.Command("pitchfork", args...)
		var errs bytes.Buffer
		cmd.Dir, cmd.Stderr = os.Getenv("INTL_PITCHFORK_DIR"), &errs
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("pitchfork %s: %v: %s\n%s", strings.Join(args, " "), err, bytes.TrimSpace(append(out, errs.Bytes()...)), noDaemon)
		}
		return out, nil
	}
	// The supervisor, when it isn't running, from a folder that is no repo's: it keeps the folder it starts in as its
	// own, and Windows can't remove a folder a process is in (a repo's, say, when its test ends).
	sup := exec.Command("pitchfork", "supervisor", "start")
	sup.Dir = os.TempDir()
	sup.Run()
	if _, err := pitchfork("start", "-q", "intl"); err != nil {
		return "", err
	}
	out, err := pitchfork("status", "--json", "intl")
	if err != nil {
		return "", err
	}
	var s struct{ Port []int }
	if i := bytes.IndexByte(out, '{'); i < 0 || json.Unmarshal(out[i:], &s) != nil || len(s.Port) == 0 {
		return "", fmt.Errorf("pitchfork status intl gives no port: %s", out)
	}
	return fmt.Sprintf("http://127.0.0.1:%d", s.Port[0]), nil
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
	// The daemon restarting, after gen rewrote its files: it is back in a moment.
	for until := time.Now().Add(20 * time.Second); err != nil && time.Now().Before(until); {
		time.Sleep(100 * time.Millisecond)
		resp, err = client.Post(base, "application/json", bytes.NewReader(b))
	}
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
