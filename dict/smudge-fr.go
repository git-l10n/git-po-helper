package dict

func init() {
	SmudgeMaps["fr"] = []SmudgeMap{
		{
			Pattern: "vous@exemple.com",
			Replace: "you@example.com",
		},
		{
			Pattern: "Vous@exemple.com",
			Replace: "you@example.com",
		},
	}
}
