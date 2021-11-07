package dict

func init() {
	SmudgeMaps["bg"] = map[interface{}]string{
		"———": "---",

		// Not keep symbols "<" and ">", the reason is ?
		"РЕГУЛЯРЕН_ИЗРАЗ":        "<РЕГУЛЯРЕН_ИЗРАЗ>",
		"ДИРЕКТОРИЯ":             "<ДИРЕКТОРИЯ>",
		"ПАКЕТЕН_ФАЙЛ":           "<ПАКЕТЕН_ФАЙЛ>",
		"align:ШИРОЧИНА,ПОЗИЦИЯ": "align:<ШИРОЧИНА>,<ПОЗИЦИЯ>",
		"color:ЦВЯТ":             "color:<ЦВЯТ>",
		"--config=НАСТРОЙКА":     "--config=<НАСТРОЙКА>",
		"--prefix=ПРЕФИКС":       "--prefix=<ПРЕФИКС>",
		"--index-output=ФАЙЛ":    "--index-output=<ФАЙЛ>",
		"--extcmd=КОМАНДА":       "--extcmd=<КОМАНДА>",
		"--tool=ПРОГРАМА":        "--tool=<ПРОГРАМА>",
		"--schedule=ЧЕСТОТА":     "--schedule=<ЧЕСТОТА>",
		// Upstream may need to add "<>" around "files"
		"--dirstat=ФАЙЛОВЕ": "--dirstat=files",
		"--dirstat=ФАЙЛ…,ПАРАМЕТЪР_1,ПАРАМЕТЪР_2,": "--dirstat=files,param1,param2",

		// Email address
		"ИМЕ@example.com":   "you@example.com",
		"пенчо@example.com": "you@example.com",

		// add or lost '--'
		"неправилен параметър към опцията „--update“": "bad value for update parameter",
		"включва опцията „--bare“ за голо хранилище":  "implies bare",
		"„--hard“/„--mixed“/„--soft“":                 "--{hard,mixed,soft}",
		// Upstream should add "--" ?
		"не поддържа опцията „--force“":         "does not support 'force'",
		"неправилна стойност за „--mirror“: %s": "unknown mirror argument: %s",

		// Add or lost "git" before subcommand
		"Командата „git pack-objects“":                                               "spawn pack-objects",
		"„git-difftool“ изисква работно дърво или опцията „--no-index“":              "difftool requires worktree or --no-index",
		"командата „git index-pack“ не завърши успешно":                              "index-pack died",
		"Командата „git pack-objects“ не завърши успешно":                            "pack-objects died",
		"указателят „%s“ не е бил включен поради опциите зададени на „git rev-list“": "ref '%s' is excluded by the rev-list options",

		// Typos ?
		"зареждане на разширенията на индекса":  "load_index_extensions",
		"зареждане на обектите от кеша":         "load_cache_entries",
		"новият индекс не може да бъде записан": "unable to write new_index file",
	}
}
