package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
)

type JackettClient struct {
	client *http.Client
}

func NewJackettClient() *JackettClient {
	return &JackettClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// for serialization :)
type JackettConfiguredIndexers struct {
	Indexers []JackettConfiguredIndexer `xml:"indexer"`
}

type JackettCategory struct {
	ID      int               `xml:"id,attr"`
	Name    string            `xml:"name,attr"`
	SubCats []JackettCategory `xml:"subcat"`
}

type JackettCaps struct {
	Categories []JackettCategory `xml:"categories>category"`
}

type JackettConfiguredIndexer struct {
	ID    string      `xml:"id,attr"`
	Title string      `xml:"title"`
	Caps  JackettCaps `xml:"caps"`
}

func (c *JackettClient) RetrieveIndexers(ctx context.Context, provider *entity.Provider) ([]dao.ProviderIndexerDAO, error) {
	// /api/v2.0/indexers/all/results/torznab?t=indexers&configured=true&apikey=g29u6iv7yf9lmiektxnzfqroqu879hit
	url := fmt.Sprintf(
		"%s://%s:%d/api/v2.0/indexers/all/results/torznab?t=indexers&configured=true&apikey=%s",
		provider.Scheme,
		provider.Host,
		provider.Port,
		provider.APIKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jackett returned status %d", resp.StatusCode)
	}

	var payload JackettConfiguredIndexers
	if err := xml.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var out []dao.ProviderIndexerDAO
	for _, idx := range payload.Indexers {
		var caps []int
		for _, cat := range idx.Caps.Categories {
			caps = append(caps, cat.ID)
			for _, sub := range cat.SubCats {
				caps = append(caps, sub.ID)
			}
		}

		out = append(out, dao.ProviderIndexerDAO{
			Name:       idx.Title,
			ExternalID: idx.ID,
			Caps:       caps,
		})
	}

	return out, nil
}
