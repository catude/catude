# Translations

catude's interface is available in **English, Español, 日本語, 简体中文, 繁體中文, 한국어, Deutsch, Français,
Português (Brasil) and Русский**.

## Choosing the language

- By default catude follows your system: `CATUDE_LANG`, then `LC_ALL` / `LC_MESSAGES` / `LANG`, then
  Windows' preferred display languages.
- In the TUI: `,` (preferences) → **Language**. Saved in catude's preferences file.
- For one run: `catude --lang ja` (codes: `en es ja zh-Hans zh-Hant ko de fr pt-BR ru`).

Chinese, Japanese and Korean need a terminal font with those characters: Windows Terminal, recent macOS
and Linux terminals have one. The old Windows console (conhost) may not; preferences warn about it.
Columns are measured in terminal cells, so double-width characters keep tables and cards aligned.

## What is translated

The tab bar, the key hints, the help screen (`?`), preferences, the Live tab, the kinds of work of the
Models tab and the headings of every tab. Longer explanations inside some tabs (for example the
description of each setting in Config, which comes from Anthropic's English documentation) and the
command-line output are still in English.

## Where the words come from

Nothing was translated from scratch. Each language follows, in this order:

1. **Anthropic's own localized Claude Code documentation** (`https://code.claude.com/docs/<lang>/…`) for
   Claude concepts: skills, subagents, hooks, MCP servers, plugins, settings, context window, sessions…
   When Anthropic keeps a word in English (for example "skills" and "hooks" in Spanish, French and Chinese
   page titles), catude does too.
2. **Visual Studio Code's language packs** (Microsoft terminology, [vscode-loc](https://github.com/microsoft/vscode-loc))
   for general interface words: settings, preferences, theme, filter, reload, keyboard shortcuts, tab…
3. **lazygit's translations** ([pkg/i18n/translations](https://github.com/jesseduffield/lazygit/tree/master/pkg/i18n/translations))
   for terminal-UI phrasing of key hints, where a language has one.

Traditional Chinese uses Taiwan terminology (工作階段, 設定檔, 外掛程式, 索引標籤…), not a character conversion of
Simplified Chinese.

The glossary of each language lists every key term, its translation, its source and the exact string seen
there, and marks the few terms no source had (translated by analogy):
[es](translations/es.md) · [ja](translations/ja.md) · [zh-Hans](translations/zh-Hans.md) ·
[zh-Hant](translations/zh-Hant.md) · [ko](translations/ko.md) · [de](translations/de.md) ·
[fr](translations/fr.md) · [pt-BR](translations/pt-BR.md) · [ru](translations/ru.md)

## For contributors

- Messages are written in English in the code with `T("…")` or `F("… %s …", x)` (gettext style).
- `node scripts/i18n-extract.mjs` lists them in `internal/i18n/locales/messages.json`, with where each is
  used.
- Catalogs are `internal/i18n/locales/<code>.json` (`"English": "translation"`). An untranslated message
  shows in English, never blank.
- `go test ./internal/i18n` fails when a catalog misses a message, keeps one that is no longer used, changes
  a placeholder (`%s`, `%d`…), translates a key name in a key hint (`enter`, `esc`, `tab`…) or a product
  or command name (catude, Claude Code, MCP, `claude mcp list`…).
- Corrections from native speakers are welcome: please say which source supports the change.
