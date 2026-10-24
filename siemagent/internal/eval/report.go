package eval

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/chverma/siemagent/internal/classifier"
)

// WriteText prints a report for people: headline numbers, per-label scores,
// confidence, then every miss so it can be read and fixed.
func WriteText(w io.Writer, r Report) error {
	p := &printer{w: w}
	p.f("Eval  prompt=%s  provider=%s  model=%s  subset=%s  cases=%d  (%.0fs)\n\n",
		r.PromptVersion, r.Provider, r.Model, r.Subset, r.Cases, r.DurationSec)

	t := p.table(0)
	t.f("accuracy\t%s\n", pct(r.Accuracy))
	for _, k := range []string{KindTypical, KindHard, KindAdversarial} {
		if s, ok := r.ByKind[k]; ok {
			t.f("  %s\t%s\t(%d cases)\n", k, pct(s.Accuracy), s.Cases)
		}
	}
	if r.SeverityCases > 0 {
		t.f("severity accuracy\t%s\t(%d cases with expected severity)\n", pct(r.SeverityAccuracy), r.SeverityCases)
	}
	t.f("invalid-output rate\t%s\t%s\n", pct(r.InvalidRate), reasons(r.InvalidByReason))
	t.f("error rate\t%s\n", pct(r.ErrorRate))
	t.f("latency p50 / p95\t%.1fs / %.1fs\n", r.LatencyP50, r.LatencyP95)
	p.flush(t)

	p.f("\nPer label\n")
	t = p.table(tabwriter.AlignRight)
	t.f("label\tsupport\tpredicted\tprecision\trecall\tF1\t\n")
	for _, l := range classifier.Labels {
		s := r.Labels[string(l)]
		t.f("%s\t%d\t%d\t%s\t%s\t%.2f\t\n", l, s.Support, s.Predicted, pct(s.Precision), pct(s.Recall), s.F1)
	}
	p.flush(t)

	c := r.Confidence
	p.f("\nConfidence  low (<%.1f): %s of valid answers   mean right: %.2f   mean wrong: %.2f\n",
		LowConfidence, pct(c.LowShare), c.MeanCorrect, c.MeanWrong)
	if c.MeanWrong > 0 && c.MeanCorrect-c.MeanWrong < 0.1 {
		p.f("  ! wrong answers are about as confident as right ones: do not trust confidence yet\n")
	}

	p.f("\nMisses (%d)\n", len(r.Misses))
	for _, m := range r.Misses {
		got := m.Predicted
		switch {
		case m.Error != "":
			got = "ERROR " + truncate(m.Error, 120)
		case len(m.Issues) > 0:
			got = fmt.Sprintf("INVALID %v", m.Issues)
			if m.Predicted != "" {
				got += fmt.Sprintf(" %q", m.Predicted)
			}
		}
		p.f("  %-8s %-12s expected %-22q got %s  conf=%.2f\n", m.ID, m.Kind, m.Expected, got, m.Confidence)
		if m.Note != "" {
			p.f("           note: %s\n", m.Note)
		}
	}
	return p.err
}

// printer writes formatted text and keeps the first error, so a report is
// written in one pass and checked once.
type printer struct {
	w   io.Writer
	tw  *tabwriter.Writer
	err error
}

func (p *printer) f(format string, args ...any) {
	if p.err == nil {
		_, p.err = fmt.Fprintf(p.w, format, args...)
	}
}

// table returns a printer whose output is aligned into columns on flush.
func (p *printer) table(flags uint) *printer {
	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', flags)
	return &printer{w: tw, tw: tw, err: p.err}
}

func (p *printer) flush(t *printer) {
	if t.err == nil {
		t.err = t.tw.Flush()
	}
	if p.err == nil {
		p.err = t.err
	}
}

func pct(f float64) string { return fmt.Sprintf("%.1f%%", f*100) }

func reasons(m map[string]int) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%d", k, m[k])
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
