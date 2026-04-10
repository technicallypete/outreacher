package tools

import (
	"regexp"
	"strings"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a human-readable name into a lowercase, hyphen-separated
// slug suitable for use as a stable identifier (e.g. "Foo Bar" → "foo-bar").
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
