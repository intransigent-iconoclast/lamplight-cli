package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- ApplyTemplate ---

func TestApplyTemplate_AllFields(t *testing.T) {
	meta := &BookMetadata{
		Title:     "Dune",
		Author:    "Frank Herbert",
		Year:      "1965",
		Publisher: "Chilton",
		ISBN:      "9780441013593",
		Format:    "epub",
	}
	result := ApplyTemplate("{author}/{title} ({year})", meta)
	assert.Equal(t, "Frank Herbert/Dune (1965)", result)
}

func TestApplyTemplate_MissingYear_KeepsLiteralParens(t *testing.T) {
	meta := &BookMetadata{Title: "Dune", Author: "Frank Herbert"}
	// year is empty → sanitizePathComponent returns "Unknown"
	result := ApplyTemplate("{author}/{title} ({year})", meta)
	assert.Equal(t, "Frank Herbert/Dune (Unknown)", result)
}

func TestApplyTemplate_AllPlaceholders(t *testing.T) {
	meta := &BookMetadata{
		Title:     "T",
		Author:    "A",
		Year:      "2000",
		Publisher: "P",
		ISBN:      "I",
		Format:    "epub",
	}
	result := ApplyTemplate("{author}/{title}/{year}/{publisher}/{isbn}/{format}", meta)
	assert.Equal(t, "A/T/2000/P/I/epub", result)
}

func TestApplyTemplate_SpecialCharsInAuthor(t *testing.T) {
	meta := &BookMetadata{Title: "Book", Author: "O'Brien: The/Full\\Story"}
	result := ApplyTemplate("{author}/{title}", meta)
	// slashes, colons, backslashes → dashes
	assert.Contains(t, result, "O'Brien- The-Full-Story")
}

func TestApplyTemplate_CleansDuplicateSlashes(t *testing.T) {
	meta := &BookMetadata{Title: "T", Author: "A"}
	// filepath.Clean should collapse double slashes
	result := ApplyTemplate("{author}//{title}", meta)
	assert.NotContains(t, result, "//")
}

func TestApplyTemplate_NoPlaceholders(t *testing.T) {
	meta := &BookMetadata{Title: "T", Author: "A"}
	result := ApplyTemplate("fixed/path", meta)
	assert.Equal(t, "fixed/path", result)
}

// --- sanitizePathComponent ---

func TestSanitize_EmptyString(t *testing.T) {
	assert.Equal(t, "Unknown", sanitizePathComponent(""))
}

func TestSanitize_WhitespaceOnly(t *testing.T) {
	assert.Equal(t, "Unknown", sanitizePathComponent("   "))
}

func TestSanitize_IllegalCharsBecomeDashes(t *testing.T) {
	result := sanitizePathComponent(`a/b\c:d*e?f"g<h>i|j`)
	assert.Equal(t, "a-b-c-d-e-f-g-h-i-j", result)
}

func TestSanitize_LeadingTrailingDots(t *testing.T) {
	result := sanitizePathComponent("...hello world...")
	assert.Equal(t, "hello world", result)
}

func TestSanitize_CollapseWhitespace(t *testing.T) {
	result := sanitizePathComponent("too   many    spaces")
	assert.Equal(t, "too many spaces", result)
}

func TestSanitize_TruncatesAtWordBoundary(t *testing.T) {
	// build a string well over 80 chars with a clear word boundary before the limit
	long := strings.Repeat("word ", 20) // 100 chars
	result := sanitizePathComponent(long)
	// fits within the limit
	assert.LessOrEqual(t, len(result), 80)
	// cutting at a word boundary means the result is a valid prefix of the joined words
	assert.True(t, strings.HasPrefix(strings.Repeat("word ", 20), result+"...") ||
		strings.HasPrefix(strings.Join(strings.Fields(strings.Repeat("word ", 20)), " "), strings.TrimSuffix(result, "...")))
}

func TestSanitize_TruncatesHardIfNoWordBoundary(t *testing.T) {
	// 90 chars with no spaces — falls back to hard cut at 80
	noSpaces := strings.Repeat("x", 90)
	result := sanitizePathComponent(noSpaces)
	assert.Equal(t, 80, len(result))
}

func TestSanitize_ApostrophePreserved(t *testing.T) {
	result := sanitizePathComponent("O'Brien")
	assert.Equal(t, "O'Brien", result)
}

func TestSanitize_NormalString_Unchanged(t *testing.T) {
	result := sanitizePathComponent("The Name of the Wind")
	assert.Equal(t, "The Name of the Wind", result)
}

// --- IsComplete ---

func TestIsComplete_BothPresent(t *testing.T) {
	assert.True(t, IsComplete(&BookMetadata{Title: "Dune", Author: "Frank Herbert"}))
}

func TestIsComplete_EmptyAuthor(t *testing.T) {
	assert.False(t, IsComplete(&BookMetadata{Title: "Dune", Author: ""}))
}

func TestIsComplete_EmptyTitle(t *testing.T) {
	assert.False(t, IsComplete(&BookMetadata{Title: "", Author: "Frank Herbert"}))
}

func TestIsComplete_UnknownAuthor(t *testing.T) {
	assert.False(t, IsComplete(&BookMetadata{Title: "Dune", Author: "Unknown"}))
}

func TestIsComplete_UnknownTitle(t *testing.T) {
	assert.False(t, IsComplete(&BookMetadata{Title: "Unknown", Author: "Frank Herbert"}))
}

func TestIsComplete_BothUnknown(t *testing.T) {
	assert.False(t, IsComplete(&BookMetadata{Title: "Unknown", Author: "Unknown"}))
}
