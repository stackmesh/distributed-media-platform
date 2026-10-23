package ports

import "strings"

func lowerTrim(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
