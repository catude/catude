#!/bin/sh
# catude installer for Linux and macOS. Downloads the latest release, verifies its checksum and runs
# `catude install`, which puts catude and ct in ~/.local/bin and on your PATH (pre-selected).
#   Review it first, then:  sh install.sh [--yes]
set -eu
REPO="${CATUDE_REPO:-catude/catude}"
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) echo "unsupported CPU $(uname -m)"; exit 1 ;; esac
api="https://api.github.com/repos/$REPO/releases/latest"
url=$(curl -fsSL "$api" | grep -o "https://[^\"]*_${os}_${arch}\.tar\.gz" | head -1)
sums=$(curl -fsSL "$api" | grep -o 'https://[^"]*checksums\.txt' | head -1)
[ -n "$url" ] && [ -n "$sums" ] || { echo "no $os/$arch build found"; exit 1; }
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
curl -fsSL "$url" -o "$tmp/catude.tar.gz"
want=$(curl -fsSL "$sums" | grep "$(basename "$url")" | cut -d' ' -f1)
have=$( (sha256sum "$tmp/catude.tar.gz" 2>/dev/null || shasum -a 256 "$tmp/catude.tar.gz") | cut -d' ' -f1)
[ "$want" = "$have" ] || { echo "checksum mismatch: refusing to install"; exit 1; }
tar -xzf "$tmp/catude.tar.gz" -C "$tmp"
"$tmp/catude" install "$@" </dev/tty
