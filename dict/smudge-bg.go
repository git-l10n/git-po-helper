package dict

import "regexp"

func init() {
	SmudgeMaps["bg"] = []SmudgeMap{
		/*
		 * In bg translations, additional "git" command name is added before
		 * subcommand, revert these changes before checking typos.
		 */
		{
			Pattern: "Командата „git pack-objects“ не може да бъде стартирана",
			Replace: "Could not spawn pack-objects",
		},
		{
			Pattern: "гарантиращите обекти не може да се подадат на командата „git pack-objects“",
			Replace: "failed to feed promisor objects to pack-objects",
		},
		{
			Pattern: "Командата „git pack-objects“ не записа файл „%s“ за пакета „%s-%s“",
			Replace: "pack-objects did not write a '%s' file for pack %s-%s",
		},
		{
			Pattern: "Командата „git pack-objects“ не завърши успешно",
			Replace: "pack-objects died",
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
			Pattern: "непозната стойност за „--object-format“: „%s“",
			Replace: "unknown value for object-format: %s",
		},
		{
			Pattern: "„--hard“/„--mixed“/„--soft“",
			Replace: "--{hard,mixed,soft}",
		},
		{
			Pattern: "Неправилен режим за „--rebase-merges“: %s",
			Replace: "Unknown rebase-merges mode: %s",
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
		{
			Pattern: "…",
			Replace: "...",
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
		 * Add "<>" markers for param1 and param2
		 */
		{
			Pattern: "files,ПАРАМЕТЪР_1,ПАРАМЕТЪР_2,",
			Replace: "files,<ПАРАМЕТЪР_1>,<ПАРАМЕТЪР_2>",
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
		 * Use the pattern below, we can rewrite "--source=КЛОН" in msgstr to:
		 *
		 *     --source=<...>
		 */
		{
			Pattern: regexp.MustCompile(`(--[a-zA-Z0-9-]+)=([a-zA-Z0-9-]*[^a-zA-Z0-9<>\[\]()%\s"',*-]+[^<>\[\]()%\s"',*]*)`),
			Replace: "$1=<...>",
		},

		/*
		 * The <place-holder>s in refspecs were translated in bg without "<>", e.g.:
		 *
		 *     msgid: "set refs/remotes/<name>/HEAD according to remote"
		 *     msgstr: "задаване на refs/remotes/ИМЕ/HEAD според отдалеченото хранилище"
		 *
		 * After replaced according to patterns defined in GlobalSkipPatterns,
		 * the result are follow:
		 *
		 *     msgid: "set refs/remotes/<...>/HEAD according to remote"
		 *     msgstr: "задаване на refs/remotes/ИМЕ/HEAD според отдалеченото хранилище"
		 *
		 * We will get the keep words as follows from patten defined in KeepWordsPattern.
		 *
		 *     msgid:  refs/remotes/<...>/HEAD
		 *     msgstr: refs/remotes/
		 *
		 * This will cause false positive report of typos. Hack as follows:
		 */
		{
			Pattern: regexp.MustCompile(`/ИМЕ/`),
			Replace: "/<name>/",
		},
	}
}
