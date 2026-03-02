# Design Document: agent-run translate/review and agent-test translate/review

## Implementation Status

**Status**: ✅ **COMPLETED** (translate functionality)

**Implementation Date**: February 2026

**Completed Steps**:
- ✅ Step 1: Add utility functions (`CountNewEntries`, `CountFuzzyEntries`)
- ✅ Step 2: Implement `agent-run translate` core logic
- ✅ Step 3: Add CLI command for `agent-run translate`
- ✅ Step 4: Implement `agent-test translate` core logic
- ✅ Step 5: Add CLI command for `agent-test translate`
- ✅ Step 6: Error handling and logging
- ✅ Step 7: Documentation and configuration

**Summary**:
All core functionality for `translate` has been implemented, tested, and integrated.
The commands `agent-run translate` and `agent-test translate` are fully functional
and ready for use. Integration tests are included in `test/t0092-agent-run-translate.sh`.

**Note**: The `review` functionality described in this document remains to be
implemented in a future iteration.

---

## 1. Original Requirements

### 1.1 Command Structure

The `agent-run` and `agent-test` subcommands are being extended with two new operations: `translate` and `review`.

**agent-run translate:**
```bash
git-po-helper agent-run translate [--agent <agent-name>] [po/XX.po]
```

This command uses a code agent with a configured prompt to translate new strings and fix fuzzy translations in a PO file (po/XX.po).

**agent-run review:**
```bash
git-po-helper agent-run review [--agent <agent-name>] [--commit <commit>] [--since <commit>] [po/XX.po]
```

This command uses a code agent to review changes in a PO file. It can review:
- Local changes (unstaged or staged)
- Changes in a specific commit (using `--commit`)
- Changes since a specific commit (using `--since`)

**agent-test translate:**
```bash
git-po-helper agent-test translate [--agent <agent-name>] [--runs <n>] [po/XX.po]
```

This command runs the `agent-run translate` operation multiple times (default: 5, configurable via `--runs` or config file) and provides an average score. Similar to `agent-test update-pot`, it reuses the `agent-run translate` logic.

**agent-test review:**
```bash
git-po-helper agent-test review [--agent <agent-name>] [--runs <n>] [--commit <commit>] [--since <commit>] [po/XX.po]
```

This command runs the `agent-run review` operation multiple times and calculates an average score.

### 1.2 Configuration File

The commands read from `git-po-helper.yaml` configuration file. Relevant configuration:

```yaml
default_lang_code: "zh_CN"
prompt:
  update_pot: "update po/git.pot according to po/README.md"
  update_po: "update {{.source}} according to po/README.md"
  translate: "translate {{.source}} according to po/README.md"
  review: "review and improve {{.source}} according to po/README.md"
agent-test:
  runs: 5
  pot_entries_before_update: null
  pot_entries_after_update: null
  po_entries_before_update: null
  po_entries_after_update: null
  po_new_entries_after_update: 0     # Expected new entries after translate (0 = all translated)
  po_fuzzy_entries_after_update: 0   # Expected fuzzy entries after translate (0 = all fixed)
agents:
  claude:
    cmd: ["claude", "-p", "{{.prompt}}"]
  gemini:
    cmd: ["gemini", "--prompt", "{{.prompt}}"]
```

### 1.3 Key Requirements

#### 1.3.1 translate Subcommand

1. **Agent Selection**: If only one agent is configured, `--agent` flag is optional. If multiple agents exist, `--agent` is required.

2. **PO File Location**: If no `po/XX.po` argument is given, the PO file is derived from `default_lang_code` in configuration (e.g., `po/zh_CN.po`).

3. **Prompt Template**: The prompt from `prompt.translate` is used, with placeholders replaced:
   - `{{.prompt}}` → the actual prompt text
   - `{{.source}}` → po file path

4. **Translation Validation**: Before and after calling the agent:
   - Count new entries (untranslated strings with empty msgstr)
   - Count fuzzy entries (marked with `#, fuzzy` flag)
   - **Success criteria**: After translation, both new entries and fuzzy entries must be 0
   - If either count is non-zero after translation, the operation is marked as failed

5. **Testing Mode**: `agent-test translate` runs the operation `--runs` times (default from config) and calculates average success rate.

6. **Output Directory for agent-test**: When using `agent-test translate` with multiple runs, save results to `output/<agent-name>/<iteration-number>/` directory:
   - Generated PO file: `output/<agent-name>/<iteration-number>/XX.po`
   - Execution log: `output/<agent-name>/<iteration-number>/translation.log`
   - If directory exists, overwrite files

#### 1.3.2 review Subcommand

1. **Agent Selection**: Same as translate.

2. **PO File Location**: Same as translate.

3. **Review Modes**:
   - **Local changes** (default): Review unstaged or staged changes in the PO file
   - **Commit mode** (`--commit <commit>`): Review changes in a specific commit
   - **Since mode** (`--since <commit>`): Review changes since a specific commit

4. **Prompt Template**:
   - Use `prompt.review` with `{{.source}}` placeholder

5. **Review Validation**: No automatic validation for review operations. The agent should provide feedback/suggestions, but the operation is considered successful if the agent command completes without error.

6. **Testing Mode**: `agent-test review` runs the operation `--runs` times. Success is based on agent exit code (no validation).

## 2. Detailed Design

### 2.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    git-po-helper                            │
│                                                              │
│  ┌──────────────┐              ┌──────────────┐            │
│  │ agent-run    │              │ agent-test    │            │
│  │              │              │               │            │
│  │ ┌──────────┐ │              │ ┌──────────┐ │            │
│  │ │update-pot│ │              │ │update-pot│ │            │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  │ ┌──────────┐ │              │ ┌──────────┐ │            │
│  │ │update-po │ │              │ │update-po │ │            │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  │ ┌──────────┐ │              │ ┌──────────┐ │            │
│  │ │translate │ │  NEW         │ │translate │ │  NEW      │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  │ ┌──────────┐ │              │ ┌──────────┐ │            │
│  │ │review    │ │  NEW         │ │review    │ │  NEW      │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  └──────────────┘              └──────────────┘            │
│         │                              │                     │
│         └──────────────┬──────────────┘                     │
│                        │                                     │
│         ┌──────────────▼──────────────┐                     │
│         │   util/agent-run.go          │                     │
│         │   - RunAgentTranslate()      │  NEW              │
│         │   - RunAgentReview()         │  NEW              │
│         │   - CmdAgentRunTranslate()   │  NEW              │
│         │   - CmdAgentRunReview()      │  NEW              │
│         └──────────────────────────────┘                     │
│                        │                                     │
│         ┌──────────────▼──────────────┐                     │
│         │   util/agent.go              │                     │
│         │   - CountNewEntries()        │  NEW              │
│         │   - CountFuzzyEntries()      │  NEW              │
│         └──────────────────────────────┘                     │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 New Utility Functions

#### 2.2.1 Count New Entries

```go
func CountNewEntries(poFile string) (int, error)
```

Responsibilities:
- Open the PO file.
- Scan for entries with non-empty `msgid` and empty `msgstr` (untranslated).
- Exclude header entry (first entry with empty msgid).
- Count entries where msgstr is empty or contains only empty strings.
- Return the count of new/untranslated entries.

Error Handling:
- If the file cannot be opened, return an error.
- If the file cannot be read, return an error.

#### 2.2.2 Count Fuzzy Entries

```go
func CountFuzzyEntries(poFile string) (int, error)
```

Responsibilities:
- Open the PO file.
- Scan for entries marked with `#, fuzzy` comment.
- Count entries that have the fuzzy flag.
- Return the count of fuzzy entries.

Error Handling:
- If the file cannot be opened, return an error.
- If the file cannot be read, return an error.

### 2.3 Command Implementation

#### 2.3.1 agent-run translate

**Flow**:
1. Load configuration from `git-po-helper.yaml`
2. Determine agent to use:
   - If `--agent` provided, use that agent
   - If only one agent in config, use it
   - Otherwise, return error requiring `--agent`
3. Determine PO file path:
   - If argument provided, use it
   - Otherwise, derive from `default_lang_code` (e.g., `po/zh_CN.po`)
4. **Pre-validation**:
   - Count new entries in PO file using `CountNewEntries()`
   - Count fuzzy entries in PO file using `CountFuzzyEntries()`
   - Log the counts for reference
5. Get prompt from `prompt.translate`
6. Replace placeholders in agent command:
   - `{{.prompt}}` → prompt text
   - `{{.source}}` → PO file path
7. Execute agent command
8. **Post-validation**:
   - Count new entries in PO file using `CountNewEntries()`
   - Count fuzzy entries in PO file using `CountFuzzyEntries()`
   - **If new entries > 0 OR fuzzy entries > 0**: mark as failed (score = 0)
   - **If both counts are 0**: mark as successful (score = 100)
9. Validate PO file syntax (using `msgfmt`)
10. Return success/failure

**Success Criteria**:
- Agent command exits with code 0
- **New entries count is 0 after translation**
- **Fuzzy entries count is 0 after translation**
- PO file is valid (can be checked with `msgfmt`)

**Error Handling**:
- Configuration file not found: warn but continue with defaults if possible
- Agent not found: return error
- Agent command fails: return error with stderr output
- Invalid PO file: return error
- **Post-validation failure**: Return error with message indicating remaining new/fuzzy entries (score = 0)

#### 2.3.2 agent-run review

**Flow**:
1. Load configuration from `git-po-helper.yaml`
2. Determine agent to use (same as translate)
3. Determine PO file path (same as translate)
4. Determine review mode:
   - If `--commit` provided: review changes in that commit
   - If `--since` provided: review changes since that commit
   - Otherwise: review local changes (using HEAD as reference)
5. Get prompt based on review mode:
   - Use `prompt.review` with `{{.source}}` placeholder
6. Replace placeholders in agent command:
   - `{{.prompt}}` → prompt text
   - `{{.source}}` → PO file path
   - `{commit}` → commit ID (HEAD, specific commit, or since commit)
7. Execute agent command
8. No post-validation required (review is informational)
9. Return success if agent command succeeds

**Success Criteria**:
- Agent command exits with code 0

**Error Handling**:
- Configuration file not found: warn but continue with defaults if possible
- Agent not found: return error
- Agent command fails: return error with stderr output
- Invalid commit reference: return error

#### 2.3.3 agent-test translate

**Flow**:
1. Load configuration
2. Require user confirmation before proceeding (to prevent data loss)
3. Determine number of runs:
   - Use `--runs` if provided
   - Otherwise use `agent-test.runs` from config
   - Default to 5 if neither provided
4. For each run:
   - Clean po/ directory to ensure a clean state
   - **Pre-validation**:
     - Count new entries in PO file
     - Count fuzzy entries in PO file
     - Log counts for reference
   - Run `agent-run translate` logic
   - **Post-validation**:
     - Count new entries in PO file
     - Count fuzzy entries in PO file
     - **If new entries = 0 AND fuzzy entries = 0**: score = 100
     - **Otherwise**: score = 0
   - **Save output to directory**: `output/<agent-name>/<iteration-number>/`
     - Copy translated PO file to `output/<agent-name>/<iteration-number>/XX.po`
     - Save execution log to `output/<agent-name>/<iteration-number>/translation.log`
     - Create directory if it doesn't exist, overwrite files if it exists
   - Record score for this run
5. Calculate average score: `(sum of scores) / number of runs`
6. Display results:
   - Individual run results (including new/fuzzy counts)
   - Average score
   - Summary statistics

**Scoring**:
- Success (new = 0 AND fuzzy = 0) = 100 points
- Failure (new > 0 OR fuzzy > 0) = 0 points
- Average = sum of scores / number of runs

#### 2.3.4 agent-test review

**Flow**:
1. Load configuration
2. Require user confirmation before proceeding
3. Determine number of runs (same as translate)
4. For each run:
   - Clean po/ directory (if reviewing local changes)
   - Run `agent-run review` logic
   - Score based on agent exit code:
     - Success (exit code 0) = 100 points
     - Failure (non-zero exit code) = 0 points
   - Record score for this run
5. Calculate average score
6. Display results

**Scoring**:
- Success = 100 points
- Failure = 0 points
- Average = sum of scores / number of runs

### 2.4 Output Directory Management (agent-test translate)

For `agent-test translate`, results are saved to preserve translation quality for later review:

**Directory Structure**:
```
output/
├── <agent-name>/
│   ├── 1/
│   │   ├── XX.po
│   │   └── translation.log
│   ├── 2/
│   │   ├── XX.po
│   │   └── translation.log
│   └── ...
```

**Implementation**:
1. After each run, determine output directory:
   - Path: `output/<agent-name>/<run-number>/`
   - Example: `output/claude/1/`, `output/claude/2/`, etc.
2. Create directory if it doesn't exist (using `os.MkdirAll`)
3. Copy translated PO file:
   - Source: `po/XX.po` (in working directory)
   - Destination: `output/<agent-name>/<run-number>/XX.po`
4. Save execution log:
   - Capture agent stdout/stderr
   - Save to `output/<agent-name>/<run-number>/translation.log`
5. If directory already exists, overwrite files

**Benefits**:
- Preserves translation results from each run
- Allows quality comparison across runs
- Enables manual review of agent translations
- Prevents data loss from iterative runs

## 3. Implementation Steps

### Step 1: Add New Utility Functions

**Tasks**:
1. Implement `CountNewEntries()` in `util/agent.go`
2. Implement `CountFuzzyEntries()` in `util/agent.go`
3. Add unit tests for both functions

**Files to Modify**:
- `util/agent.go` - Add new counting functions
- `util/agent_test.go` - Add tests for new functions

**Validation**:
- Unit tests pass
- Functions correctly count new and fuzzy entries in various PO files

### Step 2: Implement agent-run translate

**Tasks**:
1. Implement `RunAgentTranslate()` in `util/agent-run.go`
2. Implement `CmdAgentRunTranslate()` in `util/agent-run.go`
3. Integrate pre-validation (count new/fuzzy entries)
4. Integrate post-validation (verify new=0 and fuzzy=0)
5. Add PO file syntax validation

**Files to Modify**:
- `util/agent-run.go` - Add translate logic

**Functions to Implement**:
```go
func RunAgentTranslate(cfg *config.AgentConfig, agentName, poFile string) (*AgentRunResult, error)
func CmdAgentRunTranslate(agentName, poFile string) error
```

**Validation**:
- Integration test with mock agent
- Test with PO files containing new/fuzzy entries
- Test validation logic (success when new=0 and fuzzy=0)

### Step 3: Add CLI Commands (agent-run)

**Tasks**:
1. Add `translate` subcommand to `cmd/agent-run.go`
2. Add `review` subcommand to `cmd/agent-run.go`
3. Add appropriate flags and help text

**Files to Modify**:
- `cmd/agent-run.go` - Add new subcommands

**Validation**:
- `git-po-helper agent-run translate -h` shows help
- Flags are parsed correctly

### Step 4: Implement agent-test translate

**Tasks**:
1. Implement `RunAgentTestTranslate()` in `util/agent-test.go`
2. Implement `CmdAgentTestTranslate()` in `util/agent-test.go`
3. Add output directory management:
   - Create `output/<agent-name>/<run-number>/` directory
   - Copy translated PO file to output directory
   - Save execution log to output directory
4. Implement validation logic (check new=0 and fuzzy=0)
5. Display results with new/fuzzy counts

**Files to Modify**:
- `util/agent-test.go` - Add translate test logic

**Functions to Implement**:
```go
func RunAgentTestTranslate(agentName, poFile string, runs int, cfg *config.AgentConfig) ([]RunResult, float64, error)
func CmdAgentTestTranslate(agentName, poFile string, runs int, skipConfirmation bool) error
func SaveTranslateResults(agentName string, runNumber int, poFile string, stdout, stderr []byte) error
```

**Validation**:
- Integration test with multiple runs
- Test output directory creation and file copying
- Test validation logic
- Verify results are saved correctly

### Step 5: Add CLI Commands (agent-test)

**Tasks**:
1. Add `translate` subcommand to `cmd/agent-test.go`
2. Add appropriate flags (--runs) and help text

**Files to Modify**:
- `cmd/agent-test.go` - Add new subcommands

**Validation**:
- `git-po-helper agent-test translate -h` shows help
- Flags are parsed correctly

### Step 6: Error Handling and Logging

**Tasks**:
1. Add appropriate log messages
2. Handle all error cases gracefully
3. Provide helpful error messages
4. Use existing logging patterns (logrus)

**Validation**:
- Error messages are clear and actionable
- Logging follows project conventions

### Step 7: Documentation and Testing

**Tasks**:
1. Update this design document
2. Write integration tests for translate
3. Test with real agents (if available)
4. Update example configuration file

**Files to Create/Modify**:
- `docs/design/agent-run-translate.md` - This document
- `test/t*agent-translate*.sh` - Integration tests for translate
- `docs/git-po-helper.yaml.example` - Update example config

**Validation**:
- Documentation is clear and complete
- Integration tests pass
- Example config works

## 4. Testing Strategy

### 4.1 Unit Tests

- `CountNewEntries()` with various PO files
- `CountFuzzyEntries()` with various PO files
- Translation validation logic

### 4.2 Integration Tests

- Full `agent-run translate` workflow with mock agent
- Full `agent-test translate` workflow with multiple runs
- Output directory management (file creation, overwriting)
- Error scenarios (missing config, invalid agent, etc.)

### 4.3 Manual Testing

- Test with real agent commands (if available)
- Verify translations work correctly
- Verify output directory structure is correct

## 5. Future Considerations

- Parallel execution for agent-test
- Integration with CI/CD pipelines
- Support for partial translations (allow some fuzzy entries)
- Translation quality metrics
