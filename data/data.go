// Package data provides ISO 639, ISO 3166, and ISO 15924 data for locales.
package data

var (
	langMap     map[string]string
	locationMap map[string]string
	scriptMap   map[string]string
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

// GetScriptName looks up iso-15924 table and returns script name
func GetScriptName(script string) string {
	return scriptMap[script]
}
