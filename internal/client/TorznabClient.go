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
	Client *http.Client
}

func NewTorznabClient(client *http.Client) *TorznabClient {
	// log.Println("MADE IT HERE ")
	if client == nil {
		client = http.DefaultClient
	}
	return &TorznabClient{
		Client: client,
	}
}

func (client *TorznabClient) Fetch(ctx context.Context, query utils.TorznabQuery, indexer entity.Indexer) (string, error) {
	params := query.ToParams(indexer.APIKey)

	u, err := url.Parse(indexer.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL %q: %w", indexer.BaseURL, err)
	}

	// so raw
	u.RawQuery = params.Encode()
	// log.Println("Query string: %s", u.RawQuery)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	return string(bodyBytes), nil
}
