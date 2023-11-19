package dict

func init() {
	SmudgeMaps["de"] = []SmudgeMap{
		{
			Pattern: "z.B.",
			Replace: "e.g.",
		},
		{
			Pattern: "ihre@emailadresse.de",
			Replace: "you@example.com",
		},
	}
}
