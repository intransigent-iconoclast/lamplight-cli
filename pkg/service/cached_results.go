package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
)

// Cache-staleness thresholds shared by every caller (CLI, web) that resolves a
// result from the last cached search by index.
const (
	CacheWarnAfter  = 10 * time.Minute // results this old (but not yet stale) get a heads-up
	CacheStaleAfter = 30 * time.Minute // results this old are rejected unless force is set
)

// ResolveCachedResult looks up a search result from the last cached search by its
// 1-based index (the same index shown alongside cached results to the user).
// It enforces the staleness rule: results older than CacheStaleAfter are
// rejected unless force is set. The returned age lets the caller decide whether
// to show a "heads up, these are N minutes old" notice (age > CacheWarnAfter).
func ResolveCachedResult(ctx context.Context, cacheRepo *repository.CacheRepository, index int, force bool) (*dao.SearchResult, time.Duration, error) {
	cache, err := cacheRepo.GetCache(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("no cached results found — run a search first")
	}

	age := time.Since(cache.UpdatedAt)
	if age > CacheStaleAfter && !force {
		return nil, age, fmt.Errorf(
			"search results are %.0f minutes old — re-run your search or use force to download anyway",
			age.Minutes(),
		)
	}

	var results []dao.SearchResult
	if err := json.Unmarshal([]byte(cache.Result), &results); err != nil {
		return nil, age, fmt.Errorf("error parsing cached results: %w", err)
	}
	if len(results) == 0 {
		return nil, age, fmt.Errorf("cached search results are empty")
	}

	i := index - 1
	if i < 0 || i >= len(results) {
		return nil, age, fmt.Errorf("index %d out of range. Last search returned %d results", index, len(results))
	}

	return &results[i], age, nil
}
