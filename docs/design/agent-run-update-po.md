# Design Document: agent-run update-po and agent-test update-po

## 1. Original Requirements

### 1.1 Command Structure

The `agent-run` and `agent-test` subcommands integrate code agents (such as Claude, Gemini, etc.) into the git-po-helper workflow for automating localization tasks.

**agent-run update-po:**
```bash
git-po-helper agent-run update-po [--agent <agent-name>] [po/XX.po]
```

This command uses a code agent with a configured prompt to update the `po/git.pot` template file and, based on that updated template, update a specific `po/XX.po` file (where `XX` is the language code). The actual update logic is implemented by the external agent according to the prompt and project documentation (e.g., `po/README.md`).

**agent-test update-po:**
```bash
git-po-helper agent-test update-po [--agent <agent-name>] [--runs <n>] [po/XX.po]
```

This command runs the `agent-run update-po` operation multiple times (default: from config, or 5 if not configured) and provides an average score where success = 100 points and failure = 0 points.

The `po/XX.po` argument is optional. If omitted, the default language code from configuration is used to determine the PO file path (e.g., `default_lang_code: "zh_CN"` -> `po/zh_CN.po`).

### 1.2 Configuration File

Both commands read from the `git-po-helper.yaml` configuration file. Example:

```yaml
default_lang_code: "zh_CN"
prompt:
  update_pot: "update po/git.pot according to po/README.md"
  update_po: "update {source} according to po/README.md"
  translate: "translate {source} according to po/README.md"
  review: "review and improve {source} according to po/README.md"
agent-test:
  runs: 5
  pot_entries_before_update: null
  pot_entries_after_update: null
  po_entries_before_update: null
  po_entries_after_update: null
  po_new_entries_after_update: null
  po_fuzzy_entries_after_update: null
agents:
  claude:
    cmd: ["claude", "-p", "{prompt}"]
  gemini:
    cmd: ["gemini", "--prompt", "{prompt}"]
```

### 1.3 Key Requirements

1. **Agent Selection**: If only one agent is configured, the `--agent` flag is optional. If multiple agents exist, `--agent` is required.

2. **Prompt Template**: The prompt from `prompt.update_po` is used, with placeholders replaced:
   - `{prompt}` → the actual prompt text (from configuration)
   - `{source}` → the PO file path, e.g., `po/zh_CN.po`
   - `{commit}` → commit ID (not used directly by `update-po`)

3. **Command Execution**: The agent command from `agents.<agent-name>.cmd` is executed with placeholders replaced. The command runs in the repository root directory.

4. **Testing Mode**: `agent-test update-po` runs the `agent-run update-po` operation `--runs` times (default from config) and calculates the average success rate.

5. **PO Entry Count Validation**: If `po_entries_before_update` and/or `po_entries_after_update` are configured (not null and not 0), the system will:
   - For `agent-run update-po`:
     - **Before** calling the agent, count entries in the target `po/XX.po` file and compare with `po_entries_before_update` (if configured).
     - **After** calling the agent, count entries in the same `po/XX.po` file and compare with `po_entries_after_update` (if configured).
     - If any enabled validation fails, the operation is marked as failed (score = 0) and an error is reported.
   - For `agent-test update-po`:
     - Perform the same checks for each run and compute scores per run (100 for success, 0 for failure).
   - If these values are not configured (null or 0), no PO entry-count validation is performed.

6. **Success Criteria**:
   - The agent command exits with code 0.
   - The target `po/XX.po` file exists and is syntactically valid (checked with `msgfmt`).
   - **Entry-count validations (if configured)**:
     - Entry count before update matches `po_entries_before_update`.
     - Entry count after update matches `po_entries_after_update`.
   - If validation is not configured (null or 0), validation steps are skipped and scoring is based on agent command exit code.

7. **Scoring**:
   - Success = 100 points
   - Failure = 0 points
   - Average score = (sum of scores) / number of runs

> Note: Requirements for `translate` (e.g., clearing new/fuzzy entries) and `review` will be covered in separate design documents. This document focuses on `agent-run update-po` and `agent-test update-po`, especially the PO entry-count validation.

## 2. Detailed Design

### 2.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    git-po-helper                            │
│                                                             │
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
│  │ │translate │ │              │ │translate │ │            │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  │ ┌──────────┐ │              │ ┌──────────┐ │            │
│  │ │review    │ │              │ │review    │ │            │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  └──────────────┘              └──────────────┘            │
│         │                              │                   │
│         └──────────────┬──────────────┘                   │
│                        │                                   │
│         ┌──────────────▼──────────────┐                   │
│         │   util/agent.go             │                   │
│         │   - LoadAgentConfig()       │                   │
│         │   - SelectAgent()           │                   │
│         │   - BuildAgentCommand()     │                   │
│         │   - ExecuteAgentCommand()   │                   │
│         │   - CountPotEntries()       │                   │
│         │   - CountPoEntries()        │                   │
│         └──────────────────────────────┘                   │
│                        │                                   │
│         ┌──────────────▼──────────────┐                   │
│         │   util/agent-run.go         │                   │
│         │   - RunAgentUpdatePot()     │                   │
│         │   - RunAgentUpdatePo()      │                   │
│         │   - ValidatePotEntryCount() │                   │
│         │   - ValidatePoEntryCount()  │                   │
│         │   - ValidatePoFile()        │                   │
│         └──────────────────────────────┘                   │
│                        │                                   │
│         ┌──────────────▼──────────────┐                   │
│         │   util/agent-test.go        │                   │
│         │   - RunAgentTestUpdatePot() │                   │
│         │   - RunAgentTestUpdatePo()  │                   │
│         │   - CleanPoDirectory()      │                   │
│         │   - displayTestResults()    │                   │
│         └──────────────────────────────┘                   │
│                        │                                   │
│         ┌──────────────▼──────────────┐                   │
│         │   config/agent.go           │                   │
│         │   - AgentConfig struct      │                   │
│         │   - LoadAgentConfig()       │                   │
│         └──────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Configuration Management

**File Location**: `git-po-helper.yaml` can be located in:
- User home directory: `~/.git-po-helper.yaml` (lower priority)
- Repository root: `<repo-root>/git-po-helper.yaml` (higher priority)

Both files are loaded and merged by `config.LoadAgentConfig()`. The repo-level configuration overrides the user-level configuration.

**Configuration Structure** (as implemented in `config/agent.go`):

```go
type AgentConfig struct {
    DefaultLangCode string           `yaml:"default_lang_code"`
    Prompt          PromptConfig     `yaml:"prompt"`
    AgentTest       AgentTestConfig  `yaml:"agent-test"`
    Agents          map[string]Agent `yaml:"agents"`
}

type PromptConfig struct {
    UpdatePot    string `yaml:"update_pot"`
    UpdatePo     string `yaml:"update_po"`
    Translate    string `yaml:"translate"`
    Review string `yaml:"review"`
}

type AgentTestConfig struct {
    Runs                      *int `yaml:"runs"`
    PotEntriesBeforeUpdate    *int `yaml:"pot_entries_before_update"`
    PotEntriesAfterUpdate     *int `yaml:"pot_entries_after_update"`
    PoEntriesBeforeUpdate     *int `yaml:"po_entries_before_update"`
    PoEntriesAfterUpdate      *int `yaml:"po_entries_after_update"`
    PoNewEntriesAfterUpdate   *int `yaml:"po_new_entries_after_update"`
    PoFuzzyEntriesAfterUpdate *int `yaml:"po_fuzzy_entries_after_update"`
}

type Agent struct {
    Cmd []string `yaml:"cmd"`
}
```

**Configuration Loading**:
- `LoadAgentConfig()` loads and merges user-level and repo-level config.
- Defaults (including prompts and test runs) are applied if fields are missing.
- Validation ensures that at least one agent is configured and that `prompt.update_pot` exists.
- For `update-po`, `prompt.update_po` is expected to be set; if missing, a clear error message is returned.

### 2.3 PO Entry Counting

To support entry-count validation on PO files, we introduce a PO-specific counting function, analogous to `CountPotEntries`.

#### 2.3.1 Count PO Entries

```go
func CountPoEntries(poFile string) (int, error)
```

Responsibilities:
- Open the PO file.
- Scan for `msgid` entries (excluding header and commented/obsolete entries).
- Treat the first empty `msgid ""` block as header and do not count it.
- For each non-empty `msgid` (including multi-line msgid), increment the counter.
- Return the total number of non-header entries.

Error Handling:
- If the file cannot be opened, return an error.
- If the file cannot be read, return an error.

This function mirrors the behavior of `CountPotEntries` but is tailored for PO files. In many cases, POT and PO formats are similar enough that the same logic can be reused or shared.

### 2.4 PO Entry Count Validation Logic

PO entry-count validation is a critical feature for ensuring the agent updates the PO file correctly when running `update-po`. The behavior is controlled by `po_entries_before_update` and `po_entries_after_update` in configuration.

#### 2.4.1 Validation Rules

1. **Null or Zero Values**: If a validation field is `null` or `0`, validation is **disabled** for that stage. The system will not perform any entry-count checking for that stage.

2. **Non-Zero Values**: If a validation field has a non-zero value, validation is **enabled** and the system will:
   - Count entries in the target `po/XX.po` file at the specified stage using `CountPoEntries()`.
   - Compare the actual count with the expected value from configuration.
   - Mark the operation as failed (score = 0) if counts do not match.
   - Mark the operation as successful (score = 100) if counts match (for that stage).

#### 2.4.2 ValidatePoEntryCount Helper

```go
func ValidatePoEntryCount(poFile string, expectedCount *int, stage string) error
```

Behavior:
- If `expectedCount` is `nil` or `*expectedCount == 0`, return `nil` immediately (validation disabled).
- If the file does not exist:
  - For `stage == "before update"`, treat the entry count as `0` and compare against the expected value.
  - For `stage == "after update"`, treat missing file as an error, since the agent is expected to have created/updated it.
- If the file exists, use `CountPoEntries()` to obtain the actual count.
- If `actualCount != *expectedCount`, return an error with a message including stage, expected, actual, and file path.
- If counts match, log a debug message and return `nil`.

#### 2.4.3 Pre-Validation (Before Agent Execution)

**When**: `po_entries_before_update` is configured (not null and not 0).

**Process**:
1. Determine the target PO file path (from CLI argument or default language code).
2. Call `CountPoEntries()` to get the initial entry count (or treat as 0 if file is missing).
3. Compare with `po_entries_before_update` using `ValidatePoEntryCount()`.
4. **If mismatch**:
   - Log an error message: `"entry count before update: expected {expected}, got {actual}"`.
   - Return an error immediately.
   - **Do not execute the agent command**.
   - Score = 0.
5. **If match**:
   - Continue to agent execution.

Use Case: Ensures the PO file is in the expected state before the agent runs, which is particularly useful for repeatable tests in `agent-test`.

#### 2.4.4 Post-Validation (After Agent Execution)

**When**: `po_entries_after_update` is configured (not null and not 0).

**Process**:
1. Execute the agent command (if pre-validation passed or was disabled).
2. Count entries in the `po/XX.po` file using `CountPoEntries()`.
3. Compare with `po_entries_after_update` using `ValidatePoEntryCount()`.
4. **If mismatch**:
   - Log an error message: `"entry count after update: expected {expected}, got {actual}"`.
   - Return an error.
   - Score = 0.
5. **If match**:
   - Mark operation as successful.
   - Score = 100.

Use Case: Verifies that the agent correctly updated the PO file to contain the expected number of entries.

#### 2.4.5 Validation in agent-test Mode

In `agent-test` mode, validation is performed for each run:

- **Pre-validation failure**: The run is marked as failed (score = 0) and the agent is not executed for that run.
- **Post-validation failure**: The run is marked as failed (score = 0) even if the agent command succeeded.
- **Both validations pass**: The run is marked as successful (score = 100).
- **Validation disabled**: Scoring is based solely on the agent command exit code (0 = 100, non-zero = 0).

The average score is calculated as: `(sum of all run scores) / number of runs`.

#### 2.4.6 Example Scenarios

**Scenario 1: Both validations enabled**

```yaml
agent-test:
  po_entries_before_update: 5000
  po_entries_after_update: 5100
```

- Before agent: Verify 5000 entries (fail if not).
- After agent: Verify 5100 entries (fail if not).
- Success only if both match.

**Scenario 2: Only post-validation enabled**

```yaml
agent-test:
  po_entries_before_update: null
  po_entries_after_update: 5100
```

- Before agent: No validation.
- After agent: Verify 5100 entries (fail if not).

**Scenario 3: Validation disabled**

```yaml
agent-test:
  po_entries_before_update: null
  po_entries_after_update: null
```

- No entry-count validation.
- Scoring based on agent exit code only.

### 2.5 Command Implementation: agent-run update-po

#### 2.5.1 Flow

1. **Load configuration** from `git-po-helper.yaml` using `config.LoadAgentConfig()`.
2. **Determine target PO file**:
   - If `po/XX.po` is provided on the command line, use it.
   - Otherwise, derive the PO file from `cfg.DefaultLangCode`, e.g., `po/<lang>.po`.
3. **Select agent** using `SelectAgent(cfg, agentName)`:
   - If `--agent` is provided, use that agent.
   - If exactly one agent is configured, use it automatically.
   - Otherwise, return an error requiring `--agent`.
4. **Pre-validation** (if `po_entries_before_update` is configured and non-zero):
   - Count entries in the target PO file using `CountPoEntries()`.
   - Compare with `po_entries_before_update` using `ValidatePoEntryCount()`.
   - If mismatch, return an error and exit (score = 0).
5. **Get prompt** from `cfg.Prompt.UpdatePo`:
   - If empty, return an error instructing the user to set `prompt.update_po`.
6. **Build agent command**:
   - Use `BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt, "source": source})`, where `source` is the PO file path (e.g., `po/zh_CN.po`).
   - Placeholders `{prompt}`, `{source}`, and `{commit}` are replaced before execution.
7. **Execute agent command**:
   - Use `ExecuteAgentCommand(agentCmd, repository.WorkDir())`.
   - Capture stdout and stderr for debugging and error reporting.
8. **Post-validation** (if `po_entries_after_update` is configured and non-zero):
   - Count entries in the target PO file after update using `CountPoEntries()`.
   - Compare with `po_entries_after_update` using `ValidatePoEntryCount()`.
   - If mismatch, return an error and mark the operation as failed (score = 0).
   - If match, mark the operation as successful (score = 100).
   - If `po_entries_after_update` is not configured, score based on agent exit code (success = 100, failure = 0).
9. **Validate PO file syntax**:
   - Use `ValidatePoFile(poFile)` (which uses `msgfmt` for `.po` files).
   - If validation fails, return an error with a clear hint.
10. **Return success/failure** based on the combined results of agent execution, entry-count validation, and syntax validation.

#### 2.5.2 Success Criteria

- Configuration was loaded successfully.
- Agent was selected successfully.
- If enabled, pre-validation succeeded.
- Agent command executed with exit code 0.
- If enabled, post-validation succeeded.
- PO file syntax validation passed.

#### 2.5.3 Error Handling

- **Configuration file not found**: Return an error with a hint to create `git-po-helper.yaml` in the repository root or user home directory.
- **Agent not found**: Return an error listing available agents and suggesting `--agent`.
- **Prompt missing** (`prompt.update_po` empty): Return a clear error instructing the user to configure it.
- **Pre-validation failure**: Return an error indicating expected vs actual PO entry counts before update, along with a hint.
- **Agent command failure**: Return an error including exit code and stderr output.
- **Post-validation failure**: Return an error indicating expected vs actual PO entry counts after update.
- **PO file syntax invalid**: Return an error including `msgfmt` output and a hint on how to run `msgfmt` manually.

### 2.6 Command Implementation: agent-test update-po

#### 2.6.1 Flow

1. **User confirmation** (optional):
   - Reuse `ConfirmAgentTestExecution(skipConfirmation)` to warn the user and require explicit confirmation before running multiple updates that might modify files under `po/`.
2. **Load configuration** using `config.LoadAgentConfig()`.
3. **Determine number of runs**:
   - If `--runs` is provided and > 0, use that value.
   - Otherwise, use `cfg.AgentTest.Runs` if configured and > 0.
   - If neither is set, default to 5.
4. **Loop over runs** (`i = 1..runs`):
   - Optionally **clean PO directory** using `CleanPoDirectory()` to restore `po/` to `HEAD` before each run (for reproducible tests).
   - Call `RunAgentUpdatePo(cfg, agentName, poFile)` (the PO file is derived the same way as in agent-run).
   - Convert its result into a `RunResult` structure, recording:
     - Run number.
     - Score (0 or 100).
     - Pre/post validation flags.
     - Agent execution/success flags.
     - Errors (pre/post validation, agent error).
     - Entry counts before/after.
     - Expected before/after values (from configuration).
5. **Accumulate scores** and compute the average: `averageScore = totalScore / runs`.
6. **Display results** using a function similar to `displayTestResults`:
   - For each run: show `PASS`/`FAIL`, score, pre/post validation status, and entry counts.
   - At the end: show total runs, successful/failed runs, number of pre/post validation failures, and average score.

#### 2.6.2 Scoring

- **With validation enabled (po_entries_before_update/po_entries_after_update)**:
  - If any enabled validation fails, the run score is 0.
  - If all enabled validations pass and agent execution succeeds, the run score is 100.
- **With validation disabled**:
  - If the agent command exits with code 0 and PO syntax is valid, the run score is 100.
  - Otherwise, the run score is 0.

#### 2.6.3 Reuse of Existing Logic

- Reuse `ConfirmAgentTestExecution` and `CleanPoDirectory` from `util/agent-test.go`.
- Reuse `displayTestResults` by extending `RunResult` to include PO-specific fields, or create a similar function if separation is preferred.
- Reuse `RunAgentUpdatePo` for the core logic of each run, so that `agent-run` and `agent-test` share behavior.

### 2.7 Integration Points

**Existing Code Reuse**:
- `util/agent.go`:
  - `SelectAgent()` – Agent selection.
  - `BuildAgentCommand()` – Placeholder replacement for `{prompt}`, `{source}`, `{commit}`.
  - `ExecuteAgentCommand()` – Command execution and logging.
- `util/agent-run.go`:
  - `RunAgentUpdatePot()` and `ValidatePotEntryCount()` as patterns for implementing PO variants.
  - `ValidatePoFile()` for syntax checking.
- `util/agent-test.go`:
  - `ConfirmAgentTestExecution()` and `CleanPoDirectory()` for safe testing.
  - `displayTestResults()` for reporting.
- `config/agent.go`:
  - `LoadAgentConfig()` and the configuration structures.

**New/Updated Functions** (conceptual):

```go
// Count entries in a PO file.
func CountPoEntries(poFile string) (int, error)

// Validate PO entry count according to configuration.
func ValidatePoEntryCount(poFile string, expectedCount *int, stage string) error

// Run a single agent-run update-po operation.
func RunAgentUpdatePo(cfg *config.AgentConfig, agentName, poFile string) (*AgentRunResult, error)

// agent-run update-po command implementation.
func CmdAgentRunUpdatePo(agentName, poFile string) error

// Run multiple agent-test update-po operations.
func RunAgentTestUpdatePo(agentName, poFile string, runs int, cfg *config.AgentConfig) ([]RunResult, float64, error)

// agent-test update-po command implementation.
func CmdAgentTestUpdatePo(agentName, poFile string, runs int, skipConfirmation bool) error
```

The exact signatures can be aligned with existing `RunAgentUpdatePot` / `CmdAgentRunUpdatePot` / `RunAgentTestUpdatePot` implementations for consistency.

## 3. Implementation Steps

### Step 1: Implement PO Entry Counting

**Tasks**:
1. Implement `CountPoEntries()` in `util/agent.go` (or a suitable file), reusing logic from `CountPotEntries()` where possible.
2. Ensure header entries (`msgid ""`) are excluded from the count.
3. Handle multi-line `msgid` entries correctly.

**Validation**:
- Unit tests with sample PO files (including header, normal entries, and obsolete entries).
- Test with an empty file and invalid file.

### Step 2: Implement ValidatePoEntryCount

**Tasks**:
1. Implement `ValidatePoEntryCount()` in `util/agent-run.go` alongside `ValidatePotEntryCount()`.
2. Support both "before update" and "after update" stages.
3. Handle missing files according to the rules described above.
4. Produce clear error messages showing expected and actual counts and the stage.

**Validation**:
- Unit tests covering:
  - Validation disabled (nil/0 expected count).
  - File missing in "before update" stage (actual = 0).
  - File missing in "after update" stage (error).
  - Matching and non-matching counts.

### Step 3: Implement RunAgentUpdatePo

**Tasks**:
1. Implement `RunAgentUpdatePo()` in `util/agent-run.go`.
2. Reuse the existing `AgentRunResult` structure or extend it if necessary to store PO-specific data.
3. Implement the flow described in Section 2.5.1 (load config, select agent, pre-validation, execute agent, post-validation, syntax validation).
4. Integrate PO entry-count validation using `ValidatePoEntryCount()`.
5. Use `ValidatePoFile()` to check `po/XX.po` syntax.

**Validation**:
- Unit tests and/or integration tests that mock the agent command (e.g., `echo`).
- Tests for:
  - Pre-validation failures.
  - Post-validation failures.
  - Agent command failures.
  - Success with validation enabled.
  - Success with validation disabled.

### Step 4: Implement CmdAgentRunUpdatePo and CLI Wiring

**Tasks**:
1. Implement `CmdAgentRunUpdatePo()` in `util/agent-run.go` (or a new file as appropriate).
2. In `cmd/agent-run.go`, register the `update-po` subcommand:
   - Add `--agent` flag.
   - Add optional `po/XX.po` positional argument.
3. Wire the CLI command to call `CmdAgentRunUpdatePo()` with parsed parameters.
4. Add help text for the new subcommand, including description and examples.

**Validation**:
- `git-po-helper agent-run -h` shows the `update-po` subcommand.
- `git-po-helper agent-run update-po -h` shows help and flags.
- PO file argument and `--agent` flag are parsed correctly.

### Step 5: Implement RunAgentTestUpdatePo and CmdAgentTestUpdatePo

**Tasks**:
1. Implement `RunAgentTestUpdatePo()` in `util/agent-test.go`, mirroring `RunAgentTestUpdatePot()` but calling `RunAgentUpdatePo()`.
2. Implement `CmdAgentTestUpdatePo()` in `util/agent-test.go` to:
   - Confirm execution (unless skip flag is set).
   - Load configuration.
   - Determine `runs` value.
   - Call `RunAgentTestUpdatePo()`.
   - Display results using `displayTestResults()`.
3. In `cmd/agent-test.go`, register the `update-po` subcommand:
   - Add `--agent` and `--runs` flags.
   - Add optional `po/XX.po` positional argument.
4. Ensure that `agent-test update-po` shares as much logic as possible with `agent-test update-pot` to avoid duplication.

**Validation**:
- `git-po-helper agent-test -h` shows `update-po` subcommand.
- `git-po-helper agent-test update-po -h` shows help and flags.
- Multiple runs execute correctly, and average score is computed.

### Step 6: Error Handling and Logging

**Tasks**:
1. Ensure all error paths log clear and actionable messages using logrus.
2. Include hints in error messages where appropriate (e.g., missing config, missing prompt, invalid PO syntax).
3. Avoid leaking sensitive information from agent stderr while preserving enough detail for debugging.

**Validation**:
- Manual review of log messages.
- Tests that trigger typical error paths.

### Step 7: Documentation and Tests

**Tasks**:
1. Maintain this design document (`docs/design/agent-run-update-po.md`).
2. Add or update integration tests under `test/` (e.g., `t0090-agent-run.sh`, `t0091-agent-test.sh`) to cover `update-po` scenarios.
3. Optionally update `docs/agent-commands.md` and example configuration files to include `update-po` usage.

**Validation**:
- Integration tests pass.
- Example configuration works as expected.

## 4. Testing Strategy

### 4.1 Unit Tests

- **Configuration Loading**:
  - Verify `po_entries_before_update` and `po_entries_after_update` are parsed correctly.
  - Test missing or invalid configuration scenarios.
- **PO Entry Counting**:
  - Test `CountPoEntries()` with typical PO files, empty files, and invalid files.
- **Validation Logic**:
  - Test `ValidatePoEntryCount()` in all combinations of enabled/disabled stages, missing files, and matching/non-matching counts.
- **Command Logic**:
  - Test `RunAgentUpdatePo()` using a mock or simple agent command (e.g., `echo`).

### 4.2 Integration Tests

- **Full Workflow**:
  - `agent-run update-po` with:
    - Validation enabled and passing.
    - Validation enabled and failing before update.
    - Validation enabled and failing after update.
    - Validation disabled.
  - `agent-test update-po` with multiple runs and various configurations.
- **Error Scenarios**:
  - Missing configuration file.
  - Agent not configured or incorrect `--agent` name.
  - Missing `prompt.update_po`.
  - Agent command failure.
  - Invalid PO file syntax after agent execution.

### 4.3 Manual Testing

- Test with real agent commands (if available) to verify that:
  - `po/git.pot` and `po/XX.po` are updated as expected.
  - Entry-count validation behaves correctly under real-world changes.
  - Scoring and average score output make sense for repeated runs.

## 5. Future Considerations

- **Translate Command**: Extend similar validation mechanisms to `agent-run translate` / `agent-test translate`, using `po_new_entries_after_update` and `po_fuzzy_entries_after_update` to ensure that new and fuzzy entries are fully translated (counts go to zero).
- **Review Command**: Design validation and scoring for `agent-run review` / `agent-test review`, possibly based on review comments or diff analysis.
- **Partial Scoring**: Support more granular scoring (e.g., partial points when only some validations pass) instead of the current 0/100 model.
- **Parallel Execution**: For `agent-test`, consider parallelizing multiple runs to reduce total test time (with care for shared state in the repository).
- **CI/CD Integration**: Provide recommended configurations and thresholds for integrating `agent-test update-po` into CI pipelines.
