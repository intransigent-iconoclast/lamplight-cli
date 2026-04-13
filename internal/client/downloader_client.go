package client

import (
	"context"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
)

type DownloaderClient interface {
	Add(ctx context.Context, torrent *ResolvedTorrent) (string, error) // returns torrent hash
	GetTorrentStatus(ctx context.Context, hash string) (*TorrentStatus, error)
	Supports(kind entity.DownloaderType) bool
	Authenticate(ctx context.Context) error
}

type TorrentStatus struct {
	State    string  // Downloading, Seeding, Paused, Error, Queued, Checking
	Progress float64 // 0-100
	FilePath string  // full path to the file once done
}
