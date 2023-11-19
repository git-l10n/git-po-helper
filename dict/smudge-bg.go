package dict

import "regexp"

func init() {
	SmudgeMaps["bg"] = []SmudgeMap{
		/*
		 * In bg translations, additional "git" command name is added before
		 * subcommand, revert these changes before checking typos.
		 */
		{
			Pattern: "Командата „git pack-objects“",
			Replace: "spawn pack-objects",
		},
		{
			Pattern: "„git-difftool“ изисква работно дърво или опцията „--no-index“",
			Replace: "difftool requires worktree or --no-index",
		},
		{
			Pattern: "командата „git index-pack“ не завърши успешно",
			Replace: "index-pack died",
		},
		{
			Pattern: "Командата „git pack-objects“ не завърши успешно",
			Replace: "pack-objects died",
		},
		{
			Pattern: "указателят „%s“ не е бил включен поради опциите зададени на „git rev-list“",
			Replace: "ref '%s' is excluded by the rev-list options",
		},
		{
			Pattern: "    git merge-base --fork-point",
			Replace: "    merge-base --fork-point",
		},

		/*
		 * In bg translations, add additional "--" characters before command option,
		 * but the original msgid does not have.
		 */
		{
			Pattern: "неправилен параметър към опцията „--update“",
			Replace: "bad value for update parameter",
		},
		{
			Pattern: "включва опцията „--bare“ за голо хранилище",
			Replace: "implies bare",
		},
		{
			Pattern: "„--hard“/„--mixed“/„--soft“",
			Replace: "--{hard,mixed,soft}",
		},
		{
			Pattern: "„%s“ към опцията „--ancestry-path",
			Replace: "ancestry-path argument %s",
		},
		{
			Pattern: "Неправилен режим за „--rebase-merges“: %s",
			Replace: "Unknown rebase-merges mode: %s",
		},
		{
			Pattern: "не поддържа опцията „--force“",
			Replace: "does not support 'force'",
		},
		{
			Pattern: "неправилна стойност за „--mirror“: %s",
			Replace: "unknown mirror argument: %s",
		},

		// Revert changes in bg, such as quotes and dashes.
		{
			Pattern: "„",
			Replace: "\"",
		},
		{
			Pattern: "“",
			Replace: "\"",
		},
		{
			Pattern: "———",
			Replace: "---",
		},

		// Revert translated email address
		{
			Pattern: "ИМЕ@example.com",
			Replace: "you@example.com",
		},
		{
			Pattern: "пенчо@example.com",
			Replace: "you@example.com",
		},

		/*
		 * The <place-holder> in format string was translated without "<>", e.g.:
		 *
		 *     msgid "expected format: %%(color:<color>)"
		 *     msgstr "очакван формат: %%(color:ЦВЯТ)"
		 *
		 *     msgid "expected format: %%(align:<width>,<position>)"
		 *     msgstr "очакван формат: %%(align:ШИРОЧИНА,ПОЗИЦИЯ)"
		 *
		 * After replaced according to patterns defined in GlobalSkipPatterns,
		 * the result are follow:
		 *
		 *     msgid "expected format: %%(color:<...>)"
		 *     msgstr "очакван формат: %%(color:ЦВЯТ)"
		 *
		 *     msgid "expected format: %%(align:<...>,<...>)"
		 *     msgstr "очакван формат: %%(align:ШИРОЧИНА,ПОЗИЦИЯ)"
		 *
		 * In oder to check the format strings in above messages, we define
		 * several smudge maps as below.
		 */
		{
			Pattern: regexp.MustCompile(`(%%\([^\s\)]+?:([^\s\)]*?=)?).*?\)`),
			Replace: "$1)",
			Reverse: true,
		},
		{
			Pattern: regexp.MustCompile(`(%%\([^\s\)]+?:([^\s\)]*?=)?).*?\)`),
			Replace: "$1)",
		},

		/*
		 * The <place-holder>s in command options were translated in bg without "<>", e.g.:
		 *
		 *     msgid "git restore [<options>] [--source=<branch>] <file>..."
		 *     msgstr "git restore [ОПЦИЯ…] [--source=КЛОН] ФАЙЛ…"
		 *
		 * After replaced according to patterns defined in GlobalSkipPatterns,
		 * the result are follow:
		 *
		 *     msgid "git restore [<...>] [--source=<...>] <file>..."
		 *     msgstr "git restore [ОПЦИЯ…] [--source=КЛОН] ФАЙЛ…"
		 *
		 * We will get the keep words as follows from patten defined in KeepWordsPattern.
		 *
		 *     msgid:  --source=
		 *     msgstr: --source=K
		 *
		 * This will cause false positive report of typos. Hack as follows:
		 */
		{
			Pattern: regexp.MustCompile(`(--[^\s]+=)<\.\.\.>`),
			Replace: "$1 ...",
			Reverse: true,
		},
		{
			Pattern: regexp.MustCompile(`(--[^\s]+=)([a-zA-Z-]*[^a-zA-Z0-9\s-"]+[a-zA-Z0-9-]*)`),
			Replace: "$1 $2",
		},
	}
}
