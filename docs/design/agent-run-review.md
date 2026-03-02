# Design Document: agent-run review and agent-test review

## 1. Original Requirements

### 1.1 Command Structure

The `agent-run` and `agent-test` subcommands are being extended with a new `review` operation to enable automated code agent-based review of PO file translations.

**agent-run review:**
```bash
git-po-helper agent-run review [--commit commit] [--since commit] [po/XX.po]
```

This command uses a code agent with a configured prompt to review translations in a PO file. It can review:
- Local changes (unstaged or staged) in the PO file (default behavior)
- Changes in a specific commit (using `--commit`)
- Changes since a specific commit (using `--since`)

**agent-test review:**
```bash
git-po-helper agent-test review [--agent <agent-name>] [--runs <n>] [--commit commit] [--since commit] [po/XX.po]
```

This command runs the `agent-run review` operation multiple times (default: 5, configurable via `--runs` or config file) and provides an average score based on the review JSON results.

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
agents:
  claude:
    cmd: ["claude", "-p", "{{.prompt}}"]
  gemini:
    cmd: ["gemini", "--prompt", "{{.prompt}}"]
```

### 1.3 Key Requirements

1. **Agent Selection**: If only one agent is configured, `--agent` flag is optional. If multiple agents exist, `--agent` is required.

2. **Review Mode Selection**:
   - Use `prompt.review` with `{{.source}}` placeholder (review mode determined by `--range`, `--commit`, or `--since`)

3. **Prompt Template**: The prompt from configuration is used, with placeholders replaced:
   - `{{.prompt}}` → the actual prompt text
   - `{{.source}}` → po file path (e.g., "po/XX.po")
   - `{commit}` → commit ID (HEAD, specific commit, or since commit)

4. **JSON Output**: The `agent-run review` command generates a JSON file `po/XX-reviewed.json` containing:
   - `total_entries`: Total number of translation entries reviewed
   - `issues`: Array of review issues, each containing:
     - `msgid`: Original message ID
     - `msgstr`: Translated message string
     - `score`: Issue score (0 = critical, 2 = minor, 3 = perfect)
     - `description`: Description of the issue
     - `suggestion`: Suggested improvement

5. **Scoring Model**:
   - Each entry has a maximum of 3 points
   - Critical issues (must fix) = 0 points
   - Minor issues (needs adjustment) = 2 points
   - Perfect entries = 3 points
   - Final score = (total_score / (total_entries * 3)) * 100

6. **Testing Mode**: `agent-test review` runs the operation `--runs` times and:
   - Calculates average score from all JSON results
   - Saves results to `output/<agent-name>/<iteration-number>/` directory:
     - `XX-reviewed.po`: The reviewed PO file
     - `review.log`: Execution log (stdout + stderr)
     - `XX-reviewed.json`: Review JSON result
   - If directory exists, overwrite files

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
│  │ │review    │ │              │ │review    │ │            │
│  │ └──────────┘ │              │ └──────────┘ │            │
│  └──────────────┘              └──────────────┘            │
│         │                              │                     │
│         └──────────────┬──────────────┘                     │
│                        │                                     │
│         ┌──────────────▼──────────────┐                     │
│         │   util/agent-run.go         │                     │
│         │   - RunAgentReview()        │                     │
│         │   - CmdAgentRunReview()      │                     │
│         │   - ParseReviewJSON()        │                     │
│         │   - SaveReviewJSON()         │                     │
│         └──────────────────────────────┘                     │
│                        │                                     │
│         ┌──────────────▼──────────────┐                     │
│         │   util/agent-test.go        │                     │
│         │   - RunAgentTestReview()    │                     │
│         │   - CmdAgentTestReview()    │                     │
│         │   - SaveReviewResults()     │                     │
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

The configuration uses the existing `AgentConfig` structure:

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
    Runs *int `yaml:"runs"`
}

type Agent struct {
    Cmd []string `yaml:"cmd"`
}
```

**Configuration Loading**:
- Use existing `config.LoadAgentConfig()` function
- Search for `git-po-helper.yaml` in repository root
- Validate required fields (at least one agent, prompt.review)
- Merge CLI arguments with config (CLI takes precedence)

### 2.3 Command Implementation

#### 2.3.1 agent-run review

**Flow**:
1. Load configuration from `git-po-helper.yaml`
2. Determine agent to use:
   - If `--agent` provided, use that agent
   - If only one agent in config, use it
   - Otherwise, return error requiring `--agent`
3. Determine PO file path:
   - If `po/XX.po` argument provided, use it
   - Otherwise, use `default_lang_code` from config to construct `po/XX.po`
   - Return error if neither provided and `default_lang_code` not configured
4. Determine review mode:
   - If `--commit <commit>` provided: Use commit mode
   - If `--since <commit>` provided: Use since mode
   - Otherwise: Default to `--since HEAD` (local changes)
5. Get prompt based on review mode:
   - Use `prompt.review` with `{{.source}}` placeholder
6. Replace placeholders in agent command:
   - `{{.prompt}}` → prompt text
   - `{{.source}}` → PO file path (e.g., "po/XX.po")
   - `{commit}` → commit ID (HEAD, specific commit, or since commit)
7. Execute agent command
8. Parse agent output to extract JSON result:
   - Look for JSON object in stdout
   - Validate JSON structure matches `ReviewJSONResult`
   - If JSON not found or invalid, return error
9. Save JSON result to `po/XX-reviewed.json`
10. Calculate and display review score
11. Return success/failure

**Success Criteria**:
- Agent command exits with code 0
- Agent output contains valid JSON matching `ReviewJSONResult` structure
- JSON file is successfully saved to `po/XX-reviewed.json`
- PO file exists and is valid (can be checked with `msgfmt`)

**Error Handling**:
- Configuration file not found: warn but continue with defaults if possible
- Agent not found: return error
- Agent command fails: return error with stderr output
- Invalid commit reference: return error
- JSON parsing fails: return error with agent output
- Invalid JSON structure: return error with validation details

**JSON Output Format**:

```json
{
  "total_entries": 2592,
  "issues": [
    {
      "msgid": "commit",
      "msgstr": "承诺",
      "score": 0,
      "description": "术语错误：'commit'应译为'提交'",
      "suggestion": "提交"
    },
    {
      "msgid": "repository",
      "msgstr": "仓库",
      "score": 2,
      "description": "一致性问题：其他地方使用'版本库'",
      "suggestion": "版本库"
    },
    {
      "msgid": "The file has been modified",
      "msgstr": "文件已被修改了",
      "score": 2,
      "description": "风格问题：表达冗余",
      "suggestion": "文件已修改"
    }
  ]
}
```

**Scoring Calculation**:

The scoring model works as follows:
- Maximum possible score = `total_entries * 3`
- For each issue, deduct `(3 - issue.score)` points
- Final score = `(total_score / maximum_possible_score) * 100`
- Score is clamped to 0-100 range

Example:
- 2592 entries = 7776 maximum points
- 3 issues with scores 0, 2, 2
- Deductions: (3-0) + (3-2) + (3-2) = 3 + 1 + 1 = 5 points
- Final score = ((7776 - 5) / 7776) * 100 = 99.94

#### 2.3.2 agent-test review

**Flow**:
1. Load configuration
2. Determine number of runs:
   - Use `--runs` if provided
   - Otherwise use `agent-test.runs` from config
   - Default to 5 if neither provided
3. For each run:
   - **Save output to directory**: `output/<agent-name>/<iteration-number>/`
     - Create directory if it doesn't exist
     - If directory exists, overwrite files
   - Run `agent-run review` logic
   - **Save results**:
     - Copy reviewed PO file to `output/<agent-name>/<iteration-number>/XX-reviewed.po`
     - Save execution log (stdout + stderr) to `output/<agent-name>/<iteration-number>/review.log`
     - Copy JSON result to `output/<agent-name>/<iteration-number>/XX-reviewed.json`
   - Parse JSON and calculate score using `CalculateReviewScore()`
   - Record score for this run
4. Calculate average score: `(sum of scores) / number of runs`
5. Display results:
   - Individual run results (including scores and issue counts)
   - Average score
   - Summary statistics (success count, failure count)

**Scoring**:
- Each run produces a JSON file with review results
- Score is calculated from JSON using `CalculateReviewScore()`
- Success = score > 0 (agent executed and produced valid JSON)
- Failure = score = 0 (agent failed or produced invalid JSON)
- Average = sum of scores / number of runs

**Output Directory Structure**:

```
output/
├── <agent-name>/
│   ├── 1/
│   │   ├── XX-reviewed.po
│   │   ├── review.log
│   │   └── XX-reviewed.json
│   ├── 2/
│   │   ├── XX-reviewed.po
│   │   ├── review.log
│   │   └── XX-reviewed.json
│   └── ...
```

**File Naming**:
- PO file: `XX-reviewed.po` (where XX is the language code, e.g., `zh_CN-reviewed.po`)
- Log file: `review.log`
- JSON file: `XX-reviewed.json` (e.g., `zh_CN-reviewed.json`)

### 2.4 Data Structures

#### 2.4.1 ReviewJSONResult

```go
// ReviewIssue represents a single issue in a review JSON result.
type ReviewIssue struct {
    MsgID       string `json:"msgid"`
    MsgStr      string `json:"msgstr"`
    Score       int    `json:"score"`
    Description string `json:"description"`
    Suggestion  string `json:"suggestion"`
}

// ReviewJSONResult represents the overall review JSON format produced by an agent.
type ReviewJSONResult struct {
    TotalEntries int           `json:"total_entries"`
    Issues       []ReviewIssue `json:"issues"`
}
```

#### 2.4.2 AgentRunResult (Extended)

The existing `AgentRunResult` structure should be extended to include review-specific fields:

```go
type AgentRunResult struct {
    // ... existing fields ...

    // Review-specific fields
    ReviewJSON      *ReviewJSONResult `json:"review_json,omitempty"`
    ReviewScore     int              `json:"review_score,omitempty"`
    ReviewJSONPath  string           `json:"review_json_path,omitempty"`
}
```

### 2.5 Utility Functions

#### 2.5.1 Parse Review JSON

```go
func ParseReviewJSON(jsonData []byte) (*ReviewJSONResult, error)
```

Parses JSON output from agent:
- Extracts JSON object from stdout (may contain other text)
- Validates JSON structure matches `ReviewJSONResult`
- Validates score values are in range 0-3
- Returns parsed result or error

#### 2.5.2 Save Review JSON

```go
func SaveReviewJSON(poFile string, review *ReviewJSONResult) (string, error)
```

Saves review JSON result to file:
- Determines output path: `po/XX-reviewed.json` (where XX is language code from PO file)
- Creates directory if needed
- Writes JSON with proper formatting
- Returns file path or error

#### 2.5.3 Calculate Review Score

```go
func CalculateReviewScore(review *ReviewJSONResult) (int, error)
```

Calculates 0-100 score from review JSON:
- Validates `total_entries > 0`
- Validates all issue scores are in range 0-3
- Calculates: `(total_score / (total_entries * 3)) * 100`
- Returns score (0-100) or error

This function already exists in `util/agent-run.go` and should be reused.

#### 2.5.4 Save Review Results (for agent-test)

```go
func SaveReviewResults(agentName string, runNumber int, poFile string, jsonFile string, stdout, stderr []byte) error
```

Saves review results to output directory:
- Creates `output/<agent-name>/<iteration-number>/` directory
- Copies PO file to `XX-reviewed.po`
- Copies JSON file to `XX-reviewed.json`
- Saves execution log to `review.log`
- Overwrites existing files if directory exists
- Returns error if any operation fails

#### 2.5.5 Extract JSON from Agent Output

```go
func ExtractJSONFromOutput(output []byte) ([]byte, error)
```

Extracts JSON object from agent output:
- Searches for JSON object boundaries (`{` and `}`)
- Handles cases where output contains other text before/after JSON
- Returns JSON bytes or error if not found

### 2.6 Integration Points

**Existing Code Reuse**:
- `util/agent-run.go`: `CalculateReviewScore()` - already implemented
- `util/agent-run.go`: `ReviewIssue`, `ReviewJSONResult` - already defined
- `util/agent.go`: `SelectAgent()`, `ExecuteAgentCommand()` - agent execution
- `repository/repository.go`: `WorkDir()` - for getting repository root
- `util/const.go`: `PoDir` constant
- `config/agent.go`: `LoadAgentConfig()` - configuration loading
- `util/agent-test.go`: `SaveTranslateResults()` - pattern for saving results

**New Dependencies**:
- JSON parsing: Use `encoding/json` (standard library)
- File operations: Use `os`, `path/filepath` (standard library)

## 3. Implementation Steps

### Step 1: Extend Data Structures

**Tasks**:
1. Review existing `ReviewIssue` and `ReviewJSONResult` structures in `util/agent-run.go`
2. Extend `AgentRunResult` to include review-specific fields
3. Ensure `CalculateReviewScore()` function is complete and tested

**Files to Modify**:
- `util/agent-run.go` - Extend `AgentRunResult` struct

**Validation**:
- `go build` succeeds
- Data structures compile correctly

### Step 2: Implement JSON Parsing Utilities

**Tasks**:
1. Implement `ExtractJSONFromOutput()` function
2. Implement `ParseReviewJSON()` function
3. Add validation for JSON structure and score ranges
4. Handle edge cases (empty output, malformed JSON, etc.)

**Files to Create/Modify**:
- `util/agent-run.go` - Add JSON parsing functions

**Functions to Implement**:
```go
func ExtractJSONFromOutput(output []byte) ([]byte, error)
func ParseReviewJSON(jsonData []byte) (*ReviewJSONResult, error)
```

**Validation**:
- Unit test with sample agent outputs
- Test with JSON embedded in other text
- Test with invalid JSON
- Test with missing fields

### Step 3: Implement Review JSON Saving

**Tasks**:
1. Implement `SaveReviewJSON()` function
2. Determine output file path from PO file path
3. Format JSON with proper indentation
4. Handle file creation errors

**Files to Modify**:
- `util/agent-run.go` - Add JSON saving function

**Functions to Implement**:
```go
func SaveReviewJSON(poFile string, review *ReviewJSONResult) (string, error)
```

**Validation**:
- Unit test with various PO file paths
- Test file creation and writing
- Verify JSON formatting

### Step 4: Implement agent-run review Core Logic

**Tasks**:
1. Implement `RunAgentReview()` function
2. Implement review mode selection logic:
   - Handle `--commit` mode
   - Handle `--since` mode
   - Handle default (local changes) mode
3. Implement prompt selection and placeholder replacement
4. Execute agent command
5. Parse JSON from agent output
6. Save JSON to file
7. Calculate and return score

**Files to Create/Modify**:
- `util/agent-run.go` - Add review logic

**Functions to Implement**:
```go
func RunAgentReview(cfg *config.AgentConfig, agentName, poFile, commit, since string) (*AgentRunResult, error)
```

**Validation**:
- Unit test with mock agent outputs
- Test all review modes (commit, since, default)
- Test prompt replacement
- Test JSON parsing and saving
- Test error cases

### Step 5: Implement agent-run review Command

**Tasks**:
1. Implement `CmdAgentRunReview()` function
2. Load configuration
3. Handle command-line arguments
4. Call `RunAgentReview()`
5. Display results and handle errors

**Files to Modify**:
- `util/agent-run.go` - Add command function

**Functions to Implement**:
```go
func CmdAgentRunReview(agentName, poFile, commit, since string) error
```

**Validation**:
- Integration test with mock agent
- Test with various command-line arguments
- Test error handling

### Step 6: Add CLI Command for agent-run review

**Tasks**:
1. Review existing `review` subcommand in `cmd/agent-run.go`
2. Ensure flags are correctly defined:
   - `--commit` flag
   - `--since` flag
   - `--agent` flag
3. Ensure argument handling is correct (optional `po/XX.po`)
4. Verify help text is accurate

**Files to Modify**:
- `cmd/agent-run.go` - Review and update if needed

**Validation**:
- `git-po-helper agent-run review -h` shows correct help
- Flags are parsed correctly
- Command executes without errors (with mock agent)

### Step 7: Implement Review Results Saving (for agent-test)

**Tasks**:
1. Implement `SaveReviewResults()` function
2. Create output directory structure
3. Copy PO file to output directory
4. Copy JSON file to output directory
5. Save execution log (stdout + stderr)
6. Handle file overwriting

**Files to Modify**:
- `util/agent-test.go` - Add review results saving function

**Functions to Implement**:
```go
func SaveReviewResults(agentName string, runNumber int, poFile string, jsonFile string, stdout, stderr []byte) error
```

**Validation**:
- Unit test directory creation
- Test file copying
- Test log file creation
- Test overwriting existing files

### Step 8: Implement agent-test review Core Logic

**Tasks**:
1. Implement `RunAgentTestReview()` function
2. Implement run loop
3. For each run:
   - Call `RunAgentReview()`
   - Save results to output directory
   - Parse JSON and calculate score
   - Record score
4. Calculate average score
5. Return results

**Files to Modify**:
- `util/agent-test.go` - Add review test logic

**Functions to Implement**:
```go
func RunAgentTestReview(cfg *config.AgentConfig, agentName, poFile string, runs int, commit, since string) ([]RunResult, float64, error)
```

**Validation**:
- Unit test with multiple runs
- Test result saving
- Test score calculation
- Test average calculation

### Step 9: Implement agent-test review Command

**Tasks**:
1. Implement `CmdAgentTestReview()` function
2. Load configuration
3. Handle command-line arguments
4. Call `RunAgentTestReview()`
5. Display results in readable format

**Files to Modify**:
- `util/agent-test.go` - Add command function

**Functions to Implement**:
```go
func CmdAgentTestReview(agentName, poFile string, runs int, skipConfirmation bool, commit, since string) error
func displayReviewTestResults(results []RunResult, averageScore float64, totalRuns int)
```

**Validation**:
- Integration test with multiple runs
- Test result display
- Test error handling

### Step 10: Add CLI Command for agent-test review

**Tasks**:
1. Review existing `review` subcommand in `cmd/agent-test.go`
2. Ensure flags are correctly defined:
   - `--commit` flag
   - `--since` flag
   - `--agent` flag
   - `--runs` flag
3. Ensure argument handling is correct (optional `po/XX.po`)
4. Verify help text is accurate

**Files to Modify**:
- `cmd/agent-test.go` - Review and update if needed

**Validation**:
- `git-po-helper agent-test review -h` shows correct help
- Flags are parsed correctly
- Command executes without errors (with mock agent)

### Step 11: Error Handling and Logging

**Tasks**:
1. Add appropriate log messages throughout
2. Handle all error cases gracefully
3. Provide helpful error messages
4. Use existing logging patterns (logrus)
5. Add debug logging for JSON parsing and scoring

**Validation**:
- Error messages are clear and actionable
- Logging follows project conventions
- Debug logs provide useful information

### Step 12: Documentation and Testing

**Tasks**:
1. Update user documentation (`docs/agent-commands.md`)
2. Add example configuration file entries
3. Write integration tests
4. Test with real agents (if available)
5. Verify JSON output format matches specification

**Files to Create/Modify**:
- `docs/agent-commands.md` - Add review command documentation
- `test/t*agent-run-review*.sh` - Integration tests
- Example `git-po-helper.yaml` in docs

**Validation**:
- Documentation is clear and complete
- Integration tests pass
- Example config works
- JSON output matches specification

### Step 13: Code Review and Refinement

**Tasks**:
1. Code review for Go best practices
2. Ensure consistency with existing codebase
3. Optimize performance if needed
4. Add additional error handling if needed
5. Verify all edge cases are handled

**Validation**:
- Code follows project conventions
- No linter errors
- Performance is acceptable
- All error cases are handled

## 4. Testing Strategy

### 4.1 Unit Tests

- JSON parsing with various formats
- JSON extraction from agent output (with surrounding text)
- Score calculation with various scenarios
- File path determination from PO file
- Review mode selection logic
- Prompt placeholder replacement

### 4.2 Integration Tests

- Full `agent-run review` workflow with mock agent:
  - Commit mode
  - Since mode
  - Default (local changes) mode
- Full `agent-test review` workflow with multiple runs:
  - Result saving to output directory
  - Score calculation and averaging
  - File overwriting behavior
- Error scenarios:
  - Missing config
  - Invalid agent
  - Invalid commit reference
  - Invalid JSON output
  - File I/O errors

### 4.3 Manual Testing

- Test with real agent commands (if available)
- Verify JSON output format matches specification
- Verify scoring calculation is correct
- Verify output directory structure
- Test with various PO files and language codes

## 5. Future Considerations

- Support for incremental review (only review changed entries)
- Support for review filters (e.g., only show critical issues)
- Support for review export in other formats (CSV, Markdown)
- Integration with CI/CD pipelines for automated review
- Support for review templates or custom scoring models
- Support for batch review of multiple PO files
- Support for review comparison (compare reviews from different runs)
