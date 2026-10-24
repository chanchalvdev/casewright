package eval

import (
	"sort"
	"time"

	"github.com/chverma/siemagent/internal/classifier"
)

// LowConfidence is the confidence below which a prediction counts as "low".
const LowConfidence = 0.6

// Confusion-matrix columns for answers that are not a label.
const (
	PredictedInvalid = "(invalid)"
	PredictedError   = "(error)"
)

// Report is the scored outcome of a run. It is printed and saved as JSON.
type Report struct {
	PromptVersion string    `json:"prompt_version"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	Subset        string    `json:"subset"`
	Dataset       string    `json:"dataset"`
	StartedAt     time.Time `json:"started_at"`
	DurationSec   float64   `json:"duration_seconds"`

	Cases            int            `json:"cases"`
	Accuracy         float64        `json:"accuracy"`
	SeverityCases    int            `json:"severity_cases"`
	SeverityAccuracy float64        `json:"severity_accuracy"`
	InvalidRate      float64        `json:"invalid_rate"`
	InvalidByReason  map[string]int `json:"invalid_by_reason"`
	ErrorRate        float64        `json:"error_rate"`

	ByKind     map[string]KindScore      `json:"by_kind"`
	Labels     map[string]LabelScore     `json:"labels"`
	Confusion  map[string]map[string]int `json:"confusion"` // expected → predicted → count
	Confidence ConfidenceScore           `json:"confidence"`
	LatencyP50 float64                   `json:"latency_p50_seconds"`
	LatencyP95 float64                   `json:"latency_p95_seconds"`

	Misses  []Result `json:"misses"`
	Results []Result `json:"results"`
}

// KindScore is accuracy on one case kind.
type KindScore struct {
	Cases    int     `json:"cases"`
	Accuracy float64 `json:"accuracy"`
}

// LabelScore is precision and recall for one label. Support is how many cases
// expect the label; Predicted is how many valid answers chose it.
type LabelScore struct {
	Support   int     `json:"support"`
	Predicted int     `json:"predicted"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
}

// ConfidenceScore shows whether self-reported confidence means anything: if
// wrong answers are as confident as right ones, it cannot be trusted yet.
type ConfidenceScore struct {
	LowShare    float64 `json:"low_share"` // share of valid answers below LowConfidence
	MeanCorrect float64 `json:"mean_correct"`
	MeanWrong   float64 `json:"mean_wrong"` // valid but wrong answers
}

// Score turns results into a report. Accuracy is over every case: invalid
// answers and failed calls count as wrong.
func Score(results []Result) Report {
	r := Report{
		Cases:           len(results),
		InvalidByReason: map[string]int{},
		ByKind:          map[string]KindScore{},
		Labels:          map[string]LabelScore{},
		Confusion:       map[string]map[string]int{},
		Misses:          []Result{},
		Results:         results,
	}
	if len(results) == 0 {
		return r
	}

	var correct, invalid, errs, sevCorrect, low, valid int
	var confRight, confWrong []float64
	kindTotal, kindCorrect := map[string]int{}, map[string]int{}
	support, predicted, truePos := map[string]int{}, map[string]int{}, map[string]int{}
	latencies := make([]float64, 0, len(results))

	for _, res := range results {
		latencies = append(latencies, res.LatencySeconds)
		kindTotal[res.Kind]++
		support[res.Expected]++

		col := res.Predicted
		switch {
		case res.Error != "":
			errs++
			col = PredictedError
		case len(res.Issues) > 0:
			invalid++
			col = PredictedInvalid
			for _, is := range res.Issues {
				r.InvalidByReason[is]++
			}
		default:
			valid++
			predicted[res.Predicted]++
			if res.Confidence < LowConfidence {
				low++
			}
			if res.Correct {
				confRight = append(confRight, res.Confidence)
			} else {
				confWrong = append(confWrong, res.Confidence)
			}
			if res.ExpectedSeverity != "" {
				r.SeverityCases++
				if res.PredictedSeverity == res.ExpectedSeverity {
					sevCorrect++
				}
			}
		}
		if r.Confusion[res.Expected] == nil {
			r.Confusion[res.Expected] = map[string]int{}
		}
		r.Confusion[res.Expected][col]++

		if res.Correct {
			correct++
			kindCorrect[res.Kind]++
			truePos[res.Expected]++
		} else {
			r.Misses = append(r.Misses, res)
		}
	}

	n := float64(len(results))
	r.Accuracy = float64(correct) / n
	r.InvalidRate = float64(invalid) / n
	r.ErrorRate = float64(errs) / n
	r.SeverityAccuracy = ratio(sevCorrect, r.SeverityCases)
	r.Confidence = ConfidenceScore{LowShare: ratio(low, valid), MeanCorrect: mean(confRight), MeanWrong: mean(confWrong)}
	for k, t := range kindTotal {
		r.ByKind[k] = KindScore{Cases: t, Accuracy: ratio(kindCorrect[k], t)}
	}
	for _, l := range classifier.Labels {
		name := string(l)
		s := LabelScore{Support: support[name], Predicted: predicted[name]}
		s.Precision = ratio(truePos[name], s.Predicted)
		s.Recall = ratio(truePos[name], s.Support)
		if s.Precision+s.Recall > 0 {
			s.F1 = 2 * s.Precision * s.Recall / (s.Precision + s.Recall)
		}
		r.Labels[name] = s
	}
	sort.Float64s(latencies)
	r.LatencyP50 = percentile(latencies, 0.50)
	r.LatencyP95 = percentile(latencies, 0.95)
	return r
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// percentile uses the nearest-rank method on sorted values.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(p*float64(len(sorted))+0.999999) - 1
	return sorted[max(0, min(i, len(sorted)-1))]
}
