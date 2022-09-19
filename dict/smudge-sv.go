package dict

import "regexp"

func init() {
	SmudgeMaps["sv"] = []SmudgeMap{
		{"t.ex", "e.g."},
		{"ref.spec-en", "refspec"},
		{"--dirstat=filer", "--dirstat=files"},
		{"git-branch-åtgärder", "git-branch actions"},
		{"git-kommand", "git command"},
		{"git-diff-huvudet", "git diff header"},
		{"git-katalog", "git dir"},
		{"git-huvudet", "git header"},
		{"git-process", "git process"},
		{"git-arkivversion", "git repo version"},
		{regexp.MustCompile(`\bgit-attribut\b`), "git attribute"},
		{regexp.MustCompile(`\bgit-(fil|filen)\b`), "git file"},
		{regexp.MustCompile(`\bgit-(arkiv|arkivet)\b`), "git repository"},
	}
}
