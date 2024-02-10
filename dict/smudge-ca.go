package dict

func init() {
	SmudgeMaps["ca"] = []SmudgeMap{
		{
			Pattern: "usuari@domini.com",
			Replace: "you@example.com",
		},
		{
			Pattern: "«",
			Replace: "'",
		},
		{
			Pattern: "»",
			Replace: "'",
		},
	}
}
