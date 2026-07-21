package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/client"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
)

// ErrAlreadyInHistory is returned by Dispatch when the link is already present in
// download history and force was not requested. Callers can detect this (via
// errors.Is) to offer a force re-download.
var ErrAlreadyInHistory = errors.New("already in download history")

// DownloadService dispatches a chosen search result to the configured download
// client and records it in history. It is client-agnostic: the concrete client is
// resolved through the client factory.
type DownloadService struct {
	histRepo *repository.HistoryRepository
	downRepo *repository.DownloaderRepository
	http     *http.Client
}

func NewDownloadService(histRepo *repository.HistoryRepository, downRepo *repository.DownloaderRepository, httpClient *http.Client) *DownloadService {
	return &DownloadService{histRepo: histRepo, downRepo: downRepo, http: httpClient}
}

// DispatchResult describes the outcome of a successful dispatch.
type DispatchResult struct {
	Title      string
	Hash       string
	ClientName string
	// HistoryWarning is non-nil when the item was dispatched to the client but the
	// history record could not be saved (non-fatal).
	HistoryWarning error
}

// Dispatch resolves the result's link, sends it to the highest-priority download
// client, and records a history entry. Returns ErrAlreadyInHistory if the link is
// already known and force is false.
func (s *DownloadService) Dispatch(ctx context.Context, result dao.SearchResult, force bool) (*DispatchResult, error) {
	exists, err := s.histRepo.ExistsByLink(ctx, result.Link)
	if err == nil && exists && !force {
		return nil, ErrAlreadyInHistory
	}

	resolved, err := client.Resolve(ctx, s.http, result.Link)
	if err != nil {
		return nil, fmt.Errorf("resolve torrent: %w", err)
	}

	dc, details, err := ResolveDownloaderClient(ctx, s.downRepo)
	if err != nil {
		return nil, fmt.Errorf("create downloader client: %w", err)
	}

	hash, err := dc.Add(ctx, resolved)
	if err != nil {
		return nil, fmt.Errorf("add torrent: %w", err)
	}

	var sizeBytes int64
	if result.SizeBytes != nil {
		sizeBytes = *result.SizeBytes
	}
	entry := entity.DownloadHistory{
		Title:          result.Title,
		Link:           result.Link,
		IndexerName:    result.IndexerName,
		DownloaderName: details.Name,
		SizeBytes:      sizeBytes,
		Status:         entity.StatusSnatched,
		TorrentHash:    hash,
	}

	out := &DispatchResult{Title: result.Title, Hash: hash, ClientName: details.Name}
	if err := s.histRepo.Save(ctx, &entry); err != nil {
		out.HistoryWarning = err
	}
	return out, nil
}

// ResolveDownloaderClient finds the highest-priority configured downloader and
// constructs its client via the factory, returning the interface plus the stored
// details. Shared by download dispatch and history sync.
func ResolveDownloaderClient(ctx context.Context, downRepo *repository.DownloaderRepository) (client.DownloaderClient, *entity.Downloader, error) {
	details, err := downRepo.FindHighestPriorityDownloader(ctx)
	if err != nil {
		return nil, nil, err
	}
	dc, err := client.NewDownloaderClient(details)
	if err != nil {
		return nil, nil, err
	}
	return dc, details, nil
}
