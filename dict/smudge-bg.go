package dict

func init() {
	SmudgeMaps["bg"] = []SmudgeMap{
		{"———", "---"},

		// Not keep symbols "<" and ">" - the reason provided by the translator is:
		// Bulgarian uses different alphabet than English, thus it is obvious that
		// things in Cyrillic that are uppercase are to be substituted, so no need for <>.
		// Omitting the <> makes the strings shorter which is good, esp. for narrower terminals.
		// <> are dangerous when pasted on the command line so it is better to omit them.
		// In some messages they make them hard to read, ex.:
		//   commit-graph generation for commit %s is %<PRIuMAX> < %<PRIuMAX>
		//   git pack-objects [<options>] <base-name> [< <ref-list> | < <object-list>]
		//   git mailinfo [<options>] <msg> <patch> < mail >info
		{"РЕГУЛЯРЕН_ИЗРАЗ", "<РЕГУЛЯРЕН_ИЗРАЗ>"},
		{"ДИРЕКТОРИЯ", "<ДИРЕКТОРИЯ>"},
		{"ПАКЕТЕН_ФАЙЛ", "<ПАКЕТЕН_ФАЙЛ>"},
		{"align:ШИРОЧИНА,ПОЗИЦИЯ", "align:<ШИРОЧИНА>,<ПОЗИЦИЯ>"},
		{"color:ЦВЯТ", "color:<ЦВЯТ>"},
		{"--config=НАСТРОЙКА", "--config=<НАСТРОЙКА>"},
		{"--prefix=ПРЕФИКС", "--prefix=<ПРЕФИКС>"},
		{"--index-output=ФАЙЛ", "--index-output=<ФАЙЛ>"},
		{"--extcmd=КОМАНДА", "--extcmd=<КОМАНДА>"},
		{"--tool=ПРОГРАМА", "--tool=<ПРОГРАМА>"},
		{"--schedule=ЧЕСТОТА", "--schedule=<ЧЕСТОТА>"},
		{"trailers:key=ЕПИЛОГ", "trailers:key=<ЕПИЛОГ>"},
		{"ПОДАВАНЕ", "<committish>"},

		// Upstream may need to add "<>" around "files"
		{"--dirstat=ФАЙЛОВЕ", "--dirstat=files"},
		{"--dirstat=ФАЙЛ…,ПАРАМЕТЪР_1,ПАРАМЕТЪР_2,", "--dirstat=files,param1,param2"},

		// Email address
		{"ИМЕ@example.com", "you@example.com"},
		{"пенчо@example.com", "you@example.com"},

		// add or lost '--'
		{"неправилен параметър към опцията „--update“", "bad value for update parameter"},
		{"включва опцията „--bare“ за голо хранилище", "implies bare"},
		{"„--hard“/„--mixed“/„--soft“", "--{hard,mixed,soft}"},
		{"„%s“ към опцията „--ancestry-path", "ancestry-path argument %s"},

		// Upstream should add "--" ?
		{"не поддържа опцията „--force“", "does not support 'force'"},
		{"неправилна стойност за „--mirror“: %s", "unknown mirror argument: %s"},
		{"Неправилен режим за „--rebase-merges“: %s", "Unknown rebase-merges mode: %s"},

		// Add or lost "git" before subcommand
		{"Командата „git pack-objects“", "spawn pack-objects"},
		{"„git-difftool“ изисква работно дърво или опцията „--no-index“", "difftool requires worktree or --no-index"},
		{"командата „git index-pack“ не завърши успешно", "index-pack died"},
		{"Командата „git pack-objects“ не завърши успешно", "pack-objects died"},
		{"указателят „%s“ не е бил включен поради опциите зададени на „git rev-list“", "ref '%s' is excluded by the rev-list options"},
		{"    git merge-base --fork-point", "    merge-base --fork-point"},

		// Quotes in bg
		{"„", "\""},
		{"“", "\""},
	}
}
