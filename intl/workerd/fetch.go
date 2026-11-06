package workerd

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// releases are the files of the workerd release Version names, as GitHub lists them (gh api
// repos/cloudflare/workerd/releases/tags/v<Version>, each asset's digest): each OS and architecture's gzipped workerd,
// and the SHA-256 a download must match. Moving Version means moving these with it.
var releases = map[string]struct{ file, sha256 string }{
	"darwin/amd64":  {"workerd-darwin-64.gz", "26d9e557e2e9977a2fcea6e23b32adae73ca9eb16498108206b81d0ef0a3477f"},
	"darwin/arm64":  {"workerd-darwin-arm64.gz", "2811304d491bdff14006a13d326fa9eeb37ba9605cfc5aa8aebea15702595e4b"},
	"linux/amd64":   {"workerd-linux-64.gz", "4e7e6600b23b9e8ea464b57af69f930f6e1901316404b6e35aa254e666304cbb"},
	"linux/arm64":   {"workerd-linux-arm64.gz", "9def2b6c2ebc3a35b3598fe8a3ce426393b624899afa9e620c22650b6e597f0e"},
	"windows/amd64": {"workerd-windows-64.gz", "5be8eb1608df2169638d38337912063b0802583dfb869ebb9071bdf5b787f4f6"},
}

// Path is the workerd intl runs: INTL_WORKERD, when it names one, or else the release Version pins, fetched once into
// this machine's cache (os.UserCacheDir()/mise-intl/workerd/<Version>/) and checked against its pinned SHA-256. So an
// app needs nothing but intl's module: no workerd on the PATH, and no mise.
func Path() (string, error) {
	if p := os.Getenv("INTL_WORKERD"); p != "" {
		return p, nil
	}
	r, ok := releases[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return "", fmt.Errorf("workerd %s has no release for %s/%s: set INTL_WORKERD to a workerd of your own", Version, runtime.GOOS, runtime.GOARCH)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("no cache folder for workerd (%v): set INTL_WORKERD to a workerd of your own", err)
	}
	dir := filepath.Join(cache, "mise-intl", "workerd", Version)
	path := filepath.Join(dir, "workerd")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := fetch(r.file, r.sha256, dir, path); err != nil {
		return "", fmt.Errorf("workerd %s: %w (or set INTL_WORKERD to a workerd of your own)", Version, err)
	}
	return path, nil
}

// fetch downloads the release's file into path, unpacked, once its SHA-256 matches. It writes beside path first and
// moves it there whole, so several processes fetching at once (go test's packages) each leave a complete workerd.
func fetch(file, want, dir, path string) error {
	url := "https://github.com/cloudflare/workerd/releases/download/v" + Version + "/" + file
	fmt.Fprintf(os.Stderr, "intl: fetching workerd %s, once, into %s\n", Version, dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 10 * time.Minute}).Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	sum := sha256.New()
	gz, err := gzip.NewReader(io.TeeReader(resp.Body, sum))
	if err != nil {
		return fmt.Errorf("%s: %w", url, err)
	}
	tmp, err := os.CreateTemp(dir, "workerd-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, gz); err != nil {
		tmp.Close()
		return fmt.Errorf("%s: %w", url, err)
	}
	io.Copy(io.Discard, resp.Body) // the rest of the gzip stream, so the sum is the whole file's
	if err := tmp.Close(); err != nil {
		return err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); got != want {
		return errors.New(file + "'s SHA-256 is " + got + ", not the " + want + " intl pins: not used")
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		// Another process fetched it at the same time (go test's packages): on Windows a rename can't replace a file
		// another process has open, where macOS and Linux replace it. Theirs is the same release, checked the same way.
		if _, statErr := os.Stat(path); statErr == nil {
			return nil
		}
		return err
	}
	return nil
}
