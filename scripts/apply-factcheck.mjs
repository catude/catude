// Applies an adversarial fact-check to a research file of presets.
//   node scripts/apply-factcheck.mjs research/presets-a.json factcheck-a.json
// - role / philosophy / steps marked "partly" are replaced by the corrected text; "unsupported" steps
//   are dropped (unless a correction is given)
// - tools, MCP servers, plugins and artifacts listed for removal are removed
// - settings corrections replace the old values; notes are replaced, not appended
// - what users ask for is kept, with its link, so the gallery can show it
// - every claim the check could not confirm is added to "unverified"
// The research file is rewritten in place; run scripts/build-presets.mjs afterwards.
import { readFileSync, writeFileSync } from "node:fs";

const [file, check] = process.argv.slice(2);
if (!file || !check) {
  console.error("usage: node scripts/apply-factcheck.mjs RESEARCH.json FACTCHECK.json");
  process.exit(2);
}
const presets = JSON.parse(readFileSync(file, "utf8"));
const fc = new Map(JSON.parse(readFileSync(check, "utf8")).map((c) => [c.id, c]));
let changed = 0;
for (const p of presets) {
  const c = fc.get(p.id);
  if (!c) {
    console.log(`${p.id}: not checked`);
    continue;
  }
  const log = [];
  if (c.role_ok === false && c.role_fix) {
    p.role = c.role_fix;
    log.push("role");
  }
  if (c.philosophy && c.philosophy.status !== "supported" && c.philosophy.fix) {
    p.philosophy = c.philosophy.fix;
    log.push("philosophy");
  }
  if (Array.isArray(c.process) && c.process.length) {
    const byStep = new Map(c.process.map((s) => [s.step, s]));
    const next = [];
    for (const step of p.process || []) {
      const s = byStep.get(step);
      if (!s || s.status === "supported") next.push(step);
      else if (s.fix) {
        next.push(s.fix);
        log.push("step");
      } else if (s.status === "unsupported") log.push("step dropped");
      else next.push(step);
    }
    p.process = next;
  }
  const drop = (arr, names, key) => {
    if (!names || !names.length || !arr) return arr;
    const set = new Set(names.map((n) => n.toLowerCase()));
    const out = arr.filter((x) => !set.has(String(key ? x[key] : x).toLowerCase()));
    if (out.length !== arr.length) log.push(`${arr.length - out.length} removed`);
    return out;
  };
  p.tools = drop(p.tools, c.tools_remove);
  p.mcp = drop(p.mcp, c.mcp_remove, "name");
  p.plugins = drop(p.plugins, c.plugins_remove, "id");
  for (const a of c.artifacts_remove || []) {
    const k = a.kind === "skill" ? "skills" : a.kind === "agent" ? "agents" : "commands";
    p[k] = drop(p[k], [a.name], "name");
  }
  const sf = c.settings_fix || {};
  p.settings = p.settings || {};
  for (const k of ["model", "effortLevel", "notes"]) {
    if (sf[k]) {
      p.settings[k] = sf[k];
      log.push("settings." + k);
    }
  }
  if (Array.isArray(c.users_want)) p.users_want = c.users_want.filter((u) => u && u.point && /^https:\/\//.test(u.url || ""));
  p.unverified = [...new Set([...(p.unverified || []), ...(c.new_unverified || [])])];
  p.factchecked = new Date().toISOString().slice(0, 10);
  console.log(`${p.id}: ${log.length ? log.join(", ") : "no change"}`);
  changed++;
}
writeFileSync(file, JSON.stringify(presets, null, 1) + "\n");
console.log(`${changed} presets updated in ${file}`);
