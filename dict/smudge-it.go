package dict

func init() {
	SmudgeMaps["it"] = []SmudgeMap{
		{
			Pattern: "tu@esempio.com",
			Replace: "you@example.com",
		},
	}
}
