#!/usr/bin/env sh
set -eu

mkdir -p dist

go test .
CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -trimpath -ldflags="-H=windowsgui" -o dist/HighwayPursuit-Ultrawide-Patcher.exe .

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum dist/HighwayPursuit-Ultrawide-Patcher.exe
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 dist/HighwayPursuit-Ultrawide-Patcher.exe
fi
