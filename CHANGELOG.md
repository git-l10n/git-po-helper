# Changelog

Changes of git-po-helper.

## 0.8.0 (2026-03-14)

### Agent commands (agent-run, agent-test)

* feat: add agent-run and agent-test with update-po, update-pot, translate, review subcommands
* feat: support multiple AI agents (Claude, Codex, OpenCode, Gemini, Qwen, Qoder)
* feat: stream-json real-time output with type-specific icons (🤔 thinking, 🤖 text, 🔧 tool, 💬 user)
* feat: parse Claude/Gemini assistant content (text, thinking, tool_use) and user tool_result
* feat: improve Codex JSONL parsing (thread.started, item.started/completed, turn.completed)
* feat: improve OpenCode message output with tool input/output display
* feat: add Kind field for type-safe agent detection (claude, codex, opencode, gemini, qoder)
* feat: truncate long command display (256 + 128 bytes) and indent/wrap multi-line output at 80 chars
* feat: agent-test review aggregates JSON and uses lowest score per msgid
* feat: add --range, --commit, --since and two-file support to review commands
* feat: add --use-local-orchestration and --use-agent-md (rename from --all-with-llm) for review/translate
* feat: translate/review local orchestration aligned with po/AGENTS.md Task 3 and Task 4
* feat: review reads result from disk (po/review-done.json written by agent), not stdout
* feat: add report subcommand (agent-run report) to replace stat --review; return JSON path and improve output
* feat: add --batch-size for review batching; default batch size 100 (align with AGENTS.md)
* feat: invoke agent to fix PO when msgfmt fails (fix-po flow)
* feat: add NumTurns and execution time to agent-test output; execution time in agent-run summary
* fix: flush stdout so agent output appears without -v
* fix: route error output to stderr in main; send interactive prompts to stderr in compare
* fix: resolve golangci-lint unused/errcheck/govet/ineffassign

### Compare command

* feat: rename diff to compare, add --stat requirement
* feat: add --commit and --since, refine -r range parsing (a..b, a.., a)
* feat: merge new-entries into compare (default mode outputs new/changed entries)
* feat: add --json for JSON output; add --no-header to omit header
* feat: add JSON format support for input files
* feat: add --msgid for msgid-only comparison (ignore msgstr/fuzzy changes)
* feat: add --assert-no-changes and --assert-changes for CI
* feat: add -o/--output to write to file
* fix: skip obsolete entries in PoCompare loop

### msg-select and msg-cat

* feat: add msg-select command to extract PO/POT entries by index range
* feat: add msg-cat subcommand to merge PO/POT/JSON files (first occurrence of each msgid wins)
* feat: msg-select supports gettext JSON input and --json output; PO↔gettext JSON round-trip
* feat: add --head, --tail, --since as range shortcuts; allow without --range to select all
* feat: add --unset-fuzzy and --clear-fuzzy to msg-select and msg-cat
* feat: add --no-header to omit header; add -o/--output to write to file
* feat: add entry state filter options; support obsolete entries (#~ and #~|) in PO parsing and JSON
* fix: write empty file when no entries selected (no header-only output)
* fix: accept empty JSON input for msg-select and msg-cat

### stat command

* feat: add stat command for PO file statistics
* feat: support gettext JSON input; support multiple PO files
* feat: derive json and po from --review path, no args required; add Total() and POT file tests
* fix: count same-as-source entries as translated; fix stat-po count logic

### PO / gettext

* feat: add strDeQuote, BuildPoContent and ParsePoEntries round-trip test
* feat: set PoEntry.IsFuzzy from #, fuzzy flag comments; keep fuzzy state only in GettextEntry.Fuzzy
* feat: gettext JSON structs and header split for PO→JSON; PO format strings in GettextEntry for escape handling
* fix: allow blank lines in header comment block
* fix: preserve quotes in PO file header continuation lines; store PO format strings for consistent escape handling
* fix: convert absolute PO file path to relative path for git show

### Review

* feat: apply review suggestions to output PO file (applyReviewJSON); add IssueCount(), exclude score-3 from "With issues"
* feat: add ReviewIssue score enum constants (Critical, Major, Minor, Perfect); add gjson fallback for malformed LLM JSON
* feat: align review local orchestration with AGENTS.md Task 4 (review-input.po, review-todo.json, review-done.json, review-batch.txt)

### Configuration and prompts

* feat: add --config flag and unified config merge with default
* feat: add --prompt option to override prompts
* feat: add multiple AI agent configurations
* feat: add local-orchestration-translation and local-orchestration-review prompts
* feat: add prefix @ introducing po/README.md in prompts; update review prompt with quality checklist and JSON format
* feat: use Go template placeholders {{.source}}, {{.prompt}}, {{.dest}} in prompts

### Tests and documentation

* test: add integration test for translate --use-local-orchestration; PO round-trip and msg-select zh_CN example
* test: remove jq dependency in t0123 for cross-platform; isolate unit tests from GIT_DIR/GIT_WORK_TREE in pre-commit
* docs: add agent-commands.md, design docs for update-pot, update-po, translate, review; update README and AGENTS.md
* docs: document commit conventions for agents; use {{.source}} and {{.prompt}} in config docs

## 0.7.6 (2026-02-07)

* update: new option --no-file-location and --no-location
* update: use --add-location=file to remove location by default
* pot-file: change default to 'auto' with smart detection
* team: show members only with -m, and use -a to show all
* test: fix team members test case and add --all option test
* test-lib: sync with git-test-lib project
* actions: upgrade github actions versions (checkout v3->v5, setup-go v4->v6)
* docs: add AGENTS.md project guide and AI assistant config files

## 0.7.5 (2024-04-25)

* dict: dirty hacks on bg for git v2.45.0
* test: prepare for upgarde test repositories
* team: add -L to show language
* dict: update smudge table for bg for new keepwords
* dict: change KeepWordsPattern and add test cases
* Stop pretending that the l10n.yml workflow is outside git-l10n's ownership


## 0.7.3 (2024-02-10)

* dict: ca: new smudge entries
* dict: loose pattern to find typos of "refs/"
* dict: add pattern to find typos of refspecs
* typo: use more general expressions "mismatched patterns"
* test: use git-test-lib as our test framework


## 0.7.0 (2023-11-28)

* dict/bg: Smudge both msgId and msgStr
* util/bg: do not check boundary characters
* refactor: change style of definition for SmudgeMaps
* Fix typos: unmatched -> mismatched
* test: fixed chain-lint error in test cases
* test: replace sharness with git test-lib test framework
* actions: upgrade version of actions/checkout and actions/setup-go
* actions: do not run golint for go 1.17


## 0.6.5 (2023-08-07)

* test: no illegal fields among core commit metadata
* util: username must have at least one non-space character
* Download pot from master branch instead of main


## 0.6.4 (2022-09-27)

* dict: sv: new smudge entry for git 2.38
* refactor: ioutil package is obsolete for go 1.18
* dict: bg: new smudge entry for git 2.38
* dict: use ordered list for SmudgeMaps
* Add opiton --report-file-location=<none,warn,error>
* refactor: new option --report-typos=<none,warn,error>
* Show output of partial clone in debug level
* test: add test cases for "git-po-helper check-pot"
* bugfix: more diff entries should be ignored in output
* CI: lower version for golang is 1.16 now
* chekc-pot: find mismatched config variable in put file
* check-pot: show config variable in manpage or po/git.pot
* refactor: check-po: refactor to reuse scanning of po file
* repository: not panic if not in git.git repository
* Change repository name which holding pot file to pot-changes
* check-commits: do not check removed files
* Do not allow too many obsolete entries in comments
* Warn if there are untranslated, fuzzy or obsolete strings
* Rename option "--check-pot-file" to "--pot-file"
* When update XX.po, get latest pot file by downloading
* Quit if fail to download pot file
* refactor: create pot file using UpdatePotFile()
* Instead of using tmpfile for PO_FILE, use po/XX.po
* refactor: show prompt even for empty message
* Format output for core pot checking
* Tweak message for removing file-locations
* Tweak message for missing translation
* Do not show download progress in github actions
* TEAMS: show filename of po/TEAMS in error messages
* Documentation: update README and s@po-core/@po/@
* Add horizontal lines before report errors
* Fix go 1.4 incompatible issue: use ioutil.ReadAll
* Update po/git.pot and check missing translations
* check-commits: new checks for github-action
* check-po: new option "--check-file-location" to check no locations
* update: call make po-update if available
* init: call make po-init if available
* Show horizontal lines to separate groups of messages
* test: run test on git 2.36.0
* refactor: check commit changes using "diff-tree -z"
* refactor: return array of string instead errors
* refactor: add new helper functions to show error messages
* Makefile: find source files using git-ls-files
* Makefile: build before test
* contrib: update drivers for po diff and clean
* github actions: only run golint for go 1.17
* refactor: fix issues found by staticcheck
* contrib: filter to commit po files without location lines
* contrib: use msgcat for diff driver
* dict: remove typos section for bg as it is handled
* diff: ignore msgcmp return error


## 0.4.6 (2021-12-16)

* go mod: upgrade goconfig to 1.1.1
* dict: change 1 entry for bg smudge table
* dict: add 2 entries for bg smudge table
* dict: add smudge table for Korean language
* Suppress errors of commit-date drift for github actions
* refactor: check using golangci-lint
* Fix golint warnings
* Do not check line width for signatures and merge commit
* Use all versions of gettext installed to check po files
* dict: more entries for smudge table of bg language
* refactor: move global replace dict to seperate smuge maps


## 0.4.5 (2021-11-6)

* Only turn on hints if set gettext.useMultipleVersions
* Check gettext 0.14 incompatible issues
* refactor: add standalone package "gettext" to collect gettext
  versions and show hints
* dict: update smudge map for sv.po


## 0.4.3 (2021-10-22)

* Smudge on msgStr to suppress false positive for checking typos..
* Check mismatched "%(fieldname)" in format of git-for-each-ref.
* Test on po files of git 2.31.1 and latest version.


## 0.4.2 (2021-9-9)

* t0043: add check-commits test cases for partial clone
* Use goconfig to check git config for partial clone
* Show number of missing blobs fetching from partial clone
* Fix go 1.17 panic issue by update pkg golang.org/x/sys.


## 0.4.0 (2021-9-4)

* check-commits: can be run with bare repository
* check-commits: fetch missing blobs in a batch from partial clone
* Support new github-action event: `pull_request_target`
* Scan typos for option name with numbers
* check-commit: add new option "--github-action-event"
* check-commits: raise an error if fail to run rev-list
* check-commits: handle new branch action: <ZERO-OID>..<new-branch>
* Stop scanning if find no git-l10n related commit
* check-commits: fall back to threshold if too many commits
* Force colored log output for github-action
* refactor: use `util.Flag*` to get viper cached settings
* Add "iso-\*.go" so we can use "go install"
* test: exit with failure if downloading test repo fails


## 0.3.0 (2021-8-17)

* Running check-commits will check typos for each commit.
* Try to reduce a certain false positives when checking typos.
* test: run test on .po file from download git package.
* New option "--report-typos-as-errors" when checking typos.


## 0.2.0 (2021-8-9)

* check-po: do not check fragment keep words in a unicode string
* check-po: find typos in two directions
* test: add .po examples and search typos in examples


## 0.0.6 (2021-8-7)

* Check more types: command names, options, and shell variables.
* Update warning message for unable to locate gettext 0.14.
* github-action: add test on macOS.


## 0.0.5 (2021-7-3)

* Run CI using github action instead of azure-pipeline
* check-po: check typos of mismatched variable names
* Show number of commits checked
* Commit time is UTC time, no need to check offset


## 0.0.3 (2021-6-3)

* version: do not run pre-checks for version cmd
* check-commits: refactor warnings for too long subject
* bugfix: fix false positive for reporting no s-o-b
* team: show commit ID when fail to parse TEAMS file
* check-commits: checkout po/TEAMS to tmpfile before checking
* diff: update diff command output
* check-commits: check commit older than a half year
* azure pipeline: new trigger branch - pu


## 0.0.2 (2021-5-14)

Improvements:

* Add azure pipeline for build and test git-po-helper.
* Add GPL v2 license.
* Add README.md
* Makefile: run "go generate" when necessary

Bugfix:

* test: fix sed compatible issue in test cases
* test: make stable iconv output for test
* bugfix: filepath.Walk panic for non-exist dir


## 0.0.1 (2021-5-14)

The first release of git-po-helper.
