package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslatePath_NoPrefixConfig_Unchanged(t *testing.T) {
	assert.Equal(t, "/data/incomplete/book.epub", TranslatePath("/data/incomplete/book.epub", "", ""))
}

func TestTranslatePath_OnlyDelugePath_Unchanged(t *testing.T) {
	assert.Equal(t, "/data/incomplete/book.epub", TranslatePath("/data/incomplete/book.epub", "/data", ""))
}

func TestTranslatePath_OnlyHostPath_Unchanged(t *testing.T) {
	assert.Equal(t, "/data/incomplete/book.epub", TranslatePath("/data/incomplete/book.epub", "", "/opt/docker/data"))
}

func TestTranslatePath_MatchingPrefix_Translated(t *testing.T) {
	assert.Equal(t, "/opt/docker/data/incomplete/book.epub",
		TranslatePath("/data/incomplete/book.epub", "/data", "/opt/docker/data"))
}

func TestTranslatePath_NonMatchingPrefix_Unchanged(t *testing.T) {
	assert.Equal(t, "/downloads/book.epub",
		TranslatePath("/downloads/book.epub", "/data", "/opt/docker/data"))
}

func TestTranslatePath_ExactPrefixMatch_NoTrailingSlash(t *testing.T) {
	assert.Equal(t, "/opt", TranslatePath("/data", "/data", "/opt"))
}

func TestTranslatePath_NestedPath(t *testing.T) {
	assert.Equal(t, "/mnt/media/incomplete/Author/Book Title (2020)/book.epub",
		TranslatePath("/data/incomplete/Author/Book Title (2020)/book.epub", "/data/incomplete", "/mnt/media/incomplete"))
}
