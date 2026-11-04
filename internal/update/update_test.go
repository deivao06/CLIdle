package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v0.2.0", "v0.1.0", true},
		{"v0.1.10", "v0.1.9", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.1.0", "v0.2.0", false},
		{"v0.2.0", "dev", false},
		{"garbage", "v0.1.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

func TestAssetName(t *testing.T) {
	if AssetName("linux", "amd64") != "clidle-linux-amd64" || AssetName("windows", "amd64") != "clidle-windows-amd64.exe" {
		t.Fatal("wrong asset names")
	}
}

// fakeGitHub serves a release with a binary for this system and its checksum.
func fakeGitHub(t *testing.T, tag string, bin []byte, sum string) {
	name := AssetName(runtime.GOOS, runtime.GOARCH)
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			json.NewEncoder(w).Encode(map[string]any{
				"tag_name": tag,
				"body":     "## Novidades\n\n- new thing\n",
				"assets": []map[string]string{
					{"name": name, "browser_download_url": srv.URL + "/bin"},
					{"name": "SHA256SUMS", "browser_download_url": srv.URL + "/sums"},
				},
			})
		case "/bin":
			w.Write(bin)
		case "/sums":
			w.Write([]byte(sum + "  " + name + "\n"))
		}
	}))
	t.Cleanup(srv.Close)
	old, oldV := API, Version
	API, Version = srv.URL+"/latest", "v0.1.0"
	t.Cleanup(func() { API, Version = old, oldV })
}

func TestCheckAndInstall(t *testing.T) {
	bin := []byte("new binary")
	h := sha256.Sum256(bin)
	fakeGitHub(t, "v0.2.0", bin, hex.EncodeToString(h[:]))
	r, err := Check(context.Background())
	if err != nil || r == nil || r.Version != "v0.2.0" {
		t.Fatalf("check %+v %v", r, err)
	}
	exe := filepath.Join(t.TempDir(), "clidle")
	os.WriteFile(exe, []byte("old binary"), 0o755)
	if err := Install(context.Background(), r, exe); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "new binary" {
		t.Fatalf("exe has %q", got)
	}
}

func TestInstallRejectsBadChecksum(t *testing.T) {
	fakeGitHub(t, "v0.2.0", []byte("tampered"), "0000")
	r, _ := Check(context.Background())
	dir := t.TempDir()
	exe := filepath.Join(dir, "clidle")
	os.WriteFile(exe, []byte("old binary"), 0o755)
	if err := Install(context.Background(), r, exe); err == nil {
		t.Fatal("installed a binary with a bad checksum")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatal("old binary was touched")
	}
	if files, _ := os.ReadDir(dir); len(files) != 1 {
		t.Fatalf("temp file left behind: %d files", len(files))
	}
}

func TestCheckIgnoresSameVersion(t *testing.T) {
	fakeGitHub(t, "v0.1.0", nil, "")
	if r, err := Check(context.Background()); r != nil || err != nil {
		t.Fatalf("offered %+v %v", r, err)
	}
}
