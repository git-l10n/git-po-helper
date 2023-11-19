package dict

func init() {
	SmudgeMaps["bg"] = []SmudgeMap{
		{
			Pattern: "———",
			Replace: "---",
		},

		// Not keep symbols "<" and ">", the reason is ?
		{
			Pattern: "РЕГУЛЯРЕН_ИЗРАЗ",
			Replace: "<РЕГУЛЯРЕН_ИЗРАЗ>",
		},
		{
			Pattern: "ДИРЕКТОРИЯ",
			Replace: "<ДИРЕКТОРИЯ>",
		},
		{
			Pattern: "ПАКЕТЕН_ФАЙЛ",
			Replace: "<ПАКЕТЕН_ФАЙЛ>",
		},
		{
			Pattern: "align:ШИРОЧИНА,ПОЗИЦИЯ",
			Replace: "align:<ШИРОЧИНА>,<ПОЗИЦИЯ>",
		},
		{
			Pattern: "color:ЦВЯТ",
			Replace: "color:<ЦВЯТ>",
		},
		{
			Pattern: "--config=НАСТРОЙКА",
			Replace: "--config=<НАСТРОЙКА>",
		},
		{
			Pattern: "--prefix=ПРЕФИКС",
			Replace: "--prefix=<ПРЕФИКС>",
		},
		{
			Pattern: "--index-output=ФАЙЛ",
			Replace: "--index-output=<ФАЙЛ>",
		},
		{
			Pattern: "--extcmd=КОМАНДА",
			Replace: "--extcmd=<КОМАНДА>",
		},
		{
			Pattern: "--tool=ПРОГРАМА",
			Replace: "--tool=<ПРОГРАМА>",
		},
		{
			Pattern: "--schedule=ЧЕСТОТА",
			Replace: "--schedule=<ЧЕСТОТА>",
		},
		{
			Pattern: "trailers:key=ЕПИЛОГ",
			Replace: "trailers:key=<ЕПИЛОГ>",
		},

		// Upstream may need to add "<>" around "files"
		{
			Pattern: "--dirstat=ФАЙЛОВЕ",
			Replace: "--dirstat=files",
		},
		{
			Pattern: "--dirstat=ФАЙЛ…,ПАРАМЕТЪР_1,ПАРАМЕТЪР_2,",
			Replace: "--dirstat=files,param1,param2",
		},

		// Email address
		{
			Pattern: "ИМЕ@example.com",
			Replace: "you@example.com",
		},
		{
			Pattern: "пенчо@example.com",
			Replace: "you@example.com",
		},

		// add or lost '--'
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

		// Upstream should add "--" ?
		{
			Pattern: "не поддържа опцията „--force“",
			Replace: "does not support 'force'",
		},
		{
			Pattern: "неправилна стойност за „--mirror“: %s",
			Replace: "unknown mirror argument: %s",
		},

		// Add or lost "git" before subcommand
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

		// Quotes in bg
		{
			Pattern: "„",
			Replace: "\"",
		},
		{
			Pattern: "“",
			Replace: "\"",
		},
	}
}
