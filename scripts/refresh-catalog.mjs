// Refreshes the GitHub facts of internal/catalog/catalog.json: stars, date of the last push and whether
// the repository is archived. It never adds or removes entries: a repository that no longer exists is
// listed so a person decides what to do with it.
//
// usage: GITHUB_TOKEN=… node scripts/refresh-catalog.mjs [--dry-run] [--limit=N]
// (without a token GitHub allows 60 requests an hour, fewer than the catalog needs)
import { readFileSync, writeFileSync } from "node:fs";

const file = new URL("../internal/catalog/catalog.json", import.meta.url);
const text = readFileSync(file, "utf8");
const cat = JSON.parse(text);
const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN;
const dry = process.argv.includes("--dry-run");

const withRepo = [...cat.marketplaces, ...cat.skills, ...cat.tools].filter((e) => /^[\w.-]+\/[\w.-]+$/.test(e.repo || ""));
const limit = Number((process.argv.find((a) => a.startsWith("--limit=")) || "").slice(8)) || Infinity; // for trying it out
const repos = [...new Set(withRepo.map((e) => e.repo.toLowerCase()))].slice(0, limit);
const facts = new Map();
const missing = [];
for (const repo of repos) {
  const r = await fetch("https://api.github.com/repos/" + repo, {
    headers: { accept: "application/vnd.github+json", "user-agent": "catude-catalog-refresh", ...(token ? { authorization: "Bearer " + token } : {}) },
  });
  if (r.status === 404) {
    missing.push(repo);
    continue;
  }
  if (!r.ok) {
    console.error(`${repo}: HTTP ${r.status} ${r.headers.get("x-ratelimit-remaining") === "0" ? "(rate limit: set GITHUB_TOKEN)" : ""}`);
    process.exit(2);
  }
  const j = await r.json();
  facts.set(repo, { stars: j.stargazers_count, pushed: (j.pushed_at || "").slice(0, 10), archived: !!j.archived, moved: j.full_name.toLowerCase() !== repo ? j.full_name : "" });
}

let changed = 0;
const moved = [];
for (const e of withRepo) {
  const f = facts.get(e.repo.toLowerCase());
  if (!f) continue;
  const before = JSON.stringify([e.stars, e.pushed, e.archived]);
  e.stars = f.stars;
  delete e.stars_approx;
  e.pushed = f.pushed;
  if (f.archived) e.archived = true;
  else delete e.archived;
  if (f.moved) moved.push(`${e.repo} → ${f.moved}`);
  if (JSON.stringify([e.stars, e.pushed, e.archived]) !== before) changed++;
}
cat.generated = new Date().toISOString().slice(0, 10);

const year = Date.now() - 365 * 864e5;
const stale = withRepo.filter((e) => e.archived || (e.pushed && Date.parse(e.pushed) < year)).map((e) => `${e.name} (${e.archived ? "archived" : "last push " + e.pushed})`);
console.log(`${repos.length} repositories · ${changed} entries updated`);
for (const [title, xs] of [["no longer on GitHub", missing], ["moved", moved], ["archived or no push in a year", [...new Set(stale)]]]) {
  if (xs.length) console.log(`\n${title} (${xs.length}):\n` + xs.map((x) => "- " + x).join("\n"));
}
if (!dry) {
  // keep the file's layout (its indentation)
  const indent = (text.match(/^\{\n( +)"/) || [, ""])[1].length;
  writeFileSync(file, JSON.stringify(cat, null, indent || undefined) + "\n");
}
