package dict

func init() {
	SmudgeMaps["bg"] = map[interface{}]string{
		"новият индекс": "new_index",
		"зареждане на разширенията на индекса": "load_index_extensions",
		"зареждане на обектите от кеша":        "load_cache_entries",
		"--dirstat=ФАЙЛОВЕ":                    "--dirstat=files",
	}
}
