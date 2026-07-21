package client

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
)

// NewDownloaderClient constructs the concrete DownloaderClient for a stored
// downloader based on its type. This is the single extensibility point for
// supporting new download clients — add a case here and nothing else changes for
// callers, which only ever see the DownloaderClient interface.
func NewDownloaderClient(details *entity.Downloader) (DownloaderClient, error) {
	switch details.ClientType {
	case entity.Deluge:
		return NewDelugeClient(nil, details), nil
	default:
		return nil, fmt.Errorf("unsupported downloader type: %s", details.ClientType)
	}
}
