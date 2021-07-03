# Changelog

Changes of git-po-helper.

## 0.0.4-1 (2021-7-3)

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
