package dict

type SmudgeMap struct {
	Pattern interface{}
	Replace string
}

// SmudgeMaps defines replacement map locales
var SmudgeMaps map[string][]SmudgeMap = make(map[string][]SmudgeMap)
