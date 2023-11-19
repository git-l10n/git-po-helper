package dict

func init() {
	SmudgeMaps["es"] = []SmudgeMap{
		{
			Pattern: "p.e.",
			Replace: "e.g.",
		},
		{
			Pattern: "--dirstat=archivos",
			Replace: "--dirstat=files",
		},
	}
}
