# Models: which Claude model and effort level is stronger

The **Models** tab (and `catude models`) answers questions like *"is Opus 5.5 at high effort better
than Fable 5.2 at medium?"* for the kind of work you choose, and tells you, from the model you
actually use, whether something as strong costs less or something clearly stronger is worth it.

It needs no account and no API key: everything comes from public data.

## Where the data comes from

| | What | Licence |
|---|---|---|
| [Epoch AI Benchmarking Hub](https://epoch.ai/benchmarks) | benchmark results per model **and effort level** (`claude-opus-5-5_high`), and the Epoch Capabilities Index (ECI) per model. Downloaded from `https://epoch.ai/data/benchmark_data.zip` (about 2 MB) | Epoch's data: CC BY 4.0. Results Epoch collected from other projects (ARC Prize, Cursor, …) keep their source, shown with them |
| [models.dev](https://models.dev) | prices per million tokens (`https://models.dev/api.json`) | MIT |

- Downloaded only when you open the tab or run the command, then kept for a day in your private cache
  (`<user cache dir>/catude/state/models-epoch.json`, mode 0600). `r` downloads again.
- No credentials or personal data are sent: plain GET requests with the user agent `catude`.
- Files that Epoch took from Artificial Analysis are skipped (their terms forbid this use).
- catude ships none of this data and is not affiliated with Epoch AI or models.dev.

## How to read it

- **Kind of work** (`t`): all · coding · reasoning · math & science · agents · knowledge. Each benchmark
  belongs to one kind (CursorBench and DeepSWE are coding, ARC-AGI reasoning, CritPt math & science,
  LMCA and DTBench agents…), so the answer can change with the kind of work, as it should.
- **Score**: for each benchmark, a result as a percentage of the best result anyone got on it; the
  score is the average over the benchmarks measured for most Claude versions of that kind (`*` = this
  version was measured on fewer of them).
- **Verdict**: compares two versions only on benchmarks that measured both. A version is "stronger"
  when it wins more of them by at least 3 points; otherwise they are "about even". A verdict that rests
  on a single benchmark says so ("take it as a hint").
- **Cost per task**: when a benchmark publishes it (CursorBench does), the verdict shows it. It is the best
  way to see what a higher effort level costs.
- **ECI**: Epoch's overall index for the model (all effort levels together).
- **$/1M**: blended price, 3 input : 1 output tokens.
- **You use**: the model with most tokens in your transcripts, at the `effortLevel` of your
  `~/.claude/settings.json` (shown at medium when it is not set).

## In the TUI

`↑` `↓` move: the selected version is compared with yours · `t` / `T` kind of work · `space` compare
everything with the selected version instead · `/` filter · `r` download again · `o` open Epoch's site.

## From the command line

```
catude models                                            # ranking and advice on the model you use
catude models --for coding
catude models compare "opus 5.5 high" "fable 5.2 medium"
catude models compare "opus 5.5 high" "fable 5.2 medium" --for agents
catude models refresh
```

Names are matched loosely: `opus 5.5 high`, `claude-opus-5-5:high`, `claude-opus-5-5_high`,
`sonnet-5-5 xhigh`. Without an effort level, medium is used. If a level has no results, the closest one
is used and catude says so.
