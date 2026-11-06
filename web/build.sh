#!/bin/sh
# Builds the browser version into web/dist. Bubble Tea v1 has no js/wasm support,
# so a temporary copy of it gets web/_bubbletea/tty_js.go through a throwaway go.mod.
set -e
cd "$(dirname "$0")/.."
tmp=$(mktemp -d)
trap 'chmod -R u+w "$tmp"; rm -rf "$tmp"' EXIT
mod=$(go list -m -f '{{.Dir}}' github.com/charmbracelet/bubbletea)
cp -r "$mod" "$tmp/bubbletea"
chmod -R u+w "$tmp/bubbletea"
cp web/_bubbletea/tty_js.go "$tmp/bubbletea/"
cp go.mod "$tmp/go.mod"
cp go.sum "$tmp/go.sum"
go mod edit -modfile="$tmp/go.mod" -replace "github.com/charmbracelet/bubbletea=$tmp/bubbletea"
mkdir -p web/dist
GOOS=js GOARCH=wasm go build -modfile="$tmp/go.mod" -trimpath -ldflags "-s -w ${LDFLAGS:-}" -o web/dist/clidle.wasm ./web
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/dist/
cp web/index.html web/dist/
ls -l web/dist
