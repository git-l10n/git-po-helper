# Changelog

Changes of git-po-helper.

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
