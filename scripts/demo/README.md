# README images

The screenshots and GIFs of the README are made from an **invented machine** (user `alex`, projects
`storefront`, `payments-api` and `infra`, invented skills, agents, MCP servers, plugins and sessions), never
from a real one.

1. **Record** (Linux or WSL, no root needed: a user namespace makes `/home/alex` an empty folder):

   ```sh
   GOOS=linux go build -o /tmp/lr/demo ./scripts/demo
   GOOS=linux go build -o /tmp/lr/catude ./cmd/catude
   cp "$(catude-cache)/models-epoch.json" /tmp/lr/models.json   # public Epoch AI data, for the Models shot
   scripts/demo/sandbox.sh /tmp/lr /tmp/lr/frames
   ```

   `demo fixture` builds the home; `demo frames` runs catude on it with scripted keys
   (`internal/tui/demo.go`) and writes every screen with its colours. The script refuses frames that
   contain the real user name or Windows paths.

2. **Render** (Windows): `scripts/demo/render.ps1 -Frames <frames> -Demo <demo.exe>` turns each frame into
   an SVG terminal window (box-drawing characters drawn as lines, every run placed on its cell), takes a
   PNG of it with Edge headless, and assembles `assets/demo.gif`, `assets/live.gif` and
   `assets/screenshots/*.png`. On Linux or macOS, any headless browser does step 2.

The scripted keys live in `scripts/demo/main.go` (`sets`): change them to show something else.
