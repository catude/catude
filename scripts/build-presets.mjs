// Builds internal/presets/presets.json from research files (arrays of presets).
//   node scripts/build-presets.mjs research-a.json [research-b.json…]
// Keeps only safe artifact names, drops duplicates, and normalises fields. Every artifact must be
// re-verified against GitHub (see docs/presets.md) before a release.
import { readFileSync, writeFileSync } from "node:fs";
const safe = /^[a-z0-9][a-z0-9._-]{0,63}$/i;
const out = [];
const seenIds = new Set();
for (const f of process.argv.slice(2)) {
  for (const p of JSON.parse(readFileSync(f, "utf8"))) {
    if (!p.id || seenIds.has(p.id) || !(p.sources || []).length) continue;
    seenIds.add(p.id);
    const clean = (arr) => {
      const seen = new Set();
      return (arr || []).filter((a) => safe.test(a.name) && !seen.has(a.name) && seen.add(a.name) && a.repo && a.path)
        .map((a) => ({ name: a.name, repo: a.repo, path: a.path, commit: a.commit || "", license: a.license || "", description: a.description || "" }));
    };
    out.push({
      id: p.id, name: p.name, handle: p.handle || "", role: p.role || "",
      sources: (p.sources || []).filter((s) => /^https:\/\//.test(s.url)).map((s) => ({ title: s.title, url: s.url, date: s.date || "" })),
      philosophy: p.philosophy || "", process: p.process || [], tools: p.tools || [],
      mcp: (p.mcp || []).map((m) => ({ name: m.name, add: m.add || "", url: m.url || "" })),
      marketplaces: (p.marketplaces || []).map((m) => ({ name: m.name, repo: m.repo, commit: m.commit || "", license: m.license || "" })),
      plugins: (p.plugins || []).map((x) => ({ id: x.id, url: x.url || "" })),
      skills: clean(p.skills), agents: clean(p.agents), commands: clean(p.commands),
      settings: { model: p.settings?.model || "", effortLevel: p.settings?.effortLevel || "", notes: p.settings?.notes || "" },
      unverified: p.unverified || [],
      users_want: (p.users_want || []).filter((u) => (u.url || "").startsWith("https://")).map((u) => ({ point: u.point, url: u.url })),
      factchecked: p.factchecked || "",
    });
  }
}
out.sort((a, b) => a.name.localeCompare(b.name));
writeFileSync("internal/presets/presets.json", JSON.stringify(out, null, 1) + "\n");
console.log(out.length, "presets:", out.map((p) => `${p.id} (${p.skills.length}s/${p.agents.length}a/${p.commands.length}c)`).join(", "));
