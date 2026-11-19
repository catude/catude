// Builds internal/claudecfg/reference.json — every documented setting of the Claude apps — from a list
// extracted from Anthropic's documentation:
//   code-settings   settings.json keys        https://code.claude.com/docs/en/settings-reference
//   code-global     ~/.claude.json keys       same page ("Global config")
//   code-env        environment variables     https://code.claude.com/docs/en/env-vars (+ monitoring-usage)
//   desktop         claude_desktop_config.json https://modelcontextprotocol.io/docs/develop/connect-local-servers
//   desktop-policy  Claude Desktop policies   https://support.claude.com/en/articles/12622667 and
//                                             https://claude.com/docs/third-party/claude-desktop/configuration
//
// usage: node scripts/build-settings.mjs <extracted.json>
import { readFileSync, writeFileSync } from "node:fs";

const src = JSON.parse(readFileSync(process.argv[2], "utf8"));
const apps = new Set(["code-settings", "code-global", "code-env", "desktop", "desktop-policy"]);
const out = [];
for (const e of src) {
  if (!apps.has(e.app) || typeof e.key !== "string" || !e.key) continue;
  const r = { app: e.app, key: e.key, type: e.type || "string" };
  if (e.type === "enum" && Array.isArray(e.values) && e.values.length) r.values = e.values.map(String);
  if (e.default !== null && e.default !== undefined) r.default = typeof e.default === "string" ? e.default : JSON.stringify(e.default);
  r.desc = String(e.desc || "").slice(0, 160);
  r.category = e.category || "Other";
  if (e.scope) r.scope = e.scope;
  if (e.deprecated) r.deprecated = true;
  if (e.documented === false) r.undocumented = true;
  out.push(r);
}
out.sort((a, b) => a.app.localeCompare(b.app) || a.category.localeCompare(b.category) || a.key.localeCompare(b.key));
writeFileSync("internal/claudecfg/reference.json", JSON.stringify({ updated: new Date().toISOString().slice(0, 10), settings: out }, null, 0) + "\n");
const count = {};
for (const r of out) count[r.app] = (count[r.app] || 0) + 1;
console.log("wrote internal/claudecfg/reference.json", count);
