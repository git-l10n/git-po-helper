# Refactoring Design: agent-run review to Match po/AGENTS.md Task 4

This document describes the refactoring of `RunAgentReview`, `runReviewSingleBatch`, and `runReviewBatched` in `util/agent-run-review.go` so that the implementation follows the execution steps of **Task 4: review translation quality** in `po/AGENTS.md` (see `git-l10n/git-po` repository, or the project’s `po/AGENTS.md` when present).

**Reference**: `po/AGENTS.md` section "Task 4: review translation quality" (steps 1–9), especially:
- Step 1: Check for existing review
- Step 2: Extract entries (`git-po-helper compare` → `po/review.po`)
- Step 3: Prepare review batches (split `po/review.po` into `po/review-batch-<N>.po`)
- Step 4: Check batch files and select current batch
- Step 5: Read context (glossary, etc.)
- Step 6: Review entries (apply corrections to `po/review.po`)
- Step 7: Generate review report → save to `po/review-batch-<N>.json`
- Step 8: Delete `po/review-batch-<N>.po`; repeat from step 4 or go to step 9
- Step 9: Merge and summary (`git-po-helper agent-run report`)

**Scope**: Programmatic flow only. Steps 5 (read context) and 6 (review entries) are performed by the agent; the code only prepares inputs, invokes the agent, and handles outputs.

---

## 1. Current Behavior vs AGENTS.md

### 1.1 Current Implementation Summary

| Component | Current behavior |
|-----------|------------------|
| **RunAgentReview** | (1) Prepares review data via `PrepareReviewData` → writes `po/review.po`. (2) Counts entries; if ≤100 uses single batch, else batched. (3) Builds prompt and runs agent (single or batched). (4) Saves merged JSON to `po/review.json`. No “check existing review” step. |
| **runReviewSingleBatch** | Runs agent once with `source=po/review.po`, parses JSON from stdout. Does **not** write `po/review-batch-1.json`. |
| **runReviewBatched** | Uses a **single** batch file `po/review-batch.po` (no index). For each batch: msg-select into that file, run agent with `source=po/review-batch.po`, parse JSON, remove batch file. Accumulates batch JSONs in memory and merges with `AggregateReviewJSON`; does **not** write `po/review-batch-<N>.json`. Final merged result is returned to caller, which writes `po/review.json`. |
| **Batch naming** | `ReviewDefaultBatchFile` = `po/review-batch.po` (one reused path). AGENTS.md uses `po/review-batch-<N>.po` and `po/review-batch-<N>.json` with N = 1, 2, 3, … |
| **Step 9** | AGENTS.md says run `git-po-helper agent-run report` to merge batch JSONs and show summary. Current code merges in memory and writes `po/review.json`; it does **not** invoke `agent-run report`. |

### 1.2 Gaps to Address

1. **Step 1 (Check existing review)**
   Not implemented. AGENTS.md: if `po/review.po` missing → step 2; if both `po/review.po` and `po/review.json` exist → go to step 9 (merge/summary only); if `po/review.po` exists but `po/review.json` does not → step 4 (resume).

2. **Step 3 (Prepare batches)**
   AGENTS.md: create **multiple** files `po/review-batch-1.po`, `po/review-batch-2.po`, … with dynamic batch sizing. Current code: create one batch at a time in memory (single `po/review-batch.po`), no numbered batch files on disk.

3. **Step 7 (Per-batch JSON)**
   AGENTS.md: save report for current batch to `po/review-batch-<N>.json`. Current code: no per-batch JSON files; only in-memory merge and a single `po/review.json`.

4. **Step 8 (Delete batch PO, then repeat or finish)**
   AGENTS.md: after saving `po/review-batch-<N>.json`, delete `po/review-batch-<N>.po`; if no `po/review-batch-*.po` remain, go to step 9. Current code: deletes the single `po/review-batch.po` after each run but does not use multiple batch files or step 9 as a separate phase.

5. **Step 9 (Merge and summary)**
   AGENTS.md: run `git-po-helper agent-run report` to merge all `po/review-batch-*.json` and display. Current code: merge in `RunAgentReview` / `runReviewBatched` and write `po/review.json`; optional: at the end of `RunAgentReview`, call the same logic as `agent-run report` (or exec the command) to display.

---

## 2. Refactoring Steps (Detailed)

### 2.1 Step 1: Add “Check for existing review” (Step 1 of AGENTS.md)

**Location**: Start of `RunAgentReview` (or a new helper called from it).

**Behavior**:

- If `po/review.po` does **not** exist → continue to “Extract entries” (current `PrepareReviewData` path). No change to call site.
- If both `po/review.po` and `po/review.json` exist → skip review execution; go directly to “Merge and summary” (step 9): e.g. call existing logic that loads/merges batch JSONs and displays (same as `agent-run report`). Return success with result from that. **Note**: call `ReportReviewFromPathWithBatches(outputBase)` and then print the same output as `cmd/agent-run-report.go`.
- If `po/review.po` exists but `po/review.json` does **not** → “Continue previous unfinished review”: go to “Check batch files and select current batch” (step 4). So do **not** overwrite `po/review.po`; do **not** re-run “Extract entries”. Proceed to step 4 (see below).

**Code changes**:

- At the top of `RunAgentReview`, after resolving `reviewPOFile` and `reviewJSONFile` (e.g. via `ReviewOutputPaths(outputBase)`):
  - If `!Exist(reviewPOFile)` → keep current flow (prepare review data → step 2).
  - Else if `Exist(reviewJSONFile)` → run merge/summary only (step 9), then return. (New branch.)
  - Else → “resume” path: skip `PrepareReviewData` and “Prepare batches”; ensure batch numbering and file names match step 3 (see below), then jump to “select current batch” (step 4).

**Note**:

- When “merge and summary only”, the command should print the same lines as `agent-run report` (e.g. “## Review Statistics” and the table).

---

### 2.2 Step 2: Extract entries (unchanged conceptually)

**Location**: `PrepareReviewData(...)` call; equivalent to AGENTS.md “Run `git-po-helper compare` … redirect output to `po/review.po`”.

**Behavior**: No change. Keep using `PrepareReviewData(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile, reviewPOFile)` to produce `po/review.po`.

**Code changes**: None for step 2. Only ensure step 1 does not run “Extract entries” when resuming (when `po/review.po` exists and `po/review.json` does not).

---

### 2.3 Step 3: Prepare review batches (align with AGENTS.md)

**Location**: New or refactored function, e.g. `prepareReviewBatches(reviewPOFile string, minBatchSize int) (batchPOPaths []string, entryCount int, err error)`.

**Behavior** (mirror AGENTS.md script):

- Remove any existing `po/review-batch-*.po` and `po/review-batch-*.json` (within the same output base/dir as `reviewPOFile`).
- Count entries in `po/review.po` (e.g. `countMsgidEntries(reviewPOFile)`; subtract 1 for header if desired to get `entryCount`).
- Batch sizing (AGENTS.md):
  - If `entryCount <= minBatchSize*2` → no split: one logical batch (e.g. use `po/review.po` as “batch 1” or still produce `po/review-batch-1.po` by copy).
  - Else compute `NUM` (e.g. 50, 75, or 100 from the script logic) and `BATCH_COUNT = ceil(entryCount / NUM)`.
- Produce on-disk files:
  - Either: `po/review-batch-1.po`, `po/review-batch-2.po`, … using `msg-select` (or equivalent) with ranges `-NUM`, `START-END`, `START-` as in AGENTS.md.
  - Or when single batch: `cp po/review.po po/review-batch-1.po` (or equivalent).
- Return list of batch PO paths and `entryCount` (and optionally per-batch entry counts for later use).

**Code changes**:

- Replace the current “inline” batch loop in `runReviewBatched` with:
  - A call to `prepareReviewBatches(reviewPOFile, 50)` (or configurable min) that creates `po/review-batch-<N>.po` and returns their paths.
- Introduce or reuse a constant/base for batch file naming, e.g. `po/review-batch-%d.po`, so that `agent-run report` (which already expects `po/review-batch-*.json`) can merge `po/review-batch-*.json` produced in step 7.
- **Note**: AGENTS.md uses `min_batch_size=50` and a specific formula for `NUM` when `ENTRY_COUNT > min_batch_size*2`. But current code uses static values, we should adopt AGENTS.md formula exactly.

---

### 2.4 Step 4: Check batch files and select current batch

**Location**: After `prepareReviewBatches`, or at start of “resume” path.

**Behavior**:

- If there are no `po/review-batch-*.po` files (e.g. `prepareReviewBatches` returned empty list or single-batch case handled differently), proceed to step 9 (merge and summary).
- Otherwise, take the **first** remaining batch (smallest N) as the current batch. “Current batch file” = `po/review-batch-<N>.po`.

**Code changes**:

- In `RunAgentReview`:
  - When not “resume”, after `prepareReviewBatches`, if `len(batchPOPaths) == 0` (or equivalent), skip to step 9.
  - When “resume”, list `po/review-batch-*.po` (or equivalent), sort by N; if none, go to step 9; else pick smallest N.
- The “loop” over batches can be: for N = 1 to BATCH_COUNT (or for each path returned by `prepareReviewBatches`), current batch = `po/review-batch-<N>.po`.

---

### 2.5 Step 5 & 6: Read context / Review entries

**Location**: Agent’s responsibility; no code refactor.

**Behavior**: AGENTS.md step 5 (read glossary, etc.) and step 6 (review entries, apply corrections to `po/review.po`) are done by the agent. The program only provides the batch file and prompt. No change to prompt or file paths is required for this refactor beyond using the correct batch file path and JSON output path (step 7).

---

### 2.6 Step 7: Generate review report → save to `po/review-batch-<N>.json`

**Location**: `runReviewSingleBatch` and the per-batch iteration in `runReviewBatched`.

**Behavior**:

- After the agent runs for the current batch, parse JSON from stdout (current behavior).
- Save the parsed (and optionally validated) JSON to **file** `po/review-batch-<N>.json` (same N as the current `po/review-batch-<N>.po`). Do not merge yet; step 9 will merge all `po/review-batch-*.json`.

**Code changes**:

- **runReviewSingleBatch**:
  - When there is a single batch, treat it as batch 1. After parsing JSON from agent stdout, write that JSON to `po/review-batch-1.json` (path derived from output base, e.g. same dir as `po/review.po`).
  - Optionally still return the parsed `*ReviewJSONResult` for backward compatibility (e.g. for scoring display in the same run), but the canonical storage is the file.
- **runReviewBatched** (per-batch iteration):
  - After `executeReviewAgent` and `parseAndAccumulateReviewJSON`, call `saveReviewJSON(batchJSON, batchJSONPath)` where `batchJSONPath` is e.g. `po/review-batch-<N>.json`.
  - Do **not** merge in memory in `runReviewBatched`; only write per-batch JSON files. Then proceed to step 8 (delete batch PO) and, after the last batch, to step 9.

**Note**: Code should accept whatever valid JSON the agent produces; no need to filter out score-3 issues in the refactor.

---

### 2.7 Step 8: Delete `po/review-batch-<N>.po`; repeat or finish

**Location**: Per-batch loop in `RunAgentReview` / `runReviewBatched`.

**Behavior**:

- After saving `po/review-batch-<N>.json`, delete `po/review-batch-<N>.po`.
- If no `po/review-batch-*.po` remain, proceed to step 9; otherwise repeat from step 4 (select next batch).

**Code changes**:

- In the batch loop, after writing `po/review-batch-<N>.json`, call `os.Remove(batchPOPath)` for the current `po/review-batch-<N>.po`.
- Loop until all batch PO files are processed (no more `po/review-batch-*.po`), then go to step 9.

---

### 2.8 Step 9: Merge and summary

**Location**: End of `RunAgentReview`, after all batches are done (or after “check existing review” when both `po/review.po` and `po/review.json` already exist).

**Behavior**:

- Run the same logic as `git-po-helper agent-run report`: load all `po/review-batch-*.json` (if any), merge with `ReportReviewFromPathWithBatches` (or equivalent), then write merged result to `po/review.json` and display the report (e.g. “## Review Statistics” and the table).
- If there are no batch JSON files (e.g. single-batch case where only `po/review-batch-1.json` was written), merge still runs and will find that one file; if the implementation uses a single `po/review.json` for single-batch, ensure `agent-run report` can read it (current behavior) or that merge step produces `po/review.json` from the single batch file.

**Code changes**:

- After the last batch (or when “merge only” branch is taken in step 1):
  - Call `ReportReviewFromPathWithBatches(outputBase)`. That already aggregates `po/review-batch-*.json` and can write/update `po/review.json`. It returns the aggregated result.
  - Optionally: invoke the same display logic as `cmd/agent-run-report.go` (e.g. extract a small “print report” function and call it from both `RunAgentReview` and `cmd/agent-run-report.go`) so that the user sees the same “## Review Statistics” output without running `agent-run report` separately.
- **Note**: Choose to call the same Go APIs and print the same output.

---

## 3. Function-Level Refactoring Summary

### 3.1 RunAgentReview

- **Beginning**: Add step 1 (check existing review). Three branches: (a) no `po/review.po` → step 2; (b) `po/review.po` and `po/review.json` exist → step 9 only; (c) `po/review.po` exists, no `po/review.json` → resume at step 4.
- **Step 2**: Unchanged: `PrepareReviewData(...)` → `po/review.po`.
- **Step 3**: Call `prepareReviewBatches(reviewPOFile, 50)` to create `po/review-batch-<N>.po` and get list of batch paths. If empty (or single batch with no split), decide whether to go to step 9 or run one batch (see below).
- **Steps 4–8**: Loop over batch paths. For each N: set current batch file = `po/review-batch-<N>.po`; run agent with that file; parse JSON; save to `po/review-batch-<N>.json`; delete `po/review-batch-<N>.po`. For single-batch case (entryCount ≤ 100), can keep a single call to `runReviewSingleBatch` but have it write `po/review-batch-1.json` and then fall through to step 9.
- **Step 9**: Call `ReportReviewFromPathWithBatches(outputBase)` and display the report (reuse report output formatting). Fill `result.ReviewJSON`, `result.ReviewJSONPath`, `result.Score`, etc., from the aggregated result.

### 3.2 runReviewSingleBatch

- **Input**: Add a parameter for the batch index or output path, e.g. `batchJSONPath string` (e.g. `po/review-batch-1.json`).
- **After parsing**: Call `saveReviewJSON(reviewJSON, batchJSONPath)` so that step 7 is satisfied.
- **Return**: Can still return `*ReviewJSONResult` for compatibility; step 9 will re-read from file or caller can use returned value for immediate display if desired.

### 3.3 runReviewBatched

- **Input**: Accept list of batch PO paths (e.g. `[]string`) and optionally the base path for JSON outputs (e.g. `po/review-batch-%d.json`). Or derive from `reviewPOFile` (e.g. dir + base name).
- **Loop**: For each batch path `po/review-batch-<N>.po`: run agent with `source = that path`; parse JSON; save to `po/review-batch-<N>.json`; delete `po/review-batch-<N>.po`. Do **not** merge in memory; do not return merged JSON.
- **Return**: Can return `nil, nil` and rely on step 9 to merge from files; or return a minimal result so that `RunAgentReview` knows to proceed to step 9. Caller (`RunAgentReview`) will then call `ReportReviewFromPathWithBatches` and display.

### 3.4 New helper: prepareReviewBatches

- **Signature**: e.g. `prepareReviewBatches(reviewPOFile string, minBatchSize int) (batchPOPaths []string, entryCount int, err error)`.
- **Implementation**: Remove `po/review-batch-*.po` and `po/review-batch-*.json` in the same directory; count entries; compute batch count and size per AGENTS.md (or current formula); create `po/review-batch-<N>.po` via msg-select (or copy for single batch); return list of paths and entry count.
- **Placement**: `util/agent-run-review.go` or `util/review-prepare.go` (if batch creation is considered “review preparation”).

---

## 4. Constants and Naming

- **Batch PO**: Use pattern `po/review-batch-<N>.po` with N starting from 1. Replace or generalize `ReviewDefaultBatchFile` so that batch files are numbered (e.g. a helper that returns `filepath.Join(dir, fmt.Sprintf("review-batch-%d.po", n))`).
- **Batch JSON**: Use pattern `po/review-batch-<N>.json` so that existing `ReportReviewFromPathWithBatches` (which globs `*-batch-*.json`) picks them up.
- **Output base**: Keep using `outputBase` (e.g. `po/review`) so that paths are `po/review.po`, `po/review.json`, `po/review-batch-1.po`, etc. Derive directory and base name from `outputBase` for batch file names.

---

## 6. Files to Touch

- `util/agent-run-review.go`: Main refactor (RunAgentReview, runReviewSingleBatch, runReviewBatched, new prepareReviewBatches).
- `util/review-types.go`: Optional; add or adjust batch path helpers if needed.
- `cmd/agent-run-report.go`: Optional; extract “print report” into a shared function if we want RunAgentReview to print the same output.
- Tests: Update or add tests for step 1 (existing review / resume), step 3 (batch file creation), step 7 (per-batch JSON write), and step 9 (merge and display).

---

## 7. Order of Implementation (Suggested)

1. Implement `prepareReviewBatches` and batch path naming (step 3).
2. Refactor `runReviewBatched` to write per-batch JSON and delete batch PO (steps 7–8); remove in-memory merge from it.
3. Refactor `runReviewSingleBatch` to accept output path and write `po/review-batch-1.json`.
4. In `RunAgentReview`, add step 1 (check existing review) and wire step 9 (call report logic + display) at the end.
5. In `RunAgentReview`, replace current batch loop with: prepareReviewBatches → loop over batch paths → run agent per batch → save JSON → delete batch PO → step 9.
6. (Optional) Extract report display into a shared function and use it from both `RunAgentReview` and `agent-run report`.
7. Add tests and handle “resume” and “merge only” paths; clarify ambiguities above with manual edits as needed.
