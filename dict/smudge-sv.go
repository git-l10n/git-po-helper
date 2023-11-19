package dict

import "regexp"

func init() {
	SmudgeMaps["sv"] = []SmudgeMap{
		{
			Pattern: "t.ex",
			Replace: "e.g.",
		},
		{
			Pattern: "ref.spec-en",
			Replace: "refspec",
		},
		{
			Pattern: "--dirstat=filer",
			Replace: "--dirstat=files",
		},
		{
			Pattern: "git-branch-åtgärder",
			Replace: "git-branch actions",
		},
		{
			Pattern: "git-kommand",
			Replace: "git command",
		},
		{
			Pattern: "git-diff-huvudet",
			Replace: "git diff header",
		},
		{
			Pattern: "git-katalog",
			Replace: "git dir",
		},
		{
			Pattern: "git-huvudet",
			Replace: "git header",
		},
		{
			Pattern: "git-process",
			Replace: "git process",
		},
		{
			Pattern: "-läge",
			Replace: " mode",
		},
		{
			Pattern: "-krokar",
			Replace: " hooks",
		},
		{
			Pattern: "git-arkivversion",
			Replace: "git repo version",
		},
		{
			Pattern: regexp.MustCompile(`\bgit-attribut\b`),
			Replace: "git attribute",
		},
		{
			Pattern: regexp.MustCompile(`\bgit-(fil|filen)\b`),
			Replace: "git file",
		},
		{
			Pattern: regexp.MustCompile(`\bgit-(arkiv|arkivet)\b`),
			Replace: "git repository",
		},
	}
}
