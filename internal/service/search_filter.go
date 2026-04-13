package service

import "github.com/intransigent-iconoclast/lamplight-cli/internal/dao"

func FilterResults(hits []dao.SearchResult, criteria dao.FilterCriteria) []dao.SearchResult {
	// no filter applied so keep it
	if len(criteria.AllowedCategories) == 0 {
		return hits
	}

	// create a set
	allowed := createAllowedSet(criteria.AllowedCategories)

	var filteredHits []dao.SearchResult
	for _, hit := range hits {
		// if this hit has no categories skip this dude
		if len(hit.Categories) == 0 {
			continue
		}

		keep := false
		for _, cat := range hit.Categories {
			if _, ok := allowed[cat]; ok {
				keep = true
				break
			}
		}
		if !keep {
			continue
		}
		filteredHits = append(filteredHits, hit)
	}
	return filteredHits
}

// pulled this out cuz its its an eyesore
func createAllowedSet(cats []int) map[int]struct{} {
	allowed := make(map[int]struct{}, len(cats))
	for _, cat := range cats {
		allowed[cat] = struct{}{}
	}
	return allowed
}
