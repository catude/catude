// Lists every translatable message of the interface into internal/i18n/locales/messages.json:
// the literals passed to T("…") and F("…") in internal/tui, the tab names, the help screen, and the
// preference values. Each message carries where it is used, as context for translators.
//
// usage: node scripts/i18n-extract.mjs   (from the repository root)
import { readFileSync, writeFileSync, readdirSync } from "node:fs";

const dir = "internal/tui";
const msgs = new Map(); // id → Set of places
const add = (id, where) => {
  if (!id || !/[A-Za-z]/.test(id)) return;
  if (!msgs.has(id)) msgs.set(id, new Set());
  msgs.get(id).add(where);
};
const unquote = (lit) => JSON.parse(lit.replace(/\\'/g, "'")); // Go and JSON escapes agree for what we use

const strLit = /"(?:[^"\\\n]|\\.)*"/g;
for (const f of readdirSync(dir).filter((f) => f.endsWith(".go") && !f.endsWith("_test.go"))) {
  const src = readFileSync(`${dir}/${f}`, "utf8");
  const lines = src.split("\n");
  lines.forEach((line, i) => {
    for (const m of line.matchAll(/\b[TF]\(("(?:[^"\\\n]|\\.)*")/g)) add(unquote(m[1]), `${f}:${i + 1}`);
  });
  // tab names
  const tabs = src.match(/var tabNames = \[\]string\{([^}]*)\}/);
  if (tabs) for (const m of tabs[1].matchAll(strLit)) add(unquote(m[0]), `${f}: tab name`);
  // help screen: section titles and descriptions
  const help = src.match(/var helpSections = [\s\S]*?\n}\n/);
  if (help) {
    for (const m of help[0].matchAll(/\{("(?:[^"\\\n]|\\.)*"), \[\]\[2\]string\{/g)) add(unquote(m[1]), `${f}: help section title`);
    for (const m of help[0].matchAll(/\{"(?:[^"\\\n]|\\.)*", ("(?:[^"\\\n]|\\.)*")\}/g)) add(unquote(m[1]), `${f}: help, what a key does`);
  }
  // preference values shown translated
  for (const name of ["themeChoices", "glyphChoices", "accentChoices"]) {
    const m = src.match(new RegExp(`${name}\\s*=\\s*\\[\\]string\\{([^}]*)\\}`));
    if (m) for (const v of m[1].matchAll(strLit)) add(unquote(v[0]), `${f}: preference value (${name})`);
  }
  for (const m of src.matchAll(/onOff\([^,]+, ("[^"]+"), ("[^"]+")\)/g)) {
    add(unquote(m[1]), `${f}: preference value`);
    add(unquote(m[2]), `${f}: preference value`);
  }
}
// kinds of work in the Models tab
const models = readFileSync("internal/models/models.go", "utf8").match(/var Categories = \[\]string\{([^}]*)\}/);
if (models) for (const m of models[1].matchAll(strLit)) add(unquote(m[0]), "models.go: kind of work (Models tab)");

const out = [...msgs.entries()].sort((a, b) => a[0].localeCompare(b[0])).map(([id, w]) => ({ id, where: [...w].sort().slice(0, 4) }));
writeFileSync("internal/i18n/locales/messages.json", JSON.stringify(out, null, 1) + "\n");
console.log(`wrote internal/i18n/locales/messages.json: ${out.length} messages`);
