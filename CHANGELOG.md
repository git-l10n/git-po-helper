# Changelog

Changes of git-po-helper.

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
