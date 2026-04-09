# bibcheck Eval Harness

This document describes the corpus-oriented evaluation workflow implemented under `bibcheck eval`.

## Overview

The eval harness is a local, file-backed workflow for running `bibcheck` across a corpus of PDFs, storing durable run artifacts, adding human annotations, and computing aggregate reports.

The workflow is organized around four commands:

- `bibcheck eval discover`
- `bibcheck eval run`
- `bibcheck eval review`
- `bibcheck eval report`

The eval workspace is plain JSON on disk. You can inspect and edit the files directly.

## Corpus Layout

`discover` expects a directory tree shaped like:

```text
<corpus-root>/
  <venue-a>/
    paper1.pdf
    paper2.pdf
  <venue-b>/
    paper3.pdf
```

Venue ids are derived from the immediate subdirectory names.

Paper ids are the relative PDF paths under the corpus root, for example:

- `venue-a/paper1.pdf`
- `venue-b/paper3.pdf`

That `paper_id` is the stable paper identity used throughout the eval workspace.

Entry identity is:

- `paper_id`
- `entry_number`

There is no automatic remapping if entry numbering changes in a future rerun.

## Workspace Layout

By default, the eval workspace lives in `./evaldata`. You can override it with `--workspace`.

The current on-disk structure is:

```text
evaldata/
  corpus.json
  annotations/
    <paper-id>.json
  runs/
    <run-id>/
      run.json
      results/
        <paper-id>.json
      reports/
        summary.json
```

Examples:

- `evaldata/corpus.json`
- `evaldata/annotations/venue1/paper.pdf.json`
- `evaldata/runs/20260409t115514z/run.json`
- `evaldata/runs/20260409t115514z/results/venue1/paper.pdf.json`
- `evaldata/runs/20260409t115514z/reports/summary.json`

## Data Files

### `corpus.json`

Created by `bibcheck eval discover`.

It stores:

- corpus root
- discovered venues
- discovered papers

### `runs/<run-id>/run.json`

Created by `bibcheck eval run`.

It stores:

- run id
- timestamps
- git sha
- selected pipeline
- corpus root
- configured providers
- per-paper status

Paper statuses are:

- `pending`
- `running`
- `done`
- `error`

### `runs/<run-id>/results/<paper-id>.json`

Created by `bibcheck eval run`.

Each file stores one paper’s machine-readable results, including:

- `paper_id`
- `venue_id`
- `entry_number`
- extracted entry text
- normalized final decision
- primary matched source
- source statuses/details
- summary state/comment

Current normalized decisions are:

- `match_found`
- `no_match`
- `error`

### `annotations/<paper-id>.json`

Created or updated by `bibcheck eval review`, or by direct manual edits.

Each file stores human labels for entries within one paper.

Current annotation fields include:

- `entry_number`
- `label`
- `note`
- `reviewer`
- `timestamp`
- `canonical_reference_text`

Current labels are:

- `tp`
- `fp`
- `fn`
- `tn`

### `runs/<run-id>/reports/summary.json`

Created by `bibcheck eval report`.

It stores aggregate metrics derived from stored paper results and annotations.

## Command Reference

### `bibcheck eval discover`

Usage:

```bash
bibcheck eval discover <corpus-root> [--workspace evaldata]
```

This command scans the corpus root, discovers venues and PDFs, and writes `corpus.json`.

Example:

```bash
bibcheck eval discover /data/my-corpus
```

### `bibcheck eval run`

Usage:

```bash
bibcheck eval run [--workspace evaldata]
```

This creates a new run from `corpus.json`, processes papers independently, and writes:

- `run.json`
- per-paper result JSON files

#### Resume

```bash
bibcheck eval run --resume <run-id> [--workspace evaldata]
```

Resume behavior:

- `running` papers are converted back to `pending`
- `done` papers are skipped by default
- `error` papers are skipped by default

To retry errored papers:

```bash
bibcheck eval run --resume <run-id> --retry-errors
```

#### Venue and Paper Filters

You can restrict a run or resumed run with exact-match ids:

```bash
bibcheck eval run --venue venue-a
bibcheck eval run --paper venue-a/paper1.pdf
bibcheck eval run --resume <run-id> --venue venue-a
```

Filter semantics:

- `--venue` matches `venue_id`
- `--paper` matches `paper_id`

Multiple flags of the same kind are allowed.

#### Entry-Level Reruns

You can target specific entry numbers with:

```bash
bibcheck eval run --resume <run-id> --entry 2
bibcheck eval run --resume <run-id> --paper venue-a/paper1.pdf --entry 2 --entry 5
```

Current semantics:

- `--entry` requires `--resume`
- selected entries are reanalyzed within matching papers
- refreshed entries are merged back into the existing per-paper result file
- non-targeted entries remain unchanged

This is intended for surgical reruns after pipeline or prompt changes.

### `bibcheck eval review`

Usage:

```bash
bibcheck eval review <run-id> [flags]
```

Current review flow is prompt-driven over stdin/stdout. It:

- loads stored paper results from the selected run
- shows entry text, decision, summary state, source details, and any prior annotation
- prompts for a new label or skip/quit
- writes accepted edits immediately to `annotations/<paper-id>.json`

Supported flags:

- `--workspace <dir>`
- `--venue <venue-id>`
- `--paper <paper-id>`
- `--decision match_found|no_match|error`
- `--unreviewed`
- `--reviewer <name-or-initials>`

Review notes:

- blank label keeps the existing annotation if one exists
- `skip` leaves the current entry unchanged
- `quit` exits the session
- for note and canonical reference prompts:
  - blank keeps the existing value
  - `-` clears the field

### `bibcheck eval report`

Usage:

```bash
bibcheck eval report <run-id> [--workspace evaldata]
```

This command loads:

- `run.json`
- all `done` paper result files for that run
- matching annotation files

Then it writes and prints `reports/summary.json`.

The current report includes:

- run-level entry counts by decision
- run-level summary-state counts
- reviewed vs unreviewed coverage
- overall confusion matrix from annotations
- overall precision / recall / F1
- per-paper counts and rates
- per-venue counts and rates

If an annotation file does not exist for a paper, its entries are treated as unreviewed.

## Typical Workflow

### 1. Discover the corpus

```bash
bibcheck eval discover /data/corpus
```

### 2. Run analysis

```bash
bibcheck eval run
```

or, with an explicit workspace:

```bash
bibcheck eval run --workspace /tmp/my-eval
```

### 3. Review entries

```bash
bibcheck eval review <run-id> --unreviewed --reviewer ab
```

### 4. Generate a report

```bash
bibcheck eval report <run-id>
```

### 5. Targeted reruns when needed

Rerun one paper:

```bash
bibcheck eval run --resume <run-id> --paper venue-a/paper1.pdf --retry-errors
```

Rerun specific entries:

```bash
bibcheck eval run --resume <run-id> --paper venue-a/paper1.pdf --entry 2 --entry 5
```

## Notes and Current Limits

- The eval workspace is the source of truth. No database or remote service is involved.
- Annotation files are intentionally plain JSON and can be edited directly.
- `eval review` is currently prompt-driven, not a full-screen TUI.
- Entry-level reruns are resume-only so updated entries can be merged into existing paper results.
- Report metrics depend on annotations. Without labels, coverage is tracked but confusion metrics remain unpopulated.
- There is currently no run-to-run comparison mode in `eval report`.

## Practical Tips

- Commit `evaldata/` only if you actually want run artifacts and annotations in git.
- If you only want persistent annotations but not transient runs, you may want to manage the workspace path explicitly.
- When reviewing, use `--unreviewed` to avoid repeatedly walking already-labeled entries.
- When debugging a single extraction or lookup regression, combine `--resume`, `--paper`, and `--entry`.
