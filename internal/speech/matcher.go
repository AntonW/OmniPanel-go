// Phrase matching and security allowlisting.
// Implements three matching strategies (in order):
//  1. Exact match (case-insensitive)
//  2. Parameterized patterns with {param} placeholders (regex-based)
//  3. Fuzzy matching using Levenshtein distance (0.8 threshold)
//
// The allowlist supports wildcard patterns ("set * to *") to restrict
// which phrases can execute commands.
package speech

import (
	"regexp"
	"strings"
)

var paramRegex = regexp.MustCompile(`\{(\w+)\}`)

// matchPhrase checks if text matches a phrase pattern and extracts parameters.
// Supports exact match and parameterized patterns like "set {value} to {target}".
func matchPhrase(phrase, text string) (bool, map[string]string) {
	phraseLower := strings.ToLower(strings.TrimSpace(phrase))
	textLower := strings.ToLower(strings.TrimSpace(text))

	if phraseLower == textLower {
		return true, nil
	}

	if !strings.Contains(phraseLower, "{") {
		return false, nil
	}

	pattern := buildRegexPattern(phraseLower)
	re, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		return false, nil
	}

	matches := re.FindStringSubmatch(textLower)
	if matches == nil {
		return false, nil
	}

	params := make(map[string]string)
	paramNames := paramRegex.FindAllStringSubmatch(phraseLower, -1)
	for i, name := range paramNames {
		if i+1 < len(matches) {
			params[name[1]] = matches[i+1]
		}
	}

	return true, params
}

// buildRegexPattern converts a phrase with {param} placeholders to a regex.
func buildRegexPattern(phrase string) string {
	escaped := regexp.QuoteMeta(phrase)
	for _, match := range paramRegex.FindAllStringSubmatch(phrase, -1) {
		placeholder := regexp.QuoteMeta(match[0])
		escaped = strings.Replace(escaped, placeholder, `(\S+)`, 1)
	}
	return escaped
}

// fuzzyMatchCommand finds the closest matching command using Levenshtein distance.
// Returns true if similarity exceeds the threshold (0.8).
func fuzzyMatchCommand(text string, commands []SpeechCommand) (bool, SpeechCommand) {
	textLower := strings.ToLower(strings.TrimSpace(text))
	var bestMatch SpeechCommand
	bestScore := 0.0

	for _, cmd := range commands {
		candidates := append([]string{cmd.Phrase}, cmd.Aliases...)
		for _, candidate := range candidates {
			score := similarity(textLower, strings.ToLower(candidate))
			if score > bestScore {
				bestScore = score
				bestMatch = cmd
			}
		}
	}

	return bestScore >= 0.8, bestMatch
}

// similarity computes normalized similarity (0-1) using Levenshtein distance.
func similarity(a, b string) float64 {
	dist := levenshtein(a, b)
	maxLen := float64(len(a))
	if len(b) > len(a) {
		maxLen = float64(len(b))
	}
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(dist)/maxLen
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	v0 := make([]int, len(b)+1)
	v1 := make([]int, len(b)+1)

	for i := 0; i <= len(b); i++ {
		v0[i] = i
	}

	for i := 0; i < len(a); i++ {
		v1[0] = i + 1
		for j := 0; j < len(b); j++ {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			v1[j+1] = min3(v1[j]+1, v0[j+1]+1, v0[j]+cost)
		}
		for j := 0; j <= len(b); j++ {
			v0[j] = v1[j]
		}
	}

	return v1[len(b)]
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// matchAllowlist checks if text matches an allowlist pattern.
// Supports wildcards: "set * to *" matches "set throttle to 50".
func matchAllowlist(pattern, text string) bool {
	patternLower := strings.ToLower(strings.TrimSpace(pattern))
	textLower := strings.ToLower(strings.TrimSpace(text))

	if patternLower == textLower {
		return true
	}

	if !strings.Contains(patternLower, "*") {
		return false
	}

	regexPattern := regexp.QuoteMeta(patternLower)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, `.*`)
	re, err := regexp.Compile("^" + regexPattern + "$")
	if err != nil {
		return false
	}

	return re.MatchString(textLower)
}
