// Package eval scores the LLM classifier against a hand-labeled dataset, so
// every prompt or model change produces a number instead of an impression.
package eval

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/chverma/siemagent/internal/classifier"
	"github.com/chverma/siemagent/internal/models"
)

// Case kinds.
const (
	KindTypical     = "typical"
	KindHard        = "hard"
	KindAdversarial = "adversarial"
)

// Dataset splits. Holdout cases are never used while tuning prompts.
const (
	SplitDev     = "dev"
	SplitHoldout = "holdout"
)

// Subsets a run can select.
const (
	SubsetSmoke   = "smoke"   // dev cases marked smoke: cheap, every pull request
	SubsetFull    = "full"    // every dev case: nightly
	SubsetHoldout = "holdout" // holdout cases only: once, at the end
)

// Case is one hand-labeled log line, stored as one JSON object per line.
type Case struct {
	ID               string `json:"id"`
	Log              string `json:"log"`
	Format           string `json:"format,omitempty"` // syslog | json | auto (default)
	Expected         string `json:"expected"`
	ExpectedSeverity string `json:"expected_severity,omitempty"`
	Kind             string `json:"kind"`
	Split            string `json:"split,omitempty"` // dev (default) | holdout
	Smoke            bool   `json:"smoke,omitempty"`
	Note             string `json:"note,omitempty"`
	LabeledBy        string `json:"labeled_by,omitempty"` // human (default) | rule | claude: who chose expected
}

// Who chose a case's label. Labels from rule tags or an AI draft should be
// reviewed by a person; review means setting labeled_by to human.
const (
	LabeledByHuman  = "human"
	LabeledByRule   = "rule"   // mapped from a detection rule's MITRE tag
	LabeledByClaude = "claude" // drafted by an AI assistant, not the model under test
)

// LoadFile reads a JSONL dataset file.
func LoadFile(path string) ([]Case, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return Load(f)
}

// Load reads and checks a JSONL dataset. It rejects unknown fields, duplicate
// ids and labels outside the allowed set, so a typo cannot silently skew scores.
func Load(r io.Reader) ([]Case, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024) // adversarial cases include very long lines

	var cases []Case
	seen := map[string]int{}
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var c Case
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&c); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		if err := c.check(); err != nil {
			return nil, fmt.Errorf("line %d (%s): %w", n, c.ID, err)
		}
		if prev, dup := seen[c.ID]; dup {
			return nil, fmt.Errorf("line %d: duplicate id %q (first on line %d)", n, c.ID, prev)
		}
		seen[c.ID] = n
		cases = append(cases, c)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}

func (c *Case) check() error {
	if c.ID == "" || c.Log == "" {
		return fmt.Errorf("id and log are required")
	}
	l, ok := classifier.ParseLabel(c.Expected)
	if !ok {
		return fmt.Errorf("expected %q is not an allowed label", c.Expected)
	}
	c.Expected = string(l)
	if c.ExpectedSeverity != "" {
		switch models.Severity(c.ExpectedSeverity) {
		case models.SeverityP1, models.SeverityP2, models.SeverityP3, models.SeverityP4, models.SeverityP5:
		default:
			return fmt.Errorf("expected_severity %q is not P1-P5", c.ExpectedSeverity)
		}
	}
	switch c.Kind {
	case KindTypical, KindHard, KindAdversarial:
	default:
		return fmt.Errorf("kind %q is not typical, hard or adversarial", c.Kind)
	}
	switch c.Split {
	case "":
		c.Split = SplitDev
	case SplitDev, SplitHoldout:
	default:
		return fmt.Errorf("split %q is not dev or holdout", c.Split)
	}
	switch c.LabeledBy {
	case "":
		c.LabeledBy = LabeledByHuman
	case LabeledByHuman, LabeledByRule, LabeledByClaude:
	default:
		return fmt.Errorf("labeled_by %q is not human, rule or claude", c.LabeledBy)
	}
	switch c.Format {
	case "":
		c.Format = "auto"
	case "auto", "syslog", "json":
	default:
		return fmt.Errorf("format %q is not auto, syslog or json", c.Format)
	}
	return nil
}

// Select returns the cases in a subset.
func Select(cases []Case, subset string) ([]Case, error) {
	var keep func(Case) bool
	switch subset {
	case SubsetSmoke:
		keep = func(c Case) bool { return c.Split == SplitDev && c.Smoke }
	case SubsetFull:
		keep = func(c Case) bool { return c.Split == SplitDev }
	case SubsetHoldout:
		keep = func(c Case) bool { return c.Split == SplitHoldout }
	default:
		return nil, fmt.Errorf("unknown subset %q (want smoke, full or holdout)", subset)
	}
	var out []Case
	for _, c := range cases {
		if keep(c) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("subset %q has no cases", subset)
	}
	return out, nil
}
