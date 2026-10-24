package classifier

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/chverma/siemagent/internal/models"
)

func TestPrompt_DefaultIsEmbedded(t *testing.T) {
	p, err := Prompt(DefaultPromptVersion)
	if err != nil {
		t.Fatalf("Prompt(%q): %v", DefaultPromptVersion, err)
	}
	if !strings.Contains(p, `"attack_type"`) {
		t.Errorf("default prompt does not describe the output contract")
	}
	if !slices.Contains(PromptVersions(), DefaultPromptVersion) {
		t.Errorf("PromptVersions() = %v, missing %q", PromptVersions(), DefaultPromptVersion)
	}
}

func TestUsePrompt_UnknownVersionKeepsCurrent(t *testing.T) {
	c := New("k", "http://localhost", "m")
	if err := c.UsePrompt("v999"); err == nil {
		t.Fatal("UsePrompt(v999): expected error")
	}
	if got := c.PromptVersion(); got != DefaultPromptVersion {
		t.Errorf("PromptVersion() = %q after failed switch, want %q", got, DefaultPromptVersion)
	}
	if err := c.UsePrompt(DefaultPromptVersion); err != nil {
		t.Fatalf("UsePrompt(%q): %v", DefaultPromptVersion, err)
	}
}

func TestParseLabel(t *testing.T) {
	cases := map[string]Label{
		"Brute Force":          LabelBruteForce,
		"  brute   force ":     LabelBruteForce,
		"privilege_escalation": LabelPrivilegeEscalation,
		"Data-Exfiltration":    LabelDataExfiltration,
		"BENIGN":               LabelBenign,
	}
	for in, want := range cases {
		if got, ok := ParseLabel(in); !ok || got != want {
			t.Errorf("ParseLabel(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "SQL Injection", "Brute", "Malware"} {
		if got, ok := ParseLabel(in); ok {
			t.Errorf("ParseLabel(%q) = %q, want no match", in, got)
		}
	}
}

func TestValidate(t *testing.T) {
	valid := models.LLMAnalysis{AttackType: "Web Attack", Severity: "p3", Confidence: 0.7}
	cases := []struct {
		name string
		edit func(*models.LLMAnalysis)
		want []string
	}{
		{"valid", func(*models.LLMAnalysis) {}, nil},
		{"unknown label", func(a *models.LLMAnalysis) { a.AttackType = "SQL Injection" }, []string{IssueUnknownLabel}},
		{"bad severity", func(a *models.LLMAnalysis) { a.Severity = "Critical" }, []string{IssueBadSeverity}},
		{"confidence above 1", func(a *models.LLMAnalysis) { a.Confidence = 95 }, []string{IssueConfidenceRange}},
		{"everything wrong", func(a *models.LLMAnalysis) { a.AttackType, a.Severity, a.Confidence = "", "", -1 },
			[]string{IssueUnknownLabel, IssueBadSeverity, IssueConfidenceRange}},
	}
	for _, tc := range cases {
		a := valid
		tc.edit(&a)
		if got := validate(a); !slices.Equal(got, tc.want) {
			t.Errorf("%s: validate = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMetricLabel_BoundsUnknownTypes(t *testing.T) {
	if got := metricLabel("brute force"); got != string(LabelBruteForce) {
		t.Errorf("metricLabel(brute force) = %q", got)
	}
	if got := metricLabel("Totally New Category"); got != labelOther {
		t.Errorf("metricLabel(unknown) = %q, want %q", got, labelOther)
	}
}

func TestClassify_ReportsOutputIssues(t *testing.T) {
	srv := makeSSEServer(`{"attack_type":"SQL Injection","severity":"P9","confidence":0.9,"iocs":[]}`)
	defer srv.Close()

	got, err := newTestClassifier(srv).Classify(context.Background(), sampleEvent())
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if want := []string{IssueUnknownLabel, IssueBadSeverity}; !slices.Equal(got.OutputIssues, want) {
		t.Errorf("OutputIssues = %v, want %v", got.OutputIssues, want)
	}
	// Live traffic still gets a usable verdict.
	if got.Severity != models.SeverityP5 {
		t.Errorf("Severity = %q, want P5", got.Severity)
	}
	if got.PromptVersion != DefaultPromptVersion {
		t.Errorf("PromptVersion = %q, want %q", got.PromptVersion, DefaultPromptVersion)
	}
}

func TestClassify_ValidOutputIsCanonical(t *testing.T) {
	srv := makeSSEServer(`{"attack_type":"brute_force","severity":"P3","confidence":0.8,"iocs":[]}`)
	defer srv.Close()

	got, err := newTestClassifier(srv).Classify(context.Background(), sampleEvent())
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if len(got.OutputIssues) != 0 {
		t.Errorf("OutputIssues = %v, want none", got.OutputIssues)
	}
	if got.AttackType != string(LabelBruteForce) {
		t.Errorf("AttackType = %q, want %q", got.AttackType, LabelBruteForce)
	}
}
