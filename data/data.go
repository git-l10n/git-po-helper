package data

var (
	langMap     map[string]string
	locationMap map[string]string
)

//go:generate go run github.com/git-l10n/git-po-helper/data/main

// GetLanguageName looks up iso-639 table and returns language name
func GetLanguageName(lang string) string {
	return langMap[lang]
}

// GetLocationName looks up iso-3166 table and returns location name
func GetLocationName(lang string) string {
	return locationMap[lang]
}
