package service

import (
	"context"
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/client"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
)

// SyncService polls a download client for the status of active downloads and
// updates history accordingly. It contains no presentation logic — callers (CLI
// watch view, web SSE poller) render SyncResult however they like.
type SyncService struct {
	histRepo *repository.HistoryRepository
}

func NewSyncService(histRepo *repository.HistoryRepository) *SyncService {
	return &SyncService{histRepo: histRepo}
}

// SyncItem is the outcome of syncing a single active download.
type SyncItem struct {
	Entry    entity.DownloadHistory
	State    string                // client-reported state (e.g. Downloading, Seeding)
	Progress float64               // 0-100
	Status   entity.DownloadStatus // mapped internal status
	FilePath string                // translated host path (once known)
	Done     bool                  // completed
	Err      error                 // per-item error (status fetch or history update)
}

// SyncResult is the outcome of a single sync pass over all active downloads.
type SyncResult struct {
	Items   []SyncItem
	AllDone bool
}

// SyncOnce polls the client for every active download and updates history. It
// returns a structured result per entry; per-item errors are captured on the item
// rather than aborting the whole pass.
func (s *SyncService) SyncOnce(ctx context.Context, dc client.DownloaderClient, delugePath, hostPath string) (*SyncResult, error) {
	active, err := s.histRepo.FindActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active downloads: %w", err)
	}

	result := &SyncResult{AllDone: true}

	for _, entry := range active {
		item := SyncItem{Entry: entry}

		status, err := dc.GetTorrentStatus(ctx, entry.TorrentHash)
		if err != nil {
			item.Err = err
			result.AllDone = false
			result.Items = append(result.Items, item)
			continue
		}

		item.State = status.State
		item.Progress = status.Progress
		item.FilePath = utils.TranslatePath(status.FilePath, delugePath, hostPath)
		item.Status, item.Done = mapClientState(status.State)
		if !item.Done {
			result.AllDone = false
		}

		if err := s.histRepo.UpdateStatusAndPath(ctx, entry.ID, item.Status, item.FilePath); err != nil {
			item.Err = err
		}

		result.Items = append(result.Items, item)
	}

	if len(active) == 0 {
		result.AllDone = false
	}

	return result, nil
}

// mapClientState maps a download-client state string to our status enum.
// Returns (status, isComplete).
func mapClientState(state string) (entity.DownloadStatus, bool) {
	switch state {
	case "Seeding": // 100% downloaded, now seeding
		return entity.StatusCompleted, true
	case "Error":
		return entity.StatusFailed, false
	case "Downloading", "Checking", "Moving":
		return entity.StatusDownloading, false
	default: // Queued, Paused, etc.
		return entity.StatusSnatched, false
	}
}
