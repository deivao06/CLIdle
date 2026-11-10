// Package update finds newer releases on GitHub and replaces the running binary.
package update

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Version is set at build time with -ldflags "-X clidle/internal/update.Version=vX.Y.Z".
var Version = "dev"

var API = "https://api.github.com/repos/deivao06/CLIdle/releases/latest"

var NotesAPI = "https://api.github.com/repos/deivao06/CLIdle/releases?per_page=10"

// Site is the page URL in the browser build, which reads the version from the site instead of GitHub.
var Site string

const (
	Timeout   = 10 * time.Second
	sumsName  = "SHA256SUMS"
	userAgent = "clidle-updater"
)

type Release struct {
	Version string
	Asset   string
	Sums    string
}

// Enabled is false for local builds, which never look for updates.
func Enabled() bool { return Version != "dev" && (runtime.GOOS != "js" || Site != "") }

func AssetName(goos, goarch string) string {
	name := "clidle-" + goos + "-" + goarch
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Newer reports whether version a is a later semver than b.
func Newer(a, b string) bool {
	x, ok1 := parse(a)
	y, ok2 := parse(b)
	if !ok1 || !ok2 {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

func get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if Site == "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return resp, nil
}

// Latest fetches the latest release; it has no asset when there is no build for this system.
func Latest(ctx context.Context) (*Release, error) {
	resp, err := get(ctx, API)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	r := &Release{Version: data.Tag}
	want := AssetName(runtime.GOOS, runtime.GOARCH)
	for _, a := range data.Assets {
		switch a.Name {
		case want:
			r.Asset = a.URL
		case sumsName:
			r.Sums = a.URL
		}
	}
	return r, nil
}

type Note struct {
	Version, Date, Body string
}

// Notes fetches the notes of the latest published releases, newest first.
func Notes(ctx context.Context) ([]Note, error) {
	resp, err := get(ctx, NotesAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data []struct {
		Tag       string `json:"tag_name"`
		Published string `json:"published_at"`
		Body      string `json:"body"`
		Draft     bool   `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	var notes []Note
	for _, d := range data {
		if !d.Draft {
			notes = append(notes, Note{d.Tag, d.Published[:min(10, len(d.Published))], d.Body})
		}
	}
	return notes, nil
}

// Check returns the latest release when it is newer than this build and has a binary for it.
func Check(ctx context.Context) (*Release, error) {
	if Site != "" {
		return checkSite(ctx)
	}
	r, err := Latest(ctx)
	if err != nil || !Newer(r.Version, Version) || r.Asset == "" || r.Sums == "" {
		return nil, err
	}
	return r, nil
}

// checkSite reads version.txt next to the page; the query skips the browser cache.
func checkSite(ctx context.Context) (*Release, error) {
	base, err := url.Parse(Site)
	if err != nil {
		return nil, err
	}
	ref := &url.URL{Path: "version.txt", RawQuery: "t=" + strconv.FormatInt(time.Now().UnixNano(), 10)}
	resp, err := get(ctx, base.ResolveReference(ref).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if v := strings.TrimSpace(string(b)); err == nil && Newer(v, Version) {
		return &Release{Version: v}, nil
	}
	return nil, err
}

func checksum(ctx context.Context, url, name string) (string, error) {
	resp, err := get(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if f := strings.Fields(sc.Text()); len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return f[0], nil
		}
	}
	return "", fmt.Errorf("%s not in %s", name, sumsName)
}

// Install downloads the release binary, checks it and puts it in place of exe.
func Install(ctx context.Context, r *Release, exe string) error {
	want, err := checksum(ctx, r.Sums, AssetName(runtime.GOOS, runtime.GOARCH))
	if err != nil {
		return err
	}
	resp, err := get(ctx, r.Asset)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".clidle-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(tmp, h), resp.Body)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return errors.New("checksum mismatch")
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	return replace(tmp.Name(), exe)
}

// replace swaps exe for the new file; Windows cannot overwrite a running exe, so it moves it aside first.
func replace(newFile, exe string) error {
	if runtime.GOOS != "windows" {
		return os.Rename(newFile, exe)
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(newFile, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	return nil
}

// Cleanup removes the binary left behind by an update on Windows; it may still be locked, so errors are ignored.
func Cleanup(exe string) { os.Remove(exe + ".old") }

// Executable is the path of the running binary with symlinks resolved, so the real file is replaced.
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}
