# po_ai_translator Requirements and Design

## Original Requirements

Develop a program named `bin/po_ai_translator` with the following
command-line format:

```
po_ai_translator [-v] [--runs=5] [--lang-code=zh_CN] --agent claude
```

The application reads the configuration file `po_ai_translator.yaml`, using
the `<config>` example below as a reference. It must implement the following
behavior. The PO file path depends on arguments, so `po/XX.po` should be
interpreted as `po/<lang-code>.po`. Output filenames must use the iteration
start time as a prefix, so each iteration uses the same timestamp string
`<Year-Month-Date-Hour-Minute-Second>`.

1. At the start of each iteration, run
   `git restore --worktree --staged --source HEAD -- po/` to reset changes
   under `po/`.
2. Run `grep -c "^msgid " po/XX.po` and assert the count equals
   `assert.start_entry_count` if it is provided.
3. Use `--agent` to select the corresponding `cmd` in
   `po_ai_translator.yaml`, then run the command. Replace `{{.prompt}}` with
   `agent.<agent-name>.prompt` if present, otherwise use the global `prompt`.
   Replace `{lang-code}` with the YAML `lang_code` value.
4. While running the agent command, display output in the console. If `-v`
   (verbose) is set, show all output. Otherwise, use rolling output in a
   single line. Always write both stdout and stderr to the log file:
   `output/<agent-name>/<Year-Month-Date-Hour-Minute-Second>.log`.
5. When finished, run
   `git status --porcelain -- po/git.pot po/XX.po` and verify that
   `po/git.pot` is unchanged and `po/XX.po` is modified. If not, write an
   error message to
   `output/<agent-name>/<Year-Month-Date-Hour-Minute-Second>.err`.
6. Verify that the new `po` file has `^msgid ` entries equal to
   `assert.end_entry_count` if it is provided. If not, write an error message
   to the `.err` file.
7. Run `msgattrib --translated po/zh_CN.po` and count `^msgid ` entries
   (translated entries). The count must be greater than
   `assert.start_entry_count` if it is provided.
8. Ensure `fuzzy` and `untranslate` are both 0; otherwise write an error
   message to the `.err` file.
9. Copy the generated `po/XX.po` to
   `output/<agent-name>/<Year-Month-Date-Hour-Minute-Second>.po`.
10. Repeat from step 1 for `runs` iterations.

<config>
runs: 5
lang_code: "zh_CN"
assert:
  start_entry_count: null
  end_entry_count: null
prompt: "update {{.source}} and translate it according to po/README.md"
agents:
  claude:
    cmd: ["claude", "-p", "{{.prompt}}"]
    # Empty prompt means using the global prompt

  gemini:
    cmd: ["gemini", "--prompt", "{{.prompt}}"]
    # Empty prompt means using the global prompt

  custom:
    cmd: ["claude", "-p", "{{.prompt}}"]
    prompt: "custom prompt here"  # Override the global prompt
</config>

## Design Document

### Goals and Scope

- Provide a repeatable PO auto-translation runner with `runs` iterations.
- Support multiple agent commands and prompt overrides.
- Make results observable via console output, logs, error records, and
  archived outputs.

### Configuration and Arguments

- Configuration file: `po_ai_translator.yaml`.
- CLI arguments:
  - `--agent` required: the key under `agents`.
  - `--runs` optional: overrides `runs`.
  - `--lang-code` optional: overrides `lang_code`, affecting
    `po/<lang-code>.po`.
  - `-v` optional: controls output mode (full vs. rolling).

### Paths and Naming

- Working directory: repository root.
- Output directory: `output/<agent-name>/`, create if missing.
- Each iteration uses a timestamp `<Year-Month-Date-Hour-Minute-Second>`:
  - `output/<agent-name>/<timestamp>.log`
  - `output/<agent-name>/<timestamp>.err`
  - `output/<agent-name>/<timestamp>.po`

### Core Flow

1. **Iteration timestamp**: generate at iteration start so all files in the
   iteration share the same prefix.
2. **Reset workspace**: run
   `git restore --worktree --staged --source HEAD -- po/` to ensure
   repeatability.
3. **Start entry validation**: count `msgid` in `po/<lang-code>.po` and
   compare to `assert.start_entry_count` when provided.
4. **Build agent command**:
   - Select `agents.<name>.cmd`.
   - Use `agents.<name>.prompt` or the global `prompt` for `{{.prompt}}`.
   - Use the effective `lang_code` for `{lang-code}`.
5. **Run command and capture output**:
   - With `-v`, print stdout/stderr directly.
   - Without `-v`, render rolling output in one line while still writing the
     full output to the log file.
6. **Validate results**:
   - `git status --porcelain -- po/git.pot po/<lang-code>.po` must show no
     change in `git.pot` and changes in `po/<lang-code>.po`.
   - If `assert.end_entry_count` is provided, validate the `msgid` count.
   - `msgattrib --translated po/<lang-code>.po` must yield a `msgid` count
     greater than `assert.start_entry_count` when provided.
   - `fuzzy` and `untranslate` counts must be 0.
   - Any failure writes to `.err` and continues to the next iteration.
7. **Archive output**: copy `po/<lang-code>.po` to
   `output/<agent-name>/<timestamp>.po`.

### Error and Log Strategy

- `.log` stores full stdout/stderr.
- `.err` appends each validation failure with expected vs. actual values.
- Failures do not stop the overall `runs` loop.

### Commit Message Guidelines

- Commit messages must follow the
  [Conventional Commits Specification](https://www.conventionalcommits.org/en/v1.0.0/).
- Commit messages must be in English.
- Good commit messages are multi-line: title on line 1, blank line on line 2,
  and detailed description starting on line 3.
- The detailed description should explain the why and provide a concise
  summary of what changed, not just how it changed.
- The rationale should be inferred from the prompt and code changes.
- Each line must be <= 72 characters; wrap lines without adding extra blank
  lines.
- Use a HereDoc to run git commit, such as `git commit -F- <<-EOF`, instead
  of multiple `-m <message>` arguments to avoid extra blank lines.

## Development Steps Breakdown

Step 1: Add documentation and directory structure
- Use the Node.js/TypeScript project structure:
  ```
  ├── bin/
  │   ├── po_ai_translator  # Executable script (JavaScript with shebang)
  ├── src/
  │   ├── cli/
  │   │   ├── po_ai_translator.ts  # po_ai_translator entry
  │   ├── lib/
  │   │   ├── config.ts   # Shared config
  │   │   ├── runner.ts   # Shared logic
  │   │   └── utils.ts    # Utility helpers
  │   └── index.ts        # 库入口（可选）
  ├── dist/               # Compiled JavaScript (.gitignore)
  │   ├── cli/
  │   │   ├── po_ai_translator.js
  │   └── lib/
  │       └── ...
  ├── tests/
  │   └── po_ai_translator.test.ts
  ├── package.json
  ├── tsconfig.json
  ├── .gitignore
  └── README.md
  ```
- Commit new and changed files to the repository.

Step 2: Define CLI parsing and config loading
- Use a Node.js CLI parser (e.g. `commander`) to parse `-v`, `--runs`,
  `--lang-code`, and `--agent`.
- Use a YAML parser (e.g. `yaml`) to read `po_ai_translator.yaml`.
- Merge CLI overrides, validate required fields.
- Commit new and changed files to the repository.

Step 3: Implement iteration control and timestamp naming
- Loop `runs` times.
- Generate a timestamp at the start of each iteration and reuse it for all
  output file names in that iteration.
- Commit new and changed files to the repository.

Step 4: Implement workspace reset and start validation
- Run `git restore --worktree --staged --source HEAD -- po/`.
- Run `grep -c "^msgid " po/<lang-code>.po` and validate
  `assert.start_entry_count` when provided.
- Commit new and changed files to the repository.

Step 5: Build and execute the agent command
- Parse `agents.<agent-name>.cmd`.
- Substitute `{{.prompt}}` and `{lang-code}`.
- Start the subprocess and handle output:
  - With `-v`, print directly.
  - Without `-v`, render rolling output and write full output to the log.
- Commit new and changed files to the repository.

Step 6: Implement result validation and error output
- Check `git status --porcelain -- po/git.pot po/<lang-code>.po`.
- Validate `assert.end_entry_count`.
- Run `msgattrib --translated po/<lang-code>.po` and validate translations.
- Validate `fuzzy` and `untranslate` are 0.
- Write any failures to `output/<agent-name>/<timestamp>.err`.
- Commit new and changed files to the repository.

Step 7: Implement output archiving
- Copy `po/<lang-code>.po` to `output/<agent-name>/<timestamp>.po`.
- Commit new and changed files to the repository.

Step 8: Add error handling and exit codes
- Log command failures, missing files, and config errors to `.err`.
- Exit with a status code that reflects whether any iteration failed.
- Commit new and changed files to the repository.
