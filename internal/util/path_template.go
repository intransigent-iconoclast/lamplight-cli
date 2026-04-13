package utils

import (
	"path/filepath"
	"strings"
)

// ApplyTemplate resolves a template string like "{author}/{title} ({year})"
// using the provided metadata. Returns a relative path with sanitized components.
func ApplyTemplate(tmpl string, meta *BookMetadata) string {
	replacements := map[string]string{
		"{author}":    sanitizePathComponent(meta.Author),
		"{title}":     sanitizePathComponent(meta.Title),
		"{year}":      sanitizePathComponent(meta.Year),
		"{publisher}": sanitizePathComponent(meta.Publisher),
		"{isbn}":      sanitizePathComponent(meta.ISBN),
		"{format}":    sanitizePathComponent(meta.Format),
	}

	result := tmpl
	for token, value := range replacements {
		result = strings.ReplaceAll(result, token, value)
	}

	// clean up any double slashes, trailing slashes, etc.
	return filepath.Clean(result)
}

// sanitizePathComponent makes a string safe to use as a single path segment.
// Strips chars that are illegal on common filesystems, collapses whitespace.
func sanitizePathComponent(s string) string {
	if s == "" {
		return "Unknown"
	}

	// chars that are either illegal or just a pain across OSes
	bad := `/\:*?"<>|`
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(bad, r) {
			b.WriteRune('-')
		} else {
			b.WriteRune(r)
		}
	}

	result := strings.Join(strings.Fields(b.String()), " ")
	result = strings.Trim(result, ". ")

	// don't let any single segment get insane — 80 chars is plenty
	if len(result) > 80 {
		result = result[:80]
		if i := strings.LastIndex(result, " "); i > 40 {
			result = result[:i]
		}
	}

	if result == "" {
		return "Unknown"
	}
	return result
}

// IsComplete returns true if the metadata has enough to place the file in
// the main library (author + title at minimum).
func IsComplete(meta *BookMetadata) bool {
	return meta.Author != "" && meta.Author != "Unknown" &&
		meta.Title != "" && meta.Title != "Unknown"
}
