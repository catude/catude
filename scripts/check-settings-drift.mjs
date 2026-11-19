// Compares internal/claudecfg/reference.json with Anthropic's documentation and lists the settings and
// environment variables that appeared or disappeared. It changes nothing: a person (or Claude) reads the
// new entries in the docs and adds them with scripts/build-settings.mjs.
//
// Sources (the Markdown versions of the pages):
//   https://code.claude.com/docs/en/settings-reference.md  every settings.json and ~/.claude.json key ("### `key`")
//   https://code.claude.com/docs/en/env-vars.md            environment variables ("| `NAME` |")
//   https://code.claude.com/docs/en/monitoring-usage.md    OpenTelemetry variables
//
// usage: node scripts/check-settings-drift.mjs [--markdown]
// exit code: 0 no drift, 1 drift found, 2 a page could not be read
import { readFileSync } from "node:fs";

const base = "https://code.claude.com/docs/en/";
async function page(name) {
  const r = await fetch(base + name + ".md", { headers: { "user-agent": "catude-drift-check" } });
  if (!r.ok) throw new Error(`${name}: HTTP ${r.status}`);
  return r.text();
}

let docs;
try {
  docs = await Promise.all(["settings-reference", "env-vars", "monitoring-usage"].map(page));
} catch (e) {
  console.error("could not read the documentation:", e.message);
  process.exit(2);
}
const [settingsDoc, envDoc, otelDoc] = docs;
const ref = JSON.parse(readFileSync(new URL("../internal/claudecfg/reference.json", import.meta.url), "utf8"));

const top = (k) => k.split(".")[0];
const documented = (app) => ref.settings.filter((s) => s.app === app && !s.undocumented).map((s) => s.key);

// settings: settings.json and ~/.claude.json keys share the reference page
const docKeys = new Set([...settingsDoc.matchAll(/^#{2,4} `([^`]+)`/gm)].map((m) => m[1]));
const refKeys = new Set([...documented("code-settings"), ...documented("code-global")]);
const refTops = new Set([...refKeys].map(top));
const docTops = new Set([...docKeys].map(top));
const newSettings = [...docKeys].filter((k) => !refKeys.has(k) && !refTops.has(k)).sort();
const goneSettings = [...refTops].filter((k) => !docTops.has(k)).sort();

// environment variables: only names in the families Claude Code uses (tables also show examples such as
// NPM_TOKEN that are not Claude Code settings)
const refEnv = new Set(documented("code-env"));
const families = new Set([...refEnv].map((k) => k.split("_")[0]));
const docEnv = new Set(
  [...envDoc.matchAll(/^\| `([A-Z][A-Z0-9_]+)`/gm), ...otelDoc.matchAll(/^\| `([A-Z][A-Z0-9_]+)`/gm)]
    .map((m) => m[1])
    .filter((k) => families.has(k.split("_")[0])),
);
const newEnv = [...docEnv].filter((k) => !refEnv.has(k)).sort();
const goneEnv = [...refEnv].filter((k) => !docEnv.has(k)).sort();

const drift = newSettings.length + goneSettings.length + newEnv.length + goneEnv.length;
const list = (title, xs) => (xs.length ? `### ${title} (${xs.length})\n\n${xs.map((x) => "- `" + x + "`").join("\n")}\n\n` : "");
if (process.argv.includes("--markdown")) {
  console.log(
    drift
      ? `The settings reference catude ships (updated ${ref.updated}) differs from Anthropic's documentation.\n\n` +
          list("New settings", newSettings) +
          list("Settings no longer documented", goneSettings) +
          list("New environment variables", newEnv) +
          list("Environment variables no longer documented", goneEnv) +
          "Read each entry in the docs, then update the extracted list and run `node scripts/build-settings.mjs`. " +
          "A setting that left the docs may still work: mark it deprecated rather than deleting it when Anthropic says so."
      : `No drift: reference.json (updated ${ref.updated}) matches the documentation.`,
  );
} else {
  console.log(JSON.stringify({ updated: ref.updated, newSettings, goneSettings, newEnv, goneEnv }, null, 2));
}
process.exit(drift ? 1 : 0);
