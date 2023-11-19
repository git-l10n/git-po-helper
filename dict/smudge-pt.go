package dict

func init() {
	SmudgeMaps["pt_PT"] = []SmudgeMap{
		{
			Pattern: "p.e.",
			Replace: "e.g.",
		},
		{
			Pattern: "eu@exemplo.com",
			Replace: "you@example.com",
		},
		{
			Pattern: "utilizador@exemplo.com",
			Replace: "you@example.com",
		},
	}
}
