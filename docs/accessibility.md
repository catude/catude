# Accessibility and terminal compatibility

catude should look right and be usable in any terminal, with any font, with or without colour, and without
a mouse.

## Glyphs: no tofu, no broken emoji

Terminals and fonts disagree on symbols. Emoji-capable characters (✔ ⚠ ⚙ ⏳) are drawn two cells wide by some
fonts and one by others, which breaks columns; console fonts such as Consolas, Lucida Console or the Linux text
console lack many symbols and show empty boxes. catude avoids both:

| Set | Chosen when | Contains |
|---|---|---|
| `unicode` | Windows Terminal, VS Code, JetBrains, iTerm2, WezTerm, kitty, Alacritty, GNOME/Konsole, macOS Terminal; any Unix terminal with a UTF-8 locale | narrow text symbols only: ✓ ▲ ✗ ○ ● › ≡ and rounded borders, never emoji |
| `safe` | the classic Windows console (conhost) and the Linux text console (`TERM=linux`) | only characters from WGL4, present in every console font: √ ▲ x ○ ● ► and square borders |
| `ascii` | a non-UTF-8 locale or `TERM=dumb` | 7-bit ASCII only: `+ ! x - * >`, `+--+` borders, accents transliterated |

The interface is drawn once and translated cell for cell, so the layout is identical in every set (a test
checks that no line changes width). Text from your files (skill names, descriptions, session titles) is
cleaned before layout: emoji, variation selectors, zero-width characters and bidi controls are removed,
because their width differs between terminals and bidi controls can hide text.

Force a set with `--glyphs unicode|safe|ascii` or `CATUDE_GLYPHS`. The command line uses the same set.

## Colour and contrast

| Theme | |
|---|---|
| `auto` (default) | adapts to light or dark backgrounds |
| `dark` · `light` | fixed palettes, for terminals that misreport their background |
| `high-contrast` | yellow/white on dark or black on light, underlined tabs, no grey text |
| `mono` | no colour at all; selection is reverse video, headings bold and underlined |

`NO_COLOR` (or a terminal without colour) selects `mono` automatically. Choose with `--theme` or
`CATUDE_THEME`.

Colour never carries meaning alone: every status has a glyph (✓ ▲ ✗ ○), the selected row has a `›` cursor,
and the active work profile is marked with `●` and named in the header.

## Keyboard and mouse

- Everything works from the keyboard. `?` lists every key for every tab.
- `1`…`9` and `0` jump straight to the first ten tabs; `←` `→` and `tab` move between all of them. When the
  tabs do not fit, the bar scrolls with the active tab and shows `‹ ›`.
- `↑` `↓` / `j` `k`, `pgup` `pgdown`, `g` `G`. The mouse wheel scrolls; clicking a tab opens it.
- Destructive actions need the same key twice (`d d`, `c c`, `D D`, `i i`) and say what will happen first.

## Make it yours

Press `,` (or `F2`) for preferences, saved in `config.json` next to your profiles:

- **Tabs**: tick the tabs you want to see. Number keys and arrows follow what is visible.
- **Lite view**: a second, shorter set of tabs. Switch to it in preferences, start with `catude --lite`, or
  run `ctlite` (installed next to `catude` and `ct`).
- **Theme, accent colour and glyphs**, applied as you change them. Command-line flags still win for one run.
- **Animations** and the **key hints** line can be turned off.

## Motion

Spinners show that background work is running. Set `CATUDE_REDUCE_MOTION=1` for a static indicator.

## Screen readers

Full-screen terminal interfaces are hard for screen readers. Every view has a plain-text command-line
equivalent: `catude doctor`, `ls`, `audit`, `sessions`, `context`, `profile list|show`, `discover` (all with
`--json` too), and `catude snapshot` renders every tab as plain text.
