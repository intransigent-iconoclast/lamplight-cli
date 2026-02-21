package client

import (
	"context"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
)

type DownloaderClient interface {
	Add(ctx context.Context, torrent *ResolvedTorrent) error
	Supports(kind entity.DownloaderType) bool
	Authenticate(ctx context.Context) error
}
