# Blink parity harness

Measures how far blink's `--blink` output diverges from the PHP ECS engine on
real-world codebases. The goal is **0% divergence on every corpus**, so blink can
become the default engine without changing any formatting.

## Run one corpus

```bash
blink/parity/run.sh blink/parity/corpora/laravel.json
```

It builds blink from the current checkout, clones the corpus, fixes two copies
(one with `--blink`, one with the PHP engine), diffs them, and prints every
differing file grouped by the ECS checker that caused it. Exit status is non-zero
when the differing share exceeds the corpus's `max_diff_percent`.

Flags: `--workdir DIR` reuses a directory, `--keep` leaves it in place for
inspection (the diffed trees are `ours/` vs `ecs/`, the report is `report.txt`).

## Corpora

Each `corpora/*.json` describes one codebase: `repo`, optional `ref`, the `paths`
to check, ECS `skip` patterns, the prepared `sets`, and the ratchet
`max_diff_percent`. Lower a threshold whenever parity improves; never raise it.

CI runs every corpus on each pull request (`.github/workflows/blink_parity.yaml`)
and prints the categorized report to the job summary.

## Closing the gap

Pick the checker with the highest count in the report, reproduce one of its files
locally with `--keep`, fix the matching blink rule, and re-run. Verify the change
against **every** corpus before lowering a threshold — a fix that helps one
codebase can regress another.
