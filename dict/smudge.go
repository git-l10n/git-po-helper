package dict

type SmudgeMap struct {
	Pattern interface{}
	Replace string
	// If reverse is true, match and replace msgId instead.
	Reverse bool
}

// SmudgeMaps defines replacement map locales
var SmudgeMaps map[string][]SmudgeMap = make(map[string][]SmudgeMap)
