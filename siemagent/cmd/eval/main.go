// Command eval scores the LLM classifier on the hand-labeled dataset.
//
//	go run ./cmd/eval -prompt v1 -subset smoke
//	go run ./cmd/eval -prompt v2 -subset full -check evals/threshold.json
//
// The provider comes from the same environment as the server (LLM_PROVIDER,
// OLLAMA_MODEL, GEMINI_API_KEY, ...). Exit status: 0 pass, 1 a threshold was
// broken, 2 the run could not start.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/chverma/siemagent/internal/classifier"
	"github.com/chverma/siemagent/internal/config"
	"github.com/chverma/siemagent/internal/eval"
	"github.com/chverma/siemagent/internal/tracing"
)

func main() {
	os.Exit(run())
}

func run() int {
	dataset := flag.String("dataset", "evals/dataset.jsonl", "JSONL dataset of labeled cases")
	prompt := flag.String("prompt", classifier.DefaultPromptVersion, "prompt version to evaluate")
	subset := flag.String("subset", eval.SubsetSmoke, "smoke | full | holdout")
	outDir := flag.String("out", "evals/results", "directory for the JSON result; empty = do not save")
	check := flag.String("check", "", "threshold file; exit 1 if the run breaks it")
	concurrency := flag.Int("concurrency", 1, "parallel LLM calls")
	rpm := flag.Int("rpm", 0, "max LLM calls per minute (0 = unlimited; use ~5 for free-tier APIs)")
	timeout := flag.Duration("timeout", 2*time.Minute, "timeout per LLM call")
	seed := flag.Int("seed", -1, "sampling seed for repeatable runs (Ollama/OpenAI); -1 = unset")
	validate := flag.Bool("validate", false, "check the dataset and print its balance, without calling an LLM")
	flag.Parse()

	if *validate {
		all, err := eval.LoadFile(*dataset)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: dataset: %v\n", err)
			return 2
		}
		if err := eval.WriteBalance(os.Stdout, all); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 2
		}
		return 0
	}

	cfg := config.Load()
	if cfg.Provider == config.ProviderNone {
		fmt.Fprintln(os.Stderr, "error: LLM_PROVIDER=none: an eval needs an LLM")
		return 2
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	var limits *eval.Threshold
	if *check != "" {
		t, err := eval.LoadThreshold(*check)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 2
		}
		limits = &t
	}

	all, err := eval.LoadFile(*dataset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: dataset: %v\n", err)
		return 2
	}
	cases, err := eval.Select(all, *subset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	cls := classifier.New(cfg.APIKey, cfg.BaseURL, cfg.ModelName)
	if err := cls.UsePrompt(*prompt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}
	if *seed >= 0 {
		cls.SetSeed(*seed)
	}
	cls.SetStreamUsage(cfg.Provider == config.ProviderOllama)

	// Eval spans carry env=eval so they never mix with live traffic.
	shutdownTracing, err := tracing.Setup(context.Background(), tracing.Config{Env: tracing.EnvEval, LLMProvider: cfg.Provider})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: tracing: %v\n", err)
		return 2
	}
	defer func() {
		flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdownTracing(flushCtx); err != nil {
			fmt.Fprintf(os.Stderr, "warning: tracing flush: %v\n", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(os.Stderr, "running %d %s cases with prompt %s on %s/%s...\n", len(cases), *subset, *prompt, cfg.Provider, cfg.ModelName)
	started := time.Now()
	results := eval.Run(ctx, cls, cases, eval.Options{Concurrency: *concurrency, RPM: *rpm, Timeout: *timeout})

	report := eval.Score(results)
	report.PromptVersion = cls.PromptVersion()
	report.Provider = cfg.Provider
	report.Model = cfg.ModelName
	report.Subset = *subset
	report.Dataset = *dataset
	report.StartedAt = started.UTC()
	report.DurationSec = time.Since(started).Seconds()

	if err := eval.WriteText(os.Stdout, report); err != nil {
		fmt.Fprintf(os.Stderr, "error: print report: %v\n", err)
		return 2
	}

	if *outDir != "" {
		path, err := save(*outDir, report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: save result: %v\n", err)
			return 2
		}
		fmt.Fprintf(os.Stderr, "\nSaved %s\n", path)
	}

	if limits != nil {
		if broken := limits.Check(report); len(broken) > 0 {
			fmt.Fprintf(os.Stderr, "\nFAIL: %s\n", *check)
			for _, b := range broken {
				fmt.Fprintf(os.Stderr, "  - %s\n", b)
			}
			return 1
		}
		fmt.Fprintf(os.Stderr, "\nPASS: %s\n", *check)
	}
	return 0
}

// save writes the report as <prompt>_<date>_<provider>_<subset>.json.
func save(dir string, r eval.Report) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s_%s_%s_%s.json", r.PromptVersion, r.StartedAt.Format("2006-01-02T150405"), r.Provider, r.Subset)
	path := filepath.Join(dir, name)
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, append(b, '\n'), 0o644)
}
