package service

import (
	"testing"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestMapClientState_Seeding(t *testing.T) {
	status, done := mapClientState("Seeding")
	assert.Equal(t, entity.StatusCompleted, status)
	assert.True(t, done)
}

func TestMapClientState_Error(t *testing.T) {
	status, done := mapClientState("Error")
	assert.Equal(t, entity.StatusFailed, status)
	assert.False(t, done)
}

func TestMapClientState_Downloading(t *testing.T) {
	status, done := mapClientState("Downloading")
	assert.Equal(t, entity.StatusDownloading, status)
	assert.False(t, done)
}

func TestMapClientState_Checking(t *testing.T) {
	status, done := mapClientState("Checking")
	assert.Equal(t, entity.StatusDownloading, status)
	assert.False(t, done)
}

func TestMapClientState_Moving(t *testing.T) {
	status, done := mapClientState("Moving")
	assert.Equal(t, entity.StatusDownloading, status)
	assert.False(t, done)
}

func TestMapClientState_Queued(t *testing.T) {
	status, done := mapClientState("Queued")
	assert.Equal(t, entity.StatusSnatched, status)
	assert.False(t, done)
}

func TestMapClientState_Paused(t *testing.T) {
	status, done := mapClientState("Paused")
	assert.Equal(t, entity.StatusSnatched, status)
	assert.False(t, done)
}

func TestMapClientState_UnknownState(t *testing.T) {
	status, done := mapClientState("SomeWeirdState")
	assert.Equal(t, entity.StatusSnatched, status)
	assert.False(t, done)
}
