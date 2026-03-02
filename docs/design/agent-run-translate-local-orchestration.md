# Design Document: agent-run translate --use-agent-md and --use-local-orchestration

## 1. Overview

This document describes the design for extending `agent-run translate` with two mutually exclusive modes:

1. **--use-agent-md**: Use the existing translation flow. The agent receives the full PO file (or extracted untranslated/fuzzy entries) and performs translation in one or more rounds. This is the current behavior.

2. **--use-local-orchestration**: Use the new translation flow from [po/AGENTS.md Task 4](https://github.com/git-l10n/git-po/blob/master/po/AGENTS.md). git-po-helper orchestrates the workflow locally using `msg-select` and `msg-cat`; the agent is invoked **only** for translating each batch JSON file. Like `agent-run review`, this mode uses a **separate prompt** maintained in `config/prompts/local-orchestration-translation.md` (distinct from `config/prompts/translate.txt` used by `--use-agent-md`).

## 2. Reference: AGENTS.md Task 3 Translation Flow (One Batch Per Iteration)

The flow uses **one batch per iteration** (AGENTS.md Task 3):

1. **Extract pending**: `msg-select --untranslated --fuzzy --no-obsolete -o pending po/XX.po`
2. **Prepare one batch**: `msg-select --json [--range "-$NUM"] pending -o l10n-todo.json` (first N entries)
3. **Translate**: Agent reads `l10n-todo.json`, writes `l10n-done.json`
4. **Merge**: `msg-cat --unset-fuzzy -o l10n-done.po l10n-done.json`, then `msgcat --use-first l10n-done.po po/XX.po -o merged`
5. **Replace target**: `mv merged po/XX.po`
6. **Repeat** until pending is empty

Batch size formula (from AGENTS.md):
- If `ENTRY_COUNT <= min_batch_size*2`: single batch (all entries)
- If `ENTRY_COUNT > min_batch_size*8`: NUM = min_batch_size*2
- Else if `ENTRY_COUNT > min_batch_size*4`: NUM = min_batch_size + min_batch_size/2
- Else: NUM = min_batch_size

`msg-cat --unset-fuzzy` is applied before merging so that fuzzy entries translated by the agent become normal entries (equivalent to `msg-cat --unset-fuzzy`).

## 3. Command Interface

### 3.1 Flags

```bash
git-po-helper agent-run translate [--use-agent-md | --use-local-orchestration] [--agent <name>] [--batch-size <n>] [po/XX.po]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--use-agent-md` | false | Use existing flow: agent receives full/extracted PO, does translation |
| `--use-local-orchestration` | false | Use new flow: local orchestration, agent only translates batch JSON files |
| `--agent` | (from config) | Agent name when multiple agents configured |
| `--batch-size` | 50 | Min entries per batch (local-orchestration only) |

**Mutual exclusivity**: If both `--use-agent-md` and `--use-local-orchestration` are specified, return error. If neither is specified, default to `--use-agent-md` (preserve current behavior).

### 3.2 Examples

```bash
# Existing flow (default)
git-po-helper agent-run translate po/zh_CN.po

# Explicit existing flow
git-po-helper agent-run translate --use-agent-md po/zh_CN.po

# New flow: local orchestration, agent only for batch translation
git-po-helper agent-run translate --use-local-orchestration po/zh_CN.po

# With batch size
git-po-helper agent-run translate --use-local-orchestration --batch-size 30 po/zh_CN.po
```

## 4. Architecture

### 4.1 Flow Comparison

```
--use-agent-md (existing):
  [git-po-helper] → pre-validation → [Agent: full PO] → post-validation → done

--use-local-orchestration (new):
  [git-po-helper] → msg-select (todo.po) → msg-select (todo.json, one batch) →
  [Agent: todo.json → done.json] →
  [git-po-helper] → msg-cat --unset-fuzzy (done.po) → msgcat (merge into XX.po) → msgfmt (validate) →
  repeat until pending empty → done
```

### 4.2 Reference: agent-run review Implementation

`agent-run review` provides the blueprint:

- **RunAgentReview** (local orchestration): Uses `config/prompts/review.txt`; `PrepareReviewData` → `prepareReviewBatches` (msg-select PO) → `runReviewBatched` (agent per batch) → `ReportReviewFromPathWithBatches` (merge JSONs)
- **RunAgentReviewUseAgentMd**: Single agent call with dynamically built prompt; agent does extraction, batching, review, and writes `review.json`

For translate local orchestration:
- Use **separate prompt** `config/prompts/local-orchestration-translation.md` (like review uses review.txt)
- Use **JSON batches** (not PO batches) as in AGENTS.md
- One batch per iteration: Agent receives `l10n-todo.json`, writes `l10n-done.json`
- Placeholders: `{{.source}}` = input JSON path, `{{.dest}}` = output JSON path

## 5. Detailed Design

### 5.1 File Naming Convention

For `po/XX.po` (e.g. `po/zh_CN.po`), use base `po/l10n` (one batch per iteration):

| File | Purpose |
|------|---------|
| `po/l10n-todo.po` | Extracted untranslated + fuzzy entries |
| `po/l10n-todo.json` | Current batch to translate (input to agent) |
| `po/l10n-done.json` | Current batch translated (output from agent) |
| `po/l10n-done.po` | Converted from l10n-done.json (with --unset-fuzzy) for merge |

### 5.2 RunAgentTranslateUseAgentMd (existing flow)

- Rename or alias current `RunAgentTranslate` logic as the "use-agent-md" path
- No structural change; just ensure it is invoked when `--use-agent-md` is set or when both flags are absent (default)

### 5.3 RunAgentTranslateLocalOrchestration (new flow)

Implement steps matching AGENTS.md Task 3 (one batch per iteration):

**Step 1: Condition check**

- If `po/l10n-todo.po` does not exist → go to Step 2
- If `po/l10n-todo.json` exists → go to Step 4 (translate)
- If `po/l10n-done.json` exists (and no todo.json) → go to Step 5 (merge)
- Otherwise → go to Step 3 (generate one batch)

**Step 2: Generate pending file**

```go
// Remove any stale batch files
os.Remove("po/l10n-todo.json")
os.Remove("po/l10n-done.json")
// msg-select --untranslated --fuzzy --no-obsolete po/XX.po -o po/l10n-todo.po
MsgSelect(poFile, "", todoFile, false, &EntryStateFilter{Untranslated: true, Fuzzy: true, NoObsolete: true})
```

If `l10n-todo.po` is empty or has no content entries → translation complete; cleanup and return success.

**Step 3: Generate one batch**

- Count entries in `l10n-todo.po` (excluding header)
- Apply batch size formula to get `num` per batch
- `msg-select --json --range "-$num" po/l10n-todo.po -o po/l10n-todo.json` (first N entries)
- Use `WriteGettextJSONFromPOFile` or equivalent

**Step 4: Translate batch**

- Build agent command with placeholders:
  - `{{.prompt}}`: from `prompt.local_orchestration_translation`
  - `{{.source}}`: `po/l10n-todo.json`
  - `{{.dest}}`: `po/l10n-done.json`
- Execute agent
- Agent writes translated JSON to `{{.dest}}`
- Validate output JSON exists and is parseable

**Step 5: Merge and complete**

```go
// Apply --unset-fuzzy to done JSON, convert to PO
ClearFuzzyTagFromGettextJSON(doneJSON) → write to l10n-done.po
// msgcat --use-first l10n-done.po po/XX.po -o merged
exec.Command("msgcat", "--use-first", "po/l10n-done.po", poFile).Output() → merged
// msgfmt --check -o /dev/null merged
exec.Command("msgfmt", "--check", "-o", "/dev/null", "merged")
// mv merged po/XX.po
os.Rename("merged", poFile)
// Cleanup for next iteration
os.Remove("po/l10n-done.po")
os.Remove("po/l10n-done.json")
os.Remove("po/l10n-todo.json")
os.Remove("po/l10n-todo.po")
```

Loop back to Step 1; Step 2 will regenerate `l10n-todo.po` from updated `po/XX.po`. If empty, translation is complete.

### 5.4 Prompt and Placeholders for Local Orchestration

**Separate prompt file**: Like `agent-run review`, when using programmatic/local orchestration, git-po-helper uses a **separate prompt** maintained in `config/prompts/local-orchestration-translation.md`. This is distinct from `config/prompts/translate.txt`, which is used only for the `--use-agent-md` flow.

The local-orchestration prompt should instruct the agent to:
- Read the gettext JSON from `{{.source}}`
- Translate each entry (msgid → msgstr, msgstr_plural for plurals)
- Write the translated gettext JSON to `{{.dest}}`

Config key: `prompt.local_orchestration_translation` (embedded from `config/prompts/local-orchestration-translation.md`).

Placeholders:
- `{{.prompt}}`: resolved prompt content
- `{{.source}}`: `po/l10n-todo.json`
- `{{.dest}}`: `po/l10n-done.json`

Config example (git-po-helper.yaml):

```yaml
prompt:
  translate: "Translate file {{.source}} according to @po/AGENTS.md."
  local_orchestration_translation: "..."  # loaded from config/prompts/local-orchestration-translation.md
```

The `local_orchestration_translation` prompt is loaded from the embedded file; users may override it in their config.

### 5.5 Resume Support

- If `l10n-todo.json` exists: resume from Step 4 (translate)
- If only `l10n-done.json` exists: run Step 5 (merge) then loop

### 5.6 Agent Output Handling

For local orchestration, the agent writes directly to `{{.dest}}`. git-po-helper does **not** parse stdout for JSON; it only checks that the output file exists and is valid gettext JSON. If the agent uses streaming JSON output, we may need a variant that writes to file (similar to review's `output` config).

## 6. Implementation Plan

### 6.1 Files to Modify/Create

| File | Changes |
|------|---------|
| `cmd/agent-run-translate.go` | Add `--use-agent-md`, `--use-local-orchestration`, `--batch-size`; dispatch to appropriate flow |
| `util/agent-run-translate.go` | Add `RunAgentTranslateLocalOrchestration`; refactor `RunAgentTranslate` as use-agent-md path |
| `util/agent-run-translate-local.go` | **New**: Local orchestration logic (steps 1–6, batch creation, merge, msgcat/msgfmt) |
| `config/prompts/local-orchestration-translation.md` | **New**: Separate prompt for local orchestration batch translation |
| `config/agent.go` | Add `local_orchestration_translation` prompt key and embed |

### 6.2 Development Steps (each step = one commit)

| Step | Commit | Description |
|------|--------|-------------|
| 1 | Add flags to translate command | Add `--use-agent-md`, `--use-local-orchestration`, `--batch-size` to `cmd/agent-run-translate.go`; implement mutual exclusivity and default to `--use-agent-md` |
| 2 | Add local-orchestration prompt | Create `config/prompts/local-orchestration-translation.md` with batch translation instructions; add `local_orchestration_translation` to `config/agent.go` (PromptConfig, embed, default); wire `GetRawPrompt` for action `local-orchestration-translation` |
| 3 | Create local orchestration module | Create `util/agent-run-translate-local.go` with `RunAgentTranslateLocalOrchestration`; implement Steps 1–5 (condition check, msg-select todo, one-batch JSON, agent translate, merge with --unset-fuzzy, msgcat+msgfmt); one batch per iteration |
| 4 | Wire translate command to local flow | In `cmd/agent-run-translate.go`, when `--use-local-orchestration` is set, call `RunAgentTranslateLocalOrchestration`; ensure `--use-agent-md` (or default) calls existing `RunAgentTranslate` |
| 5 | Add integration tests | Add integration test for `agent-run translate --use-local-orchestration` (e.g. in `test/t0090-agent-run.sh` or new test file); verify batch flow, merge, and final PO |
| 6 | Update documentation | Update README, `docs/agent-commands.md` (or equivalent) with `--use-local-orchestration` usage and `config/prompts/local-orchestration-translation.md` reference |

### 6.3 Dependencies

- `util.MsgSelect` with `EntryStateFilter{Untranslated: true, Fuzzy: true, NoObsolete: true}` for Step 2
- `util.WriteGettextJSONFromPOFile` for batch JSON output (Step 3)
- `util.ClearFuzzyTagFromGettextJSON` before converting done JSON to PO (equivalent to `msg-cat --unset-fuzzy`)
- `util.WriteGettextJSONToPO` for done JSON → PO conversion
- External: `msgcat`, `msgfmt` (already required by project)
- `BuildAgentCommand` with `source` and `dest` placeholders

## 7. Testing

- Unit tests: batch creation logic, batch size formula, file naming
- Integration test: run `agent-run translate --use-local-orchestration` with a test agent (e.g. echo/copy) that copies JSON; verify merge and final PO
- Ensure `--use-agent-md` preserves existing behavior (no regression)

## 8. Open Questions

1. **Agent output**: If agent uses streaming JSON, should we support writing to file via config (like review)?
2. **Batch size default**: 50 (per AGENTS.md) or configurable only via `--batch-size`?
