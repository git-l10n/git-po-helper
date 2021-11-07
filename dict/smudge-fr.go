package dict

func init() {
	SmudgeMaps["fr"] = map[interface{}]string{
		"vous@exemple.com": "you@example.com",
		"Vous@exemple.com": "you@example.com",
	}
}
