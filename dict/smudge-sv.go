package dict

import "regexp"

func init() {
	SmudgeMaps["sv"] = map[interface{}]string{
		"--dirstat=filer":     "--dirstat=files",
		"git-branch-åtgärder": "git-branch actions",
		"git-kommand":         "git command",
		"git-diff-huvudet":    "git diff header",
		"git-katalog":         "git dir",
		"git-huvudet":         "git header",
		"git-process":         "git process",
		"git-arkivversion":    "git repo version",
		"ref.spec-en":         "refspec",

		regexp.MustCompile(`\bgit-attribut\b`):        "git attribute",
		regexp.MustCompile(`\bgit-(fil|filen)\b`):     "git file",
		regexp.MustCompile(`\bgit-(arkiv|arkivet)\b`): "git repository",
	}
}
