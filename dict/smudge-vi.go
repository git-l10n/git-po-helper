package dict

func init() {
	SmudgeMaps["vi"] = []SmudgeMap{
		{
			Pattern: "v.d.",
			Replace: "e.g.",
		},
		{
			Pattern: "v.v.",
			Replace: "etc.",
		},
		{
			Pattern: "bạn@ví_dụ.com",
			Replace: "you@example.com",
		},
	}
}
