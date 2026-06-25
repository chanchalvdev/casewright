package qdrant

import (
	"context"

	qd "github.com/qdrant/go-client/qdrant"

	"github.com/chverma/siemagent/internal/api"
)

// APISearchResult adapts pkg/qdrant.SearchResult to api.SearchResult.
// The Store.Search method is wrapped here so the api package never imports
// the raw Qdrant client types.

// APIStore wraps Store and implements api.Searcher.
type APIStore struct {
	*Store
}

// NewAPIStore returns a Store that satisfies api.Searcher.
func NewAPIStore(addr, collection string) (*APIStore, error) {
	s, err := New(addr, collection)
	if err != nil {
		return nil, err
	}
	return &APIStore{s}, nil
}

// Search implements api.Searcher — builds a Qdrant keyword filter for
// severityFilter (empty string = no filter) and delegates to Store.Search.
func (a *APIStore) Search(ctx context.Context, queryVector []float32, topK uint64, severityFilter string) ([]api.SearchResult, error) {
	var filter *qd.Filter
	if severityFilter != "" {
		filter = &qd.Filter{
			Must: []*qd.Condition{
				{
					ConditionOneOf: &qd.Condition_Field{
						Field: &qd.FieldCondition{
							Key: "severity",
							Match: &qd.Match{
								MatchValue: &qd.Match_Keyword{
									Keyword: severityFilter,
								},
							},
						},
					},
				},
			},
		}
	}

	hits, err := a.Store.Search(ctx, queryVector, topK, filter)
	if err != nil {
		return nil, err
	}

	out := make([]api.SearchResult, len(hits))
	for i, h := range hits {
		out[i] = api.SearchResult{
			ID:      h.ID,
			Score:   h.Score,
			Payload: h.Payload,
		}
	}
	return out, nil
}
