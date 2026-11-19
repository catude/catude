#!/usr/bin/env bash
# Records the README frames on an invented machine: user "alex", invented projects, skills, MCP servers
# and sessions. Runs in a user namespace where /home/alex is an empty folder, with a clean PATH, so no
# real path, user name or file can show up.
#
# usage: scripts/demo/sandbox.sh BIN_DIR OUT_DIR
#   BIN_DIR holds Linux builds of `demo` (./scripts/demo) and `catude` (./cmd/catude), and models.json
#           (a copy of catude's Epoch AI cache: public data, used by the Models screenshot)
#   OUT_DIR receives NNN.ansi frames and frames.json
set -euo pipefail
BIN=$(cd "$1" && pwd)
OUT=$(mkdir -p "$2" && cd "$2" && pwd)
ROOT=$(mktemp -d)
mkdir -p "$ROOT/home/alex"
install -m 755 "$BIN/demo" "$BIN/catude" "$ROOT/"
trap 'rm -rf "$ROOT"' EXIT

unshare --user --map-root-user --mount env -i ROOT="$ROOT" BIN="$BIN" OUT="$OUT" bash -c '
set -euo pipefail
mount --bind "$ROOT/home" /home
export HOME=/home/alex USER=alex LOGNAME=alex TERM=xterm-256color LANG=en_US.UTF-8 CATUDE_NO_WSL=1 PATH=/usr/local/bin:/usr/bin:/bin
cd /home/alex
"$ROOT/demo" fixture -models "$BIN/models.json"
cd /home/alex/code/storefront
echo y | "$ROOT/catude" profile new frontend --skills frontend-design,code-review,tdd --agents reviewer --mcp playwright >/dev/null
echo y | "$ROOT/catude" profile new backend --skills api-design,db-migrations,tdd --agents test-runner,explorer --mcp github,postgres >/dev/null
echo y | "$ROOT/catude" profile new writing --skills writing-docs,release-notes --agents docs-writer >/dev/null
"$ROOT/demo" frames -out "$OUT"
"$ROOT/catude" graph html > "$OUT/graph.html"
"$ROOT/catude" rig export -o "$OUT/rigcard-alex.json" >/dev/null
'
# nothing of this machine may appear in a frame or an export
if grep -l -i -E "$(id -un)|/mnt/|AppData" "$OUT"/*.ansi "$OUT"/*.html "$OUT"/*.json; then
  echo "frames contain local data: not using them" >&2
  exit 1
fi
echo "frames in $OUT"
