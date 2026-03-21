// Package data provides ISO 639, ISO 3166, and ISO 15924 data for locales.
package data

import "strings"

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

// GetLocationNameInsensitive looks up zone case-insensitively in ISO 3166.
// Returns (name, canonicalKey) if found. Canonical is uppercase.
func GetLocationNameInsensitive(zone string) (name, canonical string) {
	key := strings.ToUpper(zone)
	if n := locationMap[key]; n != "" {
		return n, key
	}
	return "", ""
}

// GetScriptNameInsensitive looks up zone case-insensitively in ISO 15924.
// Returns (name, canonicalKey) if found.
func GetScriptNameInsensitive(zone string) (name, canonical string) {
	zoneLower := strings.ToLower(zone)
	for k, v := range scriptMap {
		if strings.ToLower(k) == zoneLower {
			return v, k
		}
	}
	return "", ""
}
