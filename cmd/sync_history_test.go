package cmd

import (
	"testing"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

// --- translatePath ---

func TestTranslatePath_NoPrefixConfig_Unchanged(t *testing.T) {
	result := translatePath("/data/incomplete/book.epub", "", "")
	assert.Equal(t, "/data/incomplete/book.epub", result)
}

func TestTranslatePath_OnlyDelugePath_Unchanged(t *testing.T) {
	result := translatePath("/data/incomplete/book.epub", "/data", "")
	assert.Equal(t, "/data/incomplete/book.epub", result)
}

func TestTranslatePath_OnlyHostPath_Unchanged(t *testing.T) {
	result := translatePath("/data/incomplete/book.epub", "", "/opt/docker/data")
	assert.Equal(t, "/data/incomplete/book.epub", result)
}

func TestTranslatePath_MatchingPrefix_Translated(t *testing.T) {
	result := translatePath(
		"/data/incomplete/book.epub",
		"/data",
		"/opt/docker/data",
	)
	assert.Equal(t, "/opt/docker/data/incomplete/book.epub", result)
}

func TestTranslatePath_NonMatchingPrefix_Unchanged(t *testing.T) {
	result := translatePath(
		"/downloads/book.epub",
		"/data",
		"/opt/docker/data",
	)
	assert.Equal(t, "/downloads/book.epub", result)
}

func TestTranslatePath_ExactPrefixMatch_NoTrailingSlash(t *testing.T) {
	// path == delugePath exactly (no trailing content)
	result := translatePath("/data", "/data", "/opt")
	assert.Equal(t, "/opt", result)
}

func TestTranslatePath_NestedPath(t *testing.T) {
	result := translatePath(
		"/data/incomplete/Author/Book Title (2020)/book.epub",
		"/data/incomplete",
		"/mnt/media/incomplete",
	)
	assert.Equal(t, "/mnt/media/incomplete/Author/Book Title (2020)/book.epub", result)
}

// --- delugeStateToStatus ---

func TestDelugeStateToStatus_Seeding(t *testing.T) {
	status, done := delugeStateToStatus("Seeding")
	assert.Equal(t, entity.StatusCompleted, status)
	assert.True(t, done)
}

func TestDelugeStateToStatus_Error(t *testing.T) {
	status, done := delugeStateToStatus("Error")
	assert.Equal(t, entity.StatusFailed, status)
	assert.False(t, done)
}

func TestDelugeStateToStatus_Downloading(t *testing.T) {
	status, done := delugeStateToStatus("Downloading")
	assert.Equal(t, entity.StatusDownloading, status)
	assert.False(t, done)
}

func TestDelugeStateToStatus_Checking(t *testing.T) {
	status, done := delugeStateToStatus("Checking")
	assert.Equal(t, entity.StatusDownloading, status)
	assert.False(t, done)
}

func TestDelugeStateToStatus_Moving(t *testing.T) {
	status, done := delugeStateToStatus("Moving")
	assert.Equal(t, entity.StatusDownloading, status)
	assert.False(t, done)
}

func TestDelugeStateToStatus_Queued(t *testing.T) {
	status, done := delugeStateToStatus("Queued")
	assert.Equal(t, entity.StatusSnatched, status)
	assert.False(t, done)
}

func TestDelugeStateToStatus_Paused(t *testing.T) {
	status, done := delugeStateToStatus("Paused")
	assert.Equal(t, entity.StatusSnatched, status)
	assert.False(t, done)
}

func TestDelugeStateToStatus_UnknownState(t *testing.T) {
	status, done := delugeStateToStatus("SomeWeirdState")
	assert.Equal(t, entity.StatusSnatched, status)
	assert.False(t, done)
}
