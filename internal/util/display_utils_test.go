package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- SmartTruncate ---

func TestSmartTruncate_ShorterThanMax_Unchanged(t *testing.T) {
	assert.Equal(t, "hello", SmartTruncate("hello", 10))
}

func TestSmartTruncate_ExactlyMax_Unchanged(t *testing.T) {
	assert.Equal(t, "hello", SmartTruncate("hello", 5))
}

func TestSmartTruncate_CutsAtWordBoundary(t *testing.T) {
	result := SmartTruncate("The Name of the Wind", 15)
	assert.Equal(t, "The Name of...", result)
}

func TestSmartTruncate_HardCutWhenNoBoundaryNearEnd(t *testing.T) {
	// word boundary exists but is too early (before max/2) — hard cut
	result := SmartTruncate("A superlongwordwithnospacesatall", 15)
	assert.Equal(t, "A superlongword...", result)
}

func TestSmartTruncate_EmptyString(t *testing.T) {
	assert.Equal(t, "", SmartTruncate("", 10))
}

func TestSmartTruncate_AppendsDots(t *testing.T) {
	result := SmartTruncate("one two three four", 10)
	assert.True(t, len(result) > 0)
	assert.True(t, result[len(result)-3:] == "...")
}

// --- BytesToMb ---

func TestBytesToMb_Zero(t *testing.T) {
	assert.Equal(t, "0", BytesToMb(0))
}

func TestBytesToMb_ExactMegabyte(t *testing.T) {
	assert.Equal(t, "1", BytesToMb(1_000_000))
}

func TestBytesToMb_LargeFile(t *testing.T) {
	// 1.5 GB
	assert.Equal(t, "1500", BytesToMb(1_500_000_000))
}

func TestBytesToMb_SubMegabyte(t *testing.T) {
	result := BytesToMb(500_000)
	assert.Equal(t, "0.5", result)
}

func TestBytesToMb_Rounds(t *testing.T) {
	// 1,234,567 bytes = 1.234567 MB, rounds to 1.23
	result := BytesToMb(1_234_567)
	assert.Equal(t, "1.23", result)
}

// --- CleanString ---

func TestCleanString_HTMLEntities(t *testing.T) {
	result := CleanString("Tom &amp; Jerry")
	assert.Equal(t, "Tom & Jerry", result)
}

func TestCleanString_NonBreakingSpace(t *testing.T) {
	result := CleanString("hello\u00a0world")
	assert.Equal(t, "hello world", result)
}

func TestCleanString_CollapseWhitespace(t *testing.T) {
	result := CleanString("too   many    spaces")
	assert.Equal(t, "too many spaces", result)
}

func TestCleanString_ApostropheArtifacts(t *testing.T) {
	result := CleanString("don 039 t")
	assert.Equal(t, "don't", result)
}

func TestCleanString_AmpArtifact(t *testing.T) {
	// " amp " (with surrounding spaces) collapses to "&" and spaces are removed
	result := CleanString("fish amp chips")
	assert.Equal(t, "fish&chips", result)
}

func TestCleanString_PlainString_Unchanged(t *testing.T) {
	result := CleanString("The Name of the Wind")
	assert.Equal(t, "The Name of the Wind", result)
}

func TestCleanString_Empty(t *testing.T) {
	assert.Equal(t, "", CleanString(""))
}
