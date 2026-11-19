// Points the project at its GitHub home: node scripts/set-owner.mjs OWNER [REPO]
//   - go.mod module path becomes github.com/OWNER/REPO, so `go install github.com/OWNER/REPO/cmd/catude@latest`
//     works, and every import is rewritten to match;
//   - the installers, README and docs point at OWNER/REPO instead of the placeholder catude/catude.
// Run it once before the first push, then `go build ./... && go test ./...`.
import { readFileSync, writeFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";

const [owner, repo = "catude"] = process.argv.slice(2);
if (!owner || !/^[A-Za-z0-9-]{1,39}$/.test(owner) || !/^[A-Za-z0-9._-]{1,100}$/.test(repo)) {
  console.error("usage: node scripts/set-owner.mjs OWNER [REPO]");
  process.exit(2);
}
const modulePath = `github.com/${owner}/${repo}`;
const goMod = readFileSync("go.mod", "utf8");
const current = goMod.match(/^module\s+(\S+)/m)[1];

const skip = new Set([".git", "dist", "bin", "research", "node_modules"]);
let changed = 0;
function walk(dir) {
  for (const name of readdirSync(dir)) {
    if (skip.has(name)) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) { walk(p); continue; }
    if (!/\.(go|mod|md|ps1|sh|mjs|json|toml|ya?ml)$/.test(name)) continue;
    let s = readFileSync(p, "utf8");
    const before = s;
    if (name.endsWith(".go")) {
      s = s.replace(new RegExp(`"${current.replace(/[.*+?^${}()|[\]\\/]/g, "\\$&")}/`, "g"), `"${modulePath}/`);
    }
    if (name === "go.mod") s = s.replace(/^module\s+\S+/m, `module ${modulePath}`);
    s = s.split("catude/catude").join(`${owner}/${repo}`);
    if (s !== before) { writeFileSync(p, s); changed++; }
  }
}
walk(".");
console.log(`module ${current} -> ${modulePath}; ${changed} files updated`);
