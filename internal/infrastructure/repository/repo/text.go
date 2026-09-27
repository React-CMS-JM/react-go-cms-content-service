package repo

import (
	"regexp"
	"strings"
)

var nonWord = regexp.MustCompile(`[^\w\s-]`)
var spaces = regexp.MustCompile(`[\s_-]+`)
var edgeDash = regexp.MustCompile(`^-+|-+$`)

func slugify(text string) string {
	slug := strings.ToLower(strings.TrimSpace(text))
	slug = nonWord.ReplaceAllString(slug, "")
	slug = spaces.ReplaceAllString(slug, "-")
	return edgeDash.ReplaceAllString(slug, "")
}
