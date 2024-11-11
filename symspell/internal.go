package symspell

import (
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`['’\w-[_]]+`)

// parseWords splits the input text into words.
func parseWords(text string) []string {
	// Compatible with non-latin characters, does not split words at apostrophes
	return re.FindAllString(strings.ToLower(text), -1)
}

func addToSet(set map[string]struct{}, key string) bool {
	if _, found := set[key]; found {
		return false
	}
	set[key] = struct{}{}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// maxInt64 returns the maximum of two int64 numbers.
func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// minInt64 returns the minimum of two int64 numbers.
func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
