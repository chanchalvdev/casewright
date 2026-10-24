package eval

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/chanchalvdev/casewright/casewright/internal/classifier"
	"github.com/chanchalvdev/casewright/casewright/internal/models"
	"github.com/chanchalvdev/casewright/casewright/internal/parser"
)

// Classifier is the part of the classifier an eval needs.
type Classifier interface {
	Classify(ctx context.Context, ev models.LogEvent) (models.ClassifiedEvent, error)
}

// Options tune a run.
type Options struct {
	Concurrency int           // parallel calls; default 1 (CPU Ollama serves one at a time)
	RPM         int           // max calls per minute; 0 = unlimited (free-tier APIs need a limit)
	Timeout     time.Duration // per call; default 2 minutes
}

// Result is the outcome of one case.
type Result struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Expected          string   `json:"expected"`
	ExpectedSeverity  string   `json:"expected_severity,omitempty"`
	Predicted         string   `json:"predicted,omitempty"`
	PredictedSeverity string   `json:"predicted_severity,omitempty"`
	Confidence        float64  `json:"confidence"`
	Issues            []string `json:"issues,omitempty"` // output contract broken: counts as invalid
	Error             string   `json:"error,omitempty"`  // call failed (timeout, quota): counts as error
	Correct           bool     `json:"correct"`
	LatencySeconds    float64  `json:"latency_seconds"`
	Note              string   `json:"note,omitempty"`
}

// Valid reports whether the call succeeded and the answer kept the contract.
func (r Result) Valid() bool { return r.Error == "" && len(r.Issues) == 0 }

// Run classifies every case through the same parser and classifier the API
// uses, and returns results in case order. Invalid answers are never
// corrected or guessed: they count as wrong.
//
// It uses its own small worker loop rather than pipeline.WorkerPool because
// an eval needs each case's error and identity, which the pool drops.
func Run(ctx context.Context, cls Classifier, cases []Case, opt Options) []Result {
	if opt.Concurrency < 1 {
		opt.Concurrency = 1
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 2 * time.Minute
	}

	results := make([]Result, len(cases))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range opt.Concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := parser.New()
			for i := range jobs {
				results[i] = runCase(ctx, cls, p, cases[i], opt.Timeout)
			}
		}()
	}

	var tick <-chan time.Time
	if opt.RPM > 0 {
		t := time.NewTicker(time.Minute / time.Duration(opt.RPM))
		defer t.Stop()
		tick = t.C
	}
	for i := range cases {
		if tick != nil && i > 0 {
			select {
			case <-tick:
			case <-ctx.Done():
			}
		}
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}

func runCase(ctx context.Context, cls Classifier, p *parser.Parser, c Case, timeout time.Duration) Result {
	r := Result{ID: c.ID, Kind: c.Kind, Expected: c.Expected, ExpectedSeverity: c.ExpectedSeverity, Note: c.Note}
	if err := ctx.Err(); err != nil {
		r.Error = err.Error()
		return r
	}
	evs := p.ParseLineWithFormat(c.Log, c.Format)
	if len(evs) == 0 {
		r.Error = "parser returned no event"
		return r
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	out, err := cls.Classify(callCtx, evs[0])
	r.LatencySeconds = time.Since(start).Seconds()

	switch {
	case errors.Is(err, classifier.ErrInvalidOutput):
		r.Issues = []string{classifier.IssueParseError}
	case err != nil:
		r.Error = err.Error()
	default:
		r.Predicted = out.AttackType
		r.PredictedSeverity = string(out.Severity)
		r.Confidence = out.Confidence
		r.Issues = out.OutputIssues
	}
	r.Correct = r.Valid() && r.Predicted == r.Expected
	return r
}
