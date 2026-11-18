package analyzer

import (
	"strings"
)

// ExtractPrefix splits a Redis key by any character in delimiters up to maxDepth segments.
// For example, ExtractPrefix("session:auth:user:9912", ":", 2) returns "session:auth:*".
// If a key has fewer segments than maxDepth, it masks the trailing identifier if multiple segments exist,
// or returns "<no-prefix>" if no delimiter is found.
func ExtractPrefix(key string, delimiters string, maxDepth int) string {
	if key == "" {
		return "<empty>"
	}
	if maxDepth <= 0 {
		maxDepth = 1
	}
	if delimiters == "" {
		delimiters = ":"
	}

	segments := splitByDelimiters(key, delimiters)
	if len(segments) <= 1 {
		return "<no-prefix>"
	}

	limit := maxDepth
	if len(segments)-1 < limit {
		limit = len(segments) - 1
	}

	prefix := strings.Join(segments[:limit], string(delimiters[0]))
	return prefix + string(delimiters[0]) + "*"
}

func splitByDelimiters(s string, delimiters string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(delimiters, s[i]) != -1 {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}
