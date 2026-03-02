[![CI](https://github.com/git-l10n/git-po-helper/actions/workflows/main.yml/badge.svg)](https://github.com/git-l10n/git-po-helper/actions/workflows/main.yml)

**A helper program for Git localization (l10n) contributions**

## Overview

`git-po-helper` serves two main purposes:

1. **Quality checking** — Check conventions for Git l10n contributions. Pull requests to [git-l10n/git-po](https://github.com/git-l10n/git-po) must comply with file location, commit log format, PO syntax, and other rules. This tool automates these checks for coordinators and contributors.

2. **AI-assisted translation** — Integrate AI code agents (Claude, Gemini, Codex, OpenCode, etc.) to automate localization tasks: update POT templates, update PO files, translate new entries, and review translations. See [docs/agent-commands.md](docs/agent-commands.md) for configuration and usage.

## Conventions

`git-po-helper` is a helper program for git l10n coordinator and git l10n
contributors to check conventions for git l10n contributions. Pull request to
[git-l10n/git-po](https://github.com/git-l10n/git-po) must comply with the
following conventions.

* Only change files in the `po/` directory. Any l10n commit that violates this
  rule will be rejected.

  - Changes to the `git-gui/po` and `gitk-git/po` directories belong to the
    sub-projects "git-gui" and "gitk", which have their own workflows. Please
    DO NOT send pull requests for these projects here. See
    [Documentation/SubmittingPatches](https://github.com/git/git/blob/v2.31.0/Documentation/SubmittingPatches#L387-L393).

* Write a decent commit log for every l10n commit:

  - Add a prefix ("l10n:" followed by a space) in the subject of the commit log.
    Take history commits as an example: `git log --no-merges -- po/`.

  - Do not use non-ASCII characters in the subject of a commit.

  - The subject (the first line) of the commit log should have characters no more than 50.

  - Other lines of the commit log should not exceed 72 characters.

  - Like other git commits, add a "Signed-off-by:" signature in the trailer of the commit log.

    Add a "Signed-off-by:" signature automatically by running `git commit -s`.

  - Do not skew your clock to ensure accurate commit datetime.

* Squash trivial commits so that the pull request for each git l10n update
  window contains a clear and small number of commits.

* Check the "XX.po" file using the `msgfmt` command to make sure it has correct
  syntax.

* All translatable strings are present in your "po/XX.po". You can download
  the latest pot file from:

	https://github.com/git-l10n/pot-changes/raw/pot/main/po/git.pot

* Remove file-locations from your "po/XX.po" before submitting it to your
  commit. See the [Updating a "XX.po" file] section for reference in
  "po/README.md" of the Git repository.


To contribute for a new l10n language, contributor should follow additional
conventions:

* Initialize proper filename of the "XX.po" file conforming to iso-639 and
  iso-3166.

* Must complete a minimal translation based on the `po/git-core.pot` template.
  Using the following command to initialize the minimal `po/XX.po` file:

        make po-init PO_FILE="po/XX.po"

* Add a new entry in the `po/TEAMS` file with proper format.


## Prerequisites

`git-po-helper` is written in [golang](https://golang.org/), golang must be
installed before compiling.

Additional prerequisites needed by `git-po-helper`:

* git
* gettext
* iconv, which is used to check commit log encoding.
* gpg, which is used to verify commit with gpg signature.


## Build and install git-po-helper

Compile `git-po-helper` using the following commands:

```
$ git clone https://github.com/git-l10n/git-po-helper.git
$ cd git-po-helper
$ make
$ make test
```

Install `git-po-helper`:

```
$ cp git-po-helper /usr/local/bin/
```


## Workflows

See [po/README.md](https://github.com/git/git/tree/master/po#readme) of
the Git project.


## Usage of git-po-helper

```
$ git-po-helper -h
Helper for git l10n

Usage:
  git-po-helper [flags]
  git-po-helper [command]

Available Commands:
  agent-run     Run agent commands for automation
  agent-test    Test agent commands with multiple runs
  check         Check all ".po" files and commits
  check-commits Check commits for l10n conventions
  check-po      Check syntax of XX.po file
  check-pot     Check syntax of XX.pot file
  compare       Show changes between two l10n files
  help          Help about any command
  init          Create XX.po file
  msg-cat       Concatenate and merge PO/POT/JSON files
  msg-select    Extract entries from PO/POT file by index range
  stat          Report statistics for a PO file
  team          Show team leader/members
  update        Update XX.po file
  version       Display the version of git-po-helper

Flags:
  -h, --help              help for git-po-helper
  -q, --quiet count       quiet mode
  -v, --verbose count     verbose mode
      --version           Show version

Use "git-po-helper [command] --help" for more information about a command.
```

The `--pot-file` option (way to get latest pot file: 'auto', 'download', 'build', 'no' or filename such as po/git.pot) is available for: `check`, `check-po`, `check-commits`, `check-pot`, `init`, and `update` commands.

## Commands

### Quality checking

| Command | Description |
|---------|-------------|
| `check` | Check all `.po` files and commits. Options: `--core` (also check against git-core.pot), `--force`, `--no-gpg`, `--pot-file`, `--report-file-locations`, `--report-typos`. |
| `check-commits` | Check commits for l10n conventions. Usage: `check-commits [<range>]`. Options: `--force`, `--no-gpg`, `--pot-file`, `--report-file-locations`, `--report-typos`. |
| `check-po` | Check syntax of XX.po file. Usage: `check-po <XX.po>...`. Options: `--core`, `--pot-file`, `--report-file-locations`, `--report-typos`. |
| `check-pot` | Check config variables in POT file. Options: `--pot-file`, `--show-all-configs`, `--show-camel-case-configs` (for config manpage). |

### PO file operations

| Command | Description |
|---------|-------------|
| `compare` | Show changes between two PO files or versions. Default: output new or changed entries to stdout. With `--stat`: show diff statistics. Use `-r`, `--commit`, or `--since` for revision range. Usage: `compare [-r range] [[<src>] <target>]`. |
| `init` | Create XX.po file. Usage: `init <XX.po>`. Options: `--core` (generate from po/git-core.pot), `--pot-file`. |
| `msg-cat` | Concatenate and merge PO/POT/JSON files. Usage: `msg-cat -o <output> [--json] [inputfile]...`. Output to file or stdout (`-o -`). Duplicate msgid: first occurrence by file order wins. |
| `msg-select` | Extract entries from PO/POT file by index range. Usage: `msg-select --range "3,5,9-13" <po-file>`. Range format: `3,5` (entries 3 and 5), `9-13` (entries 9–13), `-5` (first 5), `50-` (from 50 to end). |
| `stat` | Report statistics for a PO file. Usage: `stat <po-file>`. Outputs: translated, untranslated, same (msgstr equals msgid), fuzzy, obsolete. For review JSON report use `agent-run report`. |
| `update` | Update XX.po file. Usage: `update <XX.po>...`. Options: `--no-file-location`, `--no-location`, `--pot-file`. |

### Team and version

| Command | Description |
|---------|-------------|
| `team` | Show team leader/members. Usage: `team [options] [team]...`. Options: `-a` (all users), `-c` (check members), `-L` (language), `-l` (leaders only), `-m` (members only). |
| `version` | Display the version of git-po-helper. |

### Agent commands (AI-assisted translation)

| Command | Description |
|---------|-------------|
| `agent-run` | Run agent commands for automation. Subcommands: `update-pot`, `update-po`, `translate`, `review`, `report`, `parse-log`, `show-config`. Uses git-po-helper.yaml for configuration. Options: `--prompt` (override prompt). For `translate` and `review`: `--use-agent-md` (default), `--use-local-orchestration`, `--batch-size`. |
| `agent-test` | Test agent commands with multiple runs and calculate average scores. Subcommands: `update-pot`, `update-po`, `translate`, `review`, `show-config`. Options: `--runs` (number of runs, default 5), `--dangerously-remove-po-directory`. For `translate` and `review`: `--use-agent-md` (default), `--use-local-orchestration`, `--batch-size`. |

See [docs/agent-commands.md](docs/agent-commands.md) for agent configuration and detailed usage.
