package health

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var terminalSequence = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\))`)
var credentialValue = regexp.MustCompile(`(?i)(bearer\s+\S+|eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+|-----BEGIN[^-]*PRIVATE KEY|[a-z][a-z0-9+.-]*://[^\s/?#]*@|[?&]sig=|(?:AccountKey|SharedAccessKey|Password|ClientSecret|access_token|refresh_token)\s*=)`)

// SafeText filters the accepted text fields, never arbitrary source payloads.
func SafeText(value string) (string, int) {
	value = terminalSequence.ReplaceAllString(value, "")
	credential := credentialValue.MatchString(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, value)
	if credential || credentialValue.MatchString(value) {
		return "[REDACTED]", 1
	}
	return value, 0
}

// SafeTags keeps ordinary context and redacts credential values even when the
// value itself has no recognizable token prefix. Sort keys before normalization
// so collisions after control stripping cannot overwrite values nondeterministically.
func SafeTags(tags map[string]*string) (map[string]string, int) {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result, count := map[string]string{}, 0
	for _, original := range keys {
		key, n := SafeText(original)
		count += n
		normalized := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, key)
		value := ""
		if tags[original] != nil {
			value = *tags[original]
		}
		secret := false
		for _, pattern := range []string{"password", "secret", "accesstoken", "refreshtoken", "authorization", "apikey", "accountkey", "connectionstring", "privatekey"} {
			if strings.Contains(normalized, pattern) {
				secret = true
			}
		}
		if strings.HasPrefix(normalized, "sas") || strings.HasSuffix(normalized, "sas") || strings.Contains(normalized, "sastoken") || strings.Contains(normalized, "sassignature") || strings.Contains(normalized, "sharedaccesssignature") {
			secret = true
		}
		if secret {
			value = "[REDACTED]"
			count++
		} else {
			value, n = SafeText(value)
			count += n
		}
		if _, exists := result[key]; exists {
			result[key] = "[REDACTED]"
			count++
		} else {
			result[key] = value
		}
	}
	return result, count
}
