# Chapter 11: Phrase Matching & Security

## What This Chapter Covers

After the STT engine transcribes audio to text, OmniPanel-go needs to figure out what command (if any) the user intended. The `matcher.go` file handles this with three matching strategies and an optional security allowlist.

## Exact Match

The simplest case: the transcribed text exactly matches a command phrase.

```go
// internal/speech/matcher.go
func matchPhrase(phrase, text string) (bool, map[string]string) {
    phraseLower := strings.ToLower(strings.TrimSpace(phrase))
    textLower := strings.ToLower(strings.TrimSpace(text))

    if phraseLower == textLower {
        return true, nil
    }
    // ... more strategies
}
```

> **Concept: Case-insensitive comparison**
> Voice recognition is imperfect — capitalization is unpredictable. Converting both strings to lowercase before comparison ensures "Gear Up", "gear up", and "GEAR UP" all match.

## Parameterized Patterns

Commands can include `{param}` placeholders that capture values from the spoken text:

```go
// Phrase: "set throttle to {value}"
// Spoken: "set throttle to 75"
// Result: matched=true, params={"value": "75"}
```

The matcher builds a regex from the phrase pattern:

```go
func buildRegexPattern(phrase string) string {
    escaped := regexp.QuoteMeta(phrase)
    for _, match := range paramRegex.FindAllStringSubmatch(phrase, -1) {
        placeholder := regexp.QuoteMeta(match[0])
        escaped = strings.Replace(escaped, placeholder, `(\S+)`, 1)
    }
    return escaped
}
```

> **Concept: `regexp.QuoteMeta`**
> This escapes all regex special characters in a string. If your phrase is "set {value} to {target}", `QuoteMeta` produces `set\ \{value\}\ to\ \{target\}`. Then we replace `\{value\}` with `(\S+)` (a capture group for non-whitespace characters). This prevents user-defined phrases from accidentally containing regex syntax.

The regex is then compiled and matched:

```go
pattern := buildRegexPattern(phraseLower)
re, _ := regexp.Compile("^" + pattern + "$")
matches := re.FindStringSubmatch(textLower)

if matches != nil {
    params := make(map[string]string)
    paramNames := paramRegex.FindAllStringSubmatch(phraseLower, -1)
    for i, name := range paramNames {
        if i+1 < len(matches) {
            params[name[1]] = matches[i+1]
        }
    }
    return true, params
}
```

> **Key Pattern: Named capture groups via parallel arrays**
> Go's `regexp` doesn't support named capture groups like Python's `(?P<name>...)`. Instead, we extract parameter names from the original phrase (`["{value}", "{target}"]`) and match them to the regex capture groups by index. This is a common workaround in Go.

## Fuzzy Matching

When no exact or parameterized match is found, the matcher falls back to fuzzy matching using Levenshtein distance:

```go
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
```

The similarity score is computed as `1 - (edit_distance / max_length)`:

```go
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
```

> **Concept: Levenshtein distance**
> The edit distance between two strings is the minimum number of single-character insertions, deletions, or substitutions needed to transform one into the other. "gear up" vs "gear up" = 0 (identical). "gear up" vs "gear down" = 3 (up → down). The implementation uses the classic dynamic programming approach with two rows.

A threshold of 0.8 (80% similarity) balances false positives and false negatives. "gear up" matching "gear down" (similarity ~0.5) would be rejected, while "gear up" matching "gearup" (similarity ~0.875) would be accepted.

## Allowlist Security

When `speech_allowlist` is configured, only matching phrases are allowed to execute:

```go
func (sm *SpeechManager) isAllowed(text string) bool {
    if len(sm.config.SpeechAllowlist) == 0 {
        return true
    }

    for _, pattern := range sm.config.SpeechAllowlist {
        if matchAllowlist(pattern, text) {
            return true
        }
    }
    return false
}
```

The allowlist supports wildcards:

```go
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
    re, _ := regexp.Compile("^" + regexPattern + "$")
    return re.MatchString(textLower)
}
```

> **Key Pattern: Wildcard to regex conversion**
> `"set * to *"` becomes `^set\ .*\\ to\ .*$` after escaping and replacing `\*` with `.*`. This allows flexible patterns while keeping the allowlist simple for users to write.

Example allowlist:
```json
{
  "speech_allowlist": [
    "gear up",
    "gear down",
    "set * to *"
  ]
}
```

This allows "gear up", "gear down", and any phrase matching "set X to Y" pattern. Everything else is blocked.

## The Match-and-Execute Pipeline

```go
func (sm *SpeechManager) matchAndExecute(text string) (bool, string) {
    if !sm.isAllowed(text) {
        slog.Warn("Speech command blocked by allowlist", "text", text)
        return false, ""
    }

    // 1. Try exact match for each command and alias
    for _, cmd := range sm.commands {
        if matched, params := matchPhrase(cmd.Phrase, text); matched {
            return sm.executeCommand(cmd, params)
        }
        for _, alias := range cmd.Aliases {
            if matched, params := matchPhrase(alias, text); matched {
                return sm.executeCommand(cmd, params)
            }
        }
    }

    // 2. Fall back to fuzzy matching
    if fuzzyMatch, cmd := fuzzyMatchCommand(text, sm.commands); fuzzyMatch {
        return sm.executeCommand(cmd, nil)
    }

    return false, ""
}
```

The pipeline is: allowlist check → exact match → parameterized match → fuzzy match → execute.

## Key Takeaways

- Three matching strategies: exact, parameterized (regex), and fuzzy (Levenshtein)
- Parameterized commands use `{param}` placeholders captured via regex
- Fuzzy matching with 0.8 threshold handles voice recognition variance
- Allowlist provides security by restricting which phrases can execute
- Wildcards (`*`) in allowlist patterns convert to regex `.*`
- The match pipeline is ordered from most specific to most lenient

[← Back: Chapter 10](10-speech-engines.md) · [Next: Chapter 12 →](12-audio-recording.md)
