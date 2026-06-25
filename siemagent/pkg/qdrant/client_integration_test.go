//go:build integration

package qdrant

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	qd "github.com/qdrant/go-client/qdrant"
)

func qdrantAddr(t *testing.T) string {
	t.Helper()
	if addr := os.Getenv("QDRANT_TEST_HOST"); addr != "" {
		return addr
	}
	return "localhost:6334"
}

func TestQdrant_Integration(t *testing.T) {
	collection := fmt.Sprintf("test_%d", time.Now().UnixNano())
	addr := qdrantAddr(t)

	store, err := New(addr, collection)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	// Create a 4-dimensional test collection.
	t.Run("EnsureCollection", func(t *testing.T) {
		if err := store.EnsureCollection(ctx, 4); err != nil {
			t.Fatalf("EnsureCollection: %v", err)
		}
		// Idempotent — calling again must not error.
		if err := store.EnsureCollection(ctx, 4); err != nil {
			t.Fatalf("EnsureCollection (second call): %v", err)
		}
	})

	// Upsert 5 test points.
	type point struct {
		id      string
		vector  []float32
		payload map[string]any
	}
	points := []point{
		{"ev1", []float32{1, 0, 0, 0}, map[string]any{"severity": "P1", "attack_type": "Brute Force"}},
		{"ev2", []float32{0, 1, 0, 0}, map[string]any{"severity": "P2", "attack_type": "Port Scan"}},
		{"ev3", []float32{0, 0, 1, 0}, map[string]any{"severity": "P3", "attack_type": "SQL Injection"}},
		{"ev4", []float32{0, 0, 0, 1}, map[string]any{"severity": "P4", "attack_type": "Policy Violation"}},
		{"ev5", []float32{0.9, 0.1, 0, 0}, map[string]any{"severity": "P1", "attack_type": "Brute Force"}},
	}

	t.Run("Upsert", func(t *testing.T) {
		for _, p := range points {
			if err := store.Upsert(ctx, p.id, p.vector, p.payload); err != nil {
				t.Fatalf("Upsert %s: %v", p.id, err)
			}
		}
	})

	t.Run("Search_TopK", func(t *testing.T) {
		// Query closest to [1,0,0,0] — should return ev1 and ev5 as top results.
		results, err := store.Search(ctx, []float32{1, 0, 0, 0}, 3, nil)
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least 1 result")
		}
		top := results[0]
		if top.Payload["attack_type"] != "Brute Force" {
			t.Errorf("top result attack_type: want Brute Force, got %v", top.Payload["attack_type"])
		}
	})

	t.Run("Search_WithFilter", func(t *testing.T) {
		filter := &qd.Filter{
			Must: []*qd.Condition{{
				ConditionOneOf: &qd.Condition_Field{
					Field: &qd.FieldCondition{
						Key: "severity",
						Match: &qd.Match{
							MatchValue: &qd.Match_Keyword{Keyword: "P2"},
						},
					},
				},
			}},
		}
		results, err := store.Search(ctx, []float32{0, 1, 0, 0}, 5, filter)
		if err != nil {
			t.Fatalf("Search with filter: %v", err)
		}
		for _, r := range results {
			if r.Payload["severity"] != "P2" {
				t.Errorf("filter breach: expected P2 severity, got %v", r.Payload["severity"])
			}
		}
	})

	// Clean up the test collection.
	t.Run("Cleanup", func(t *testing.T) {
		if err := store.client.DeleteCollection(ctx, collection); err != nil {
			t.Logf("cleanup: %v (non-fatal)", err)
		}
	})
}
