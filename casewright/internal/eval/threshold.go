package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/chanchalvdev/casewright/casewright/internal/classifier"
)

// Threshold is the CI gate: a run that breaks any set limit fails. Unset
// (null) limits are not checked.
type Threshold struct {
	PromptVersion          string             `json:"prompt_version,omitempty"` // the run the limits were based on
	Note                   string             `json:"note,omitempty"`
	MinAccuracy            *float64           `json:"min_accuracy"`
	MinAdversarialAccuracy *float64           `json:"min_adversarial_accuracy"`
	MaxInvalidRate         *float64           `json:"max_invalid_rate"`
	MaxErrorRate           *float64           `json:"max_error_rate"`
	MinLabelRecall         map[string]float64 `json:"min_label_recall,omitempty"`
}

// LoadThreshold reads a threshold file, rejecting unknown fields and labels.
func LoadThreshold(path string) (Threshold, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Threshold{}, err
	}
	var t Threshold
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return Threshold{}, fmt.Errorf("%s: %w", path, err)
	}
	for l := range t.MinLabelRecall {
		if _, ok := classifier.ParseLabel(l); !ok {
			return Threshold{}, fmt.Errorf("%s: min_label_recall has unknown label %q", path, l)
		}
	}
	return t, nil
}

// Check returns one message per broken limit; empty means the run passes.
func (t Threshold) Check(r Report) []string {
	var broken []string
	atLeast := func(name string, got float64, limit *float64) {
		if limit != nil && got < *limit {
			broken = append(broken, fmt.Sprintf("%s %.3f is below the minimum %.3f", name, got, *limit))
		}
	}
	atMost := func(name string, got float64, limit *float64) {
		if limit != nil && got > *limit {
			broken = append(broken, fmt.Sprintf("%s %.3f is above the maximum %.3f", name, got, *limit))
		}
	}

	atLeast("accuracy", r.Accuracy, t.MinAccuracy)
	if t.MinAdversarialAccuracy != nil {
		if adv, ok := r.ByKind[KindAdversarial]; ok {
			atLeast("adversarial accuracy", adv.Accuracy, t.MinAdversarialAccuracy)
		} else {
			broken = append(broken, "min_adversarial_accuracy is set but the run has no adversarial cases")
		}
	}
	atMost("invalid-output rate", r.InvalidRate, t.MaxInvalidRate)
	atMost("error rate", r.ErrorRate, t.MaxErrorRate)

	labels := make([]string, 0, len(t.MinLabelRecall))
	for l := range t.MinLabelRecall {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	for _, l := range labels {
		canon, _ := classifier.ParseLabel(l)
		s := r.Labels[string(canon)]
		if s.Support == 0 {
			continue // the subset has no case for this label
		}
		limit := t.MinLabelRecall[l]
		atLeast(fmt.Sprintf("recall for %q", canon), s.Recall, &limit)
	}
	return broken
}
