package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chverma/siemagent/internal/eval"
)

func TestSave_NamesFileByVersionProviderAndSubset(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "results")
	r := eval.Report{
		PromptVersion: "v2", Provider: "ollama", Subset: "smoke",
		StartedAt: time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC), Accuracy: 0.5,
	}
	path, err := save(dir, r)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if want := filepath.Join(dir, "v2_2026-10-24T093000_ollama_smoke.json"); path != want {
		t.Errorf("path = %s, want %s", path, want)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back eval.Report
	if err := json.Unmarshal(b, &back); err != nil || back.Accuracy != 0.5 || back.PromptVersion != "v2" {
		t.Errorf("round trip: %+v, %v", back, err)
	}
}
