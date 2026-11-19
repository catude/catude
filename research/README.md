# Research data

Source data behind the practitioner presets (`internal/presets/presets.json`), kept so anyone can check
where each preset comes from:

- `presets-a.json`, `presets-b.json`: what each practitioner has published about their setup, with links.
- `factcheck-a.json`, `factcheck-b.json`: a second pass that checked those claims against the sources.

`scripts/build-presets.mjs` and `scripts/apply-factcheck.mjs` turn them into `internal/presets/presets.json`.
