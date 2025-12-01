package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
)

type TorznabClient struct {
	client *http.Client
}

func NewTorznabClient(client *http.Client) *TorznabClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &TorznabClient{
		client: client,
	}
}

func (client *TorznabClient) fetch(ctx context.Context, query utils.TorznabQuery, indexer entity.Indexer) (string, error) {
	params := query.ToParams(indexer.APIKey)

	u, err := url.Parse(indexer.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL %q: %w", indexer.BaseURL, err)
	}

	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// you could read body for more detail here if you want
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	return string(bodyBytes), nil
}
