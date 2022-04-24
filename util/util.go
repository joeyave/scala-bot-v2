package util

import (
	"regexp"
	"strings"
)

func CleanUpText(text string) string {
	numbersRegex := regexp.MustCompile(`\(.*?\)|[1-9.()_]*`)
	return numbersRegex.ReplaceAllString(text, "")
}

var newLinesRegex = regexp.MustCompile(`\s*[\t\r\n]+`)

func SplitTextByNewlines(query string) []string {
	songNames := strings.Split(newLinesRegex.ReplaceAllString(query, "\n"), "\n")
	for _, songName := range songNames {
		songName = strings.TrimSpace(songName)
	}

	return songNames
}

func IetfToIsoLangCode(languageCode string) string {
	switch languageCode {
	default:
		return "ru_RU"
	}
}
