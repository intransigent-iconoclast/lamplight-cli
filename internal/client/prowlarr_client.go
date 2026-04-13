package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
)

type ProwlarrClient struct {
	client *http.Client
}

func NewProwlarrClient() *ProwlarrClient {
	return &ProwlarrClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type prowlarrCategory struct {
	ID            int                `json:"id"`
	Name          string             `json:"name"`
	SubCategories []prowlarrCategory `json:"subCategories"`
}

type prowlarrCapabilities struct {
	Categories []prowlarrCategory `json:"categories"`
}

type prowlarrIndexer struct {
	ID           int                  `json:"id"`
	Name         string               `json:"name"`
	Enable       bool                 `json:"enable"`
	Protocol     string               `json:"protocol"`
	Capabilities prowlarrCapabilities `json:"capabilities"`
}

func (c *ProwlarrClient) RetrieveIndexers(ctx context.Context, provider *entity.Provider) ([]dao.ProviderIndexerDAO, error) {

	url := fmt.Sprintf(
		"%s://%s:%d/api/v1/indexer",
		provider.Scheme,
		provider.Host,
		provider.Port,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Api-Key", provider.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prowlarr returned status %d", resp.StatusCode)
	}

	var payload []prowlarrIndexer
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var out []dao.ProviderIndexerDAO

	for _, idx := range payload {
		if !idx.Enable {
			continue
		}
		if idx.Protocol != "torrent" {
			continue
		}

		var caps []int
		for _, cat := range idx.Capabilities.Categories {
			caps = append(caps, cat.ID)
			for _, sub := range cat.SubCategories {
				caps = append(caps, sub.ID)
			}
		}

		out = append(out, dao.ProviderIndexerDAO{
			Name:       idx.Name,
			ExternalID: fmt.Sprintf("%d", idx.ID),
			Caps:       caps,
		})
	}

	return out, nil
}
