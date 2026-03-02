# Design Document: agent-run update-pot and agent-test update-pot

## 1. Original Requirements

### 1.1 Command Structure

The `agent-run` and `agent-test` subcommands are designed to integrate code agents (like Claude, Gemini, etc.) into the git-po-helper workflow for automating localization tasks.

**agent-run update-pot:**
```bash
git-po-helper agent-run update-pot [--agent <agent-name>]
```

This command uses a code agent with a configured prompt to update the `po/git.pot` template file according to `po/README.md`.

**agent-test update-pot:**
```bash
git-po-helper agent-test update-pot [--agent <agent-name>] [--runs <n>]
```

This command runs the `agent-run update-pot` operation multiple times (default: 5, configurable via `--runs` or config file) and provides an average score where success = 100 points and failure = 0 points.

### 1.2 Configuration File

The commands read from `git-po-helper.yaml` configuration file. Example:

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
  po_new_entries_after_update: null
  po_fuzzy_entries_after_update: null
agents:
  claude:
    cmd: ["claude", "-p", "{{.prompt}}"]
  gemini:
    cmd: ["gemini", "--prompt", "{{.prompt}}"]
```

### 1.3 Key Requirements

1. **Agent Selection**: If only one agent is configured, `--agent` flag is optional. If multiple agents exist, `--agent` is required.

2. **Prompt Template**: The prompt from `prompt.update_pot` is used, with placeholders replaced:
   - `{{.prompt}}` → the actual prompt text
   - `{{.source}}` → po file path (not used in update-pot)
   - `{commit}` → commit ID (default: HEAD, not used in update-pot)

3. **Command Execution**: The agent command from `agents.<agent-name>.cmd` is executed with placeholders replaced.

4. **Testing Mode**: `agent-test` runs the operation `--runs` times (default from config) and calculates average success rate.

5. **Entry Count Validation**: If `pot_entries_before_update` and `pot_entries_after_update` are configured (not null and not 0), the system will:
   - Count entries in `po/git.pot` before calling the agent
   - Verify the count matches `pot_entries_before_update` (if configured)
   - Count entries in `po/git.pot` after calling the agent
   - Verify the count matches `pot_entries_after_update` (if configured)
   - If validation fails, the operation is marked as failed (score = 0)
   - If validation passes, the operation is marked as successful (score = 100)
   - If these values are not configured (null or 0), no validation is performed

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
│  │ │translate │ │              │ │translate │ │            │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  │ ┌──────────┐ │              │ ┌──────────┐ │            │
│  │ │review    │ │              │ │review    │ │            │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  └──────────────┘              └──────────────┘            │
│         │                              │                     │
│         └──────────────┬──────────────┘                     │
│                        │                                     │
│         ┌──────────────▼──────────────┐                     │
│         │   util/agent.go              │                     │
│         │   - LoadConfig()             │                     │
│         │   - RunAgent()               │                     │
│         │   - CountPotEntries()        │                     │
│         └──────────────────────────────┘                     │
│                        │                                     │
│         ┌──────────────▼──────────────┐                     │
│         │   config/agent.go            │                     │
│         │   - AgentConfig struct       │                     │
│         │   - LoadYAML()               │                     │
│         └──────────────────────────────┘                     │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Configuration Management

**File Location**: `git-po-helper.yaml` in the repository root (same directory as `.git/`).

**Configuration Structure**:

```go
type AgentConfig struct {
    DefaultLangCode string            `yaml:"default_lang_code"`
    Prompt          PromptConfig      `yaml:"prompt"`
    AgentTest       AgentTestConfig   `yaml:"agent-test"`
    Agents          map[string]Agent  `yaml:"agents"`
}

type PromptConfig struct {
    UpdatePot    string `yaml:"update_pot"`
    UpdatePo     string `yaml:"update_po"`
    Translate    string `yaml:"translate"`
    Review string `yaml:"review"`
}

type AgentTestConfig struct {
    Runs                        *int `yaml:"runs"`
    PotEntriesBeforeUpdate      *int `yaml:"pot_entries_before_update"`
    PotEntriesAfterUpdate       *int `yaml:"pot_entries_after_update"`
    PoEntriesBeforeUpdate       *int `yaml:"po_entries_before_update"`
    PoEntriesAfterUpdate        *int `yaml:"po_entries_after_update"`
    PoNewEntriesAfterUpdate     *int `yaml:"po_new_entries_after_update"`
    PoFuzzyEntriesAfterUpdate  *int `yaml:"po_fuzzy_entries_after_update"`
}

type Agent struct {
    Cmd []string `yaml:"cmd"`
}
```

**Configuration Loading**:
- Use `gopkg.in/yaml.v3` or `github.com/spf13/viper` (already in use) to load YAML
- Search for `git-po-helper.yaml` in repository root
- Validate required fields (at least one agent, prompt.update_pot)
- Merge CLI arguments with config (CLI takes precedence)

### 2.3 Command Implementation

#### 2.3.1 agent-run update-pot

**Flow**:
1. Load configuration from `git-po-helper.yaml`
2. Determine agent to use:
   - If `--agent` provided, use that agent
   - If only one agent in config, use it
   - Otherwise, return error requiring `--agent`
3. **Pre-validation** (if `pot_entries_before_update` is configured and not 0):
   - Count entries in `po/git.pot` using `CountPotEntries()`
   - Compare with `pot_entries_before_update`
   - If mismatch, return error and exit (score = 0)
4. Get prompt from `prompt.update_pot`
5. Replace placeholders in agent command:
   - `{{.prompt}}` → prompt text
6. Execute agent command
7. **Post-validation** (if `pot_entries_after_update` is configured and not 0):
   - Count entries in `po/git.pot` using `CountPotEntries()`
   - Compare with `pot_entries_after_update`
   - If mismatch, return error and exit (score = 0)
8. Validate pot file syntax (using `msgfmt` or similar)
9. Return success/failure

**Success Criteria**:
- Agent command exits with code 0
- `po/git.pot` file exists and is valid (can be checked with `msgfmt`)
- **Pre-validation** (if configured): Entry count before update matches `pot_entries_before_update`
- **Post-validation** (if configured): Entry count after update matches `pot_entries_after_update`
- If validation is not configured (null or 0), validation steps are skipped

**Error Handling**:
- Configuration file not found: warn but continue with defaults if possible
- Agent not found: return error
- Agent command fails: return error with stderr output
- Invalid pot file: return error
- **Pre-validation failure**: Return error with message indicating expected vs actual entry count before update (score = 0)
- **Post-validation failure**: Return error with message indicating expected vs actual entry count after update (score = 0)

#### 2.3.2 agent-test update-pot

**Flow**:
1. Load configuration
2. Determine number of runs:
   - Use `--runs` if provided
   - Otherwise use `agent-test.runs` from config
   - Default to 5 if neither provided
3. For each run:
   - **Pre-validation** (if `pot_entries_before_update` is configured and not 0):
     - Count entries in `po/git.pot` before update
     - Verify count matches `pot_entries_before_update`
     - If mismatch, mark as failure (score = 0) and skip agent execution
   - Run `agent-run update-pot` logic (if pre-validation passed)
   - **Post-validation** (if `pot_entries_after_update` is configured and not 0):
     - Count entries in `po/git.pot` after update
     - Verify count matches `pot_entries_after_update`
     - If mismatch, mark as failure (score = 0)
     - If match, mark as success (score = 100)
   - If validation is not configured, use agent command exit code:
     - Success (exit code 0) = 100 points
     - Failure (non-zero exit code) = 0 points
   - Record score for this run
4. Calculate average score: `(sum of scores) / number of runs`
5. Display results:
   - Individual run results (including validation status)
   - Average score
   - Summary statistics (success count, failure count, validation failures)

**Scoring**:
- Success = 100 points
- Failure = 0 points
- Average = sum of scores / number of runs

**Entry Count Validation**:
- **If `pot_entries_before_update` is configured (not null and not 0)**:
  - Before agent execution, count entries in `po/git.pot`
  - If count does not match `pot_entries_before_update`, mark run as failed (score = 0)
  - Skip agent execution if pre-validation fails
- **If `pot_entries_after_update` is configured (not null and not 0)**:
  - After agent execution, count entries in `po/git.pot`
  - If count does not match `pot_entries_after_update`, mark run as failed (score = 0)
  - If count matches, mark run as successful (score = 100)
- **If validation is not configured (null or 0)**:
  - No entry count validation is performed
  - Scoring is based solely on agent command exit code (0 = success, non-zero = failure)

### 2.4 Entry Count Validation Logic

The entry count validation is a critical feature for ensuring the agent updates the POT file correctly. The validation behavior is determined by the configuration values `pot_entries_before_update` and `pot_entries_after_update`.

#### 2.4.1 Validation Rules

1. **Null or Zero Values**: If a validation field is `null` or `0`, validation is **disabled** for that stage. The system will not perform any entry count checking.

2. **Non-Zero Values**: If a validation field has a non-zero value, validation is **enabled** and the system will:
   - Count entries in `po/git.pot` at the specified stage
   - Compare the actual count with the expected value
   - Mark the operation as failed (score = 0) if counts don't match
   - Mark the operation as successful (score = 100) if counts match

#### 2.4.2 Pre-Validation (Before Agent Execution)

**When**: `pot_entries_before_update` is configured (not null and not 0)

**Process**:
1. Count entries in `po/git.pot` using `CountPotEntries()`
2. Compare with `pot_entries_before_update`
3. **If mismatch**:
   - Log error message: `"entry count before update: expected {expected}, got {actual}"`
   - Return error immediately
   - **Do not execute agent command**
   - Score = 0
4. **If match**:
   - Continue to agent execution
   - Score determined by post-validation or agent exit code

**Use Case**: Ensures the POT file is in the expected state before the agent runs, useful for testing scenarios where you want to verify the starting condition.

#### 2.4.3 Post-Validation (After Agent Execution)

**When**: `pot_entries_after_update` is configured (not null and not 0)

**Process**:
1. Execute agent command (if pre-validation passed or was disabled)
2. Count entries in `po/git.pot` using `CountPotEntries()`
3. Compare with `pot_entries_after_update`
4. **If mismatch**:
   - Log error message: `"entry count after update: expected {expected}, got {actual}"`
   - Return error
   - Score = 0
5. **If match**:
   - Mark operation as successful
   - Score = 100

**Use Case**: Verifies that the agent correctly updated the POT file with the expected number of entries. This is the primary validation for ensuring agent correctness.

#### 2.4.4 Validation in agent-test Mode

In `agent-test` mode, validation is performed for each run:

- **Pre-validation failure**: The run is marked as failed (score = 0) and the agent is not executed for that run
- **Post-validation failure**: The run is marked as failed (score = 0) even if the agent command succeeded
- **Both validations pass**: The run is marked as successful (score = 100)
- **Validation disabled**: Scoring is based on agent command exit code (0 = 100, non-zero = 0)

The average score is calculated as: `(sum of all run scores) / number of runs`

#### 2.4.5 Example Scenarios

**Scenario 1: Both validations enabled**
```yaml
agent-test:
  pot_entries_before_update: 5000
  pot_entries_after_update: 5100
```
- Before agent: Verify 5000 entries (fail if not)
- After agent: Verify 5100 entries (fail if not)
- Success only if both match

**Scenario 2: Only post-validation enabled**
```yaml
agent-test:
  pot_entries_before_update: null
  pot_entries_after_update: 5100
```
- Before agent: No validation
- After agent: Verify 5100 entries (fail if not)

**Scenario 3: Validation disabled**
```yaml
agent-test:
  pot_entries_before_update: null
  pot_entries_after_update: null
```
- No entry count validation
- Scoring based on agent exit code only

### 2.5 Utility Functions

#### 2.4.1 Count Pot Entries

```go
func CountPotEntries(potFile string) (int, error)
```

Counts `msgid` entries in a POT file by:
- Opening the file
- Scanning for lines starting with `^msgid `
- Counting non-empty msgid entries (excluding header)

#### 2.4.2 Execute Agent Command

```go
func ExecuteAgentCommand(cmd []string, workDir string) (stdout, stderr []byte, err error)
```

Executes the agent command:
- Replaces placeholders in command
- Runs in specified working directory
- Captures stdout and stderr
- Returns exit code and outputs

#### 2.4.3 Validate Pot File

```go
func ValidatePoFile(potFile string) error
```

Validates POT file syntax using `msgfmt --check-format` or similar.

#### 2.4.4 Validate Entry Count

```go
func ValidatePotEntryCount(potFile string, expectedCount *int, stage string) error
```

Validates entry count in POT file:
- If `expectedCount` is nil or 0, returns nil (no validation)
- Otherwise, counts entries using `CountPotEntries()`
- Compares with `expectedCount`
- Returns error if mismatch, nil if match
- `stage` parameter is used for error messages ("before update" or "after update")

### 2.6 Integration Points

**Existing Code Reuse**:
- `util/update.go`: `UpdatePotFile()` - for understanding pot file handling
- `repository/repository.go`: `WorkDir()` - for getting repository root
- `util/const.go`: `PoDir`, `GitPot` constants
- `flag/flag.go`: Pattern for flag handling

**New Dependencies**:
- YAML parser: Use `gopkg.in/yaml.v3` (add to `go.mod`)
- Or extend viper to support YAML config files

## 3. Implementation Steps

### Step 1: Create Directory Structure and Configuration Package

**Tasks**:
1. Create `config/agent.go` for configuration structures
2. Create `util/agent.go` for agent execution logic
3. Add YAML dependency to `go.mod` if needed

**Files to Create**:
- `config/agent.go` - Configuration structures and loading
- `util/agent.go` - Agent execution utilities

**Files to Modify**:
- `go.mod` - Add YAML dependency (if not using viper's YAML support)

**Validation**:
- `go build` succeeds
- Configuration structures compile

### Step 2: Implement Configuration Loading

**Tasks**:
1. Implement `LoadAgentConfig()` function in `config/agent.go`
2. Search for `git-po-helper.yaml` in repository root
3. Parse YAML into `AgentConfig` struct
4. Validate required fields
5. Handle missing config file gracefully

**Functions to Implement**:
```go
func LoadAgentConfig() (*AgentConfig, error)
func (c *AgentConfig) Validate() error
```

**Validation**:
- Unit test with sample YAML file
- Test missing config file handling
- Test invalid YAML handling

### Step 3: Implement Pot Entry Counting

**Tasks**:
1. Implement `CountPotEntries()` in `util/agent.go`
2. Handle file reading errors
3. Parse POT file format correctly (skip header, count msgid entries)

**Functions to Implement**:
```go
func CountPotEntries(potFile string) (int, error)
```

**Validation**:
- Unit test with sample POT file
- Test with empty file
- Test with invalid file

### Step 4: Implement Agent Command Execution

**Tasks**:
1. Implement placeholder replacement (`{{.prompt}}`, `{{.source}}`, `{{.commit}}`)
2. Implement `ExecuteAgentCommand()` function
3. Handle command execution errors
4. Capture stdout/stderr

**Functions to Implement**:
```go
type PlaceholderVars map[string]string
func ReplacePlaceholders(template string, kv PlaceholderVars) (string, error)
func ExecuteAgentCommand(cmd []string, workDir string) ([]byte, []byte, error)
```

**Validation**:
- Unit test with mock commands
- Test placeholder replacement
- Test error handling

### Step 5: Implement agent-run update-pot Command

**Tasks**:
1. Create `cmd/agent-run.go` with `update-pot` subcommand
2. Integrate configuration loading
3. Implement agent selection logic
4. **Implement pre-validation**:
   - Check if `pot_entries_before_update` is configured (not null and not 0)
   - If configured, count entries and validate
   - Return error if validation fails
5. Execute agent command
6. **Implement post-validation**:
   - Check if `pot_entries_after_update` is configured (not null and not 0)
   - If configured, count entries and validate
   - Return error if validation fails
7. Validate pot file syntax after update
8. Return appropriate exit codes

**Files to Create**:
- `cmd/agent-run.go` - Main command structure
- `util/agent-run.go` - Business logic for agent-run

**Functions to Implement**:
```go
func CmdAgentRunUpdatePot(agentName string) error
func ValidatePotEntryCount(potFile string, expectedCount *int, stage string) error
```

**Validation**:
- Integration test with mock agent
- Test agent selection logic
- Test pre-validation (matching and non-matching counts)
- Test post-validation (matching and non-matching counts)
- Test with validation disabled (null/0 values)
- Test error cases

### Step 6: Implement agent-test update-pot Command

**Tasks**:
1. Create `cmd/agent-test.go` with `update-pot` subcommand
2. Implement run loop
3. **Implement validation logic**:
   - For each run, perform pre-validation if `pot_entries_before_update` is configured
   - Skip agent execution if pre-validation fails
   - Perform post-validation if `pot_entries_after_update` is configured
   - Score based on validation results (100 if pass, 0 if fail)
4. Implement scoring logic:
   - If validation is configured: score based on validation results
   - If validation is not configured: score based on agent exit code
5. Display results in readable format:
   - Show validation status for each run
   - Show entry counts (expected vs actual) for failed validations
6. Handle optional validation assertions

**Files to Create**:
- `cmd/agent-test.go` - Main command structure
- `util/agent-test.go` - Business logic for agent-test

**Functions to Implement**:
```go
func CmdAgentTestUpdatePot(agentName string, runs int) error
func RunAgentTestUpdatePot(agentName string, runs int) ([]int, float64, error)
// Returns: scores for each run, average score, error
```

**Validation**:
- Unit test scoring logic with validation enabled/disabled
- Integration test with multiple runs
- Test pre-validation failures (should skip agent execution)
- Test post-validation failures (should mark as failed)
- Test with validation disabled (should use agent exit code)
- Test optional assertions

### Step 7: Add CLI Flags and Integration

**Tasks**:
1. Add `--agent` flag to both commands
2. Add `--runs` flag to `agent-test`
3. Register commands in `cmd/root.go`
4. Add help text

**Files to Modify**:
- `cmd/agent-run.go` - Add flags
- `cmd/agent-test.go` - Add flags
- `cmd/root.go` - Register commands

**Validation**:
- `git-po-helper agent-run -h` shows help
- `git-po-helper agent-test -h` shows help
- Flags are parsed correctly

### Step 8: Error Handling and Logging

**Tasks**:
1. Add appropriate log messages
2. Handle all error cases gracefully
3. Provide helpful error messages
4. Use existing logging patterns (logrus)

**Validation**:
- Error messages are clear and actionable
- Logging follows project conventions

### Step 9: Documentation and Testing

**Tasks**:
1. Update `README.md` or create `docs/agent-commands.md`
2. Add example configuration file
3. Write integration tests
4. Test with real agents (if available)

**Files to Create/Modify**:
- `docs/agent-commands.md` - User documentation
- `test/t*agent-run*.sh` - Integration tests
- Example `git-po-helper.yaml` in docs

**Validation**:
- Documentation is clear and complete
- Integration tests pass
- Example config works

### Step 10: Code Review and Refinement

**Tasks**:
1. Code review for Go best practices
2. Ensure consistency with existing codebase
3. Optimize performance if needed
4. Add additional error handling if needed

**Validation**:
- Code follows project conventions
- No linter errors
- Performance is acceptable

## 4. Testing Strategy

### 4.1 Unit Tests

- Configuration loading with various YAML formats
- Placeholder replacement
- Pot entry counting
- Agent command execution (mocked)

### 4.2 Integration Tests

- Full `agent-run update-pot` workflow with mock agent
- Full `agent-test update-pot` workflow with multiple runs
- Entry count validation scenarios:
  - Pre-validation success and failure
  - Post-validation success and failure
  - Validation disabled (null/0 values)
  - Both validations enabled
  - Only pre-validation enabled
  - Only post-validation enabled
- Error scenarios (missing config, invalid agent, etc.)

### 4.3 Manual Testing

- Test with real agent commands (if available)
- Verify pot file updates correctly
- Verify scoring works as expected

## 5. Future Considerations

- Support for other subcommands (update-po, translate, review)
- Caching of agent responses
- Parallel execution for agent-test
- More sophisticated scoring metrics
- Integration with CI/CD pipelines
