package service

import (
	"context"
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/client"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
)

// IndexResult is the outcome of a history action performed against a single
// entry. Warning carries a non-fatal issue (e.g. the client couldn't be told
// about a removal, but history was still cleaned up) — the action still
// succeeded overall.
type IndexResult struct {
	Title   string
	Warning string
	// HadHash is set by CancelByIndex: true if the entry had a torrent hash and
	// removal from the client was attempted, false if there was no hash to
	// remove (client removal was skipped entirely — distinct from a removal
	// that was attempted and failed, which sets Warning instead).
	HadHash bool
}

// RetryByIndex re-sends the history entry at the given 1-based global index
// (position in HistoryRepository.FindAll) to the configured download client,
// saving the new hash and resetting its status to snatched. On add failure the
// entry is marked failed and an error is returned.
func (s *DownloadService) RetryByIndex(ctx context.Context, index int) (*IndexResult, error) {
	entries, err := s.histRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}
	if index <= 0 || index > len(entries) {
		return nil, fmt.Errorf("index %d out of range (have %d entries)", index, len(entries))
	}
	target := entries[index-1]

	resolved, err := client.Resolve(ctx, s.http, target.Link)
	if err != nil {
		return nil, fmt.Errorf("resolve torrent: %w", err)
	}

	dc, _, err := ResolveDownloaderClient(ctx, s.downRepo)
	if err != nil {
		return nil, fmt.Errorf("create downloader client: %w", err)
	}

	hash, err := dc.Add(ctx, resolved)
	if err != nil {
		_ = s.histRepo.UpdateStatus(ctx, target.ID, entity.StatusFailed)
		return nil, fmt.Errorf("add torrent: %w", err)
	}

	result := &IndexResult{Title: target.Title}
	if err := s.histRepo.UpdateStatusAndHash(ctx, target.ID, entity.StatusSnatched, hash); err != nil {
		result.Warning = "couldn't update status: " + err.Error()
	}
	return result, nil
}

// RetryAllFailed re-sends every failed history entry to the configured download
// client. Per-entry failures are captured on that entry's IndexResult.Warning
// rather than aborting the whole batch.
func (s *DownloadService) RetryAllFailed(ctx context.Context) ([]IndexResult, error) {
	entries, err := s.histRepo.FindByStatus(ctx, entity.StatusFailed)
	if err != nil {
		return nil, fmt.Errorf("load failed entries: %w", err)
	}
	if len(entries) == 0 {
		return nil, nil
	}

	dc, _, err := ResolveDownloaderClient(ctx, s.downRepo)
	if err != nil {
		return nil, fmt.Errorf("connect to downloader: %w", err)
	}

	results := make([]IndexResult, 0, len(entries))
	for _, entry := range entries {
		resolved, err := client.Resolve(ctx, s.http, entry.Link)
		if err != nil {
			results = append(results, IndexResult{Title: entry.Title, Warning: "resolve: " + err.Error()})
			continue
		}
		hash, err := dc.Add(ctx, resolved)
		if err != nil {
			results = append(results, IndexResult{Title: entry.Title, Warning: err.Error()})
			continue
		}
		if err := s.histRepo.UpdateStatusAndHash(ctx, entry.ID, entity.StatusSnatched, hash); err != nil {
			results = append(results, IndexResult{Title: entry.Title, Warning: "couldn't update status: " + err.Error()})
			continue
		}
		results = append(results, IndexResult{Title: entry.Title})
	}
	return results, nil
}

// CancelByIndex removes the history entry at the given 1-based global index
// from the download client (if it has a hash) and deletes it from history.
// A client-connect failure is fatal (history is left untouched); a
// remove-on-client failure is non-fatal — the entry is still removed from
// history, since the client may have already finished or lost the torrent.
func (s *DownloadService) CancelByIndex(ctx context.Context, index int, deleteData bool) (*IndexResult, error) {
	entries, err := s.histRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}
	if index <= 0 || index > len(entries) {
		return nil, fmt.Errorf("index %d out of range (have %d entries)", index, len(entries))
	}
	target := entries[index-1]
	result := &IndexResult{Title: target.Title}

	if target.TorrentHash != "" {
		result.HadHash = true
		dc, _, err := ResolveDownloaderClient(ctx, s.downRepo)
		if err != nil {
			return nil, fmt.Errorf("connect to downloader: %w", err)
		}
		if err := dc.Remove(ctx, target.TorrentHash, deleteData); err != nil {
			result.Warning = "couldn't remove from client: " + err.Error()
		}
	}

	if err := s.histRepo.Delete(ctx, target.ID); err != nil {
		return nil, fmt.Errorf("remove from history: %w", err)
	}
	return result, nil
}
