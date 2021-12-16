# Changelog

Changes of git-po-helper.

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
* Check unmatched "%(fieldname)" in format of git-for-each-ref.
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
* check-po: check typos of unmatched variable names
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
