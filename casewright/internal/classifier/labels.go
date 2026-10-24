package classifier

import (
	"strings"

	"github.com/chanchalvdev/casewright/casewright/internal/models"
)

// Label is an attack category the classifier is allowed to return. The set is
// fixed so evals can score each label and metrics keep a bounded label set.
type Label string

const (
	LabelBruteForce          Label = "Brute Force"
	LabelCredentialTheft     Label = "Credential Theft"
	LabelWebAttack           Label = "Web Attack"
	LabelReconnaissance      Label = "Reconnaissance"
	LabelPrivilegeEscalation Label = "Privilege Escalation"
	LabelMaliciousExecution  Label = "Malicious Execution"
	LabelDefenseEvasion      Label = "Defense Evasion"
	LabelDataExfiltration    Label = "Data Exfiltration"
	LabelBenign              Label = "Benign"
)

// Labels lists every allowed label.
var Labels = []Label{
	LabelBruteForce, LabelCredentialTheft, LabelWebAttack, LabelReconnaissance,
	LabelPrivilegeEscalation, LabelMaliciousExecution, LabelDefenseEvasion,
	LabelDataExfiltration, LabelBenign,
}

// labelOther is the metrics label for an attack type outside Labels, so a
// model inventing categories cannot grow the number of time series.
const labelOther = "Other"

// Reasons an LLM output is invalid. A valid output has none of them.
const (
	IssueParseError      = "parse_error"
	IssueUnknownLabel    = "unknown_label"
	IssueBadSeverity     = "bad_severity"
	IssueConfidenceRange = "confidence_out_of_range"
)

// ParseLabel matches s against the allowed labels, ignoring case, spacing,
// underscores and hyphens ("privilege_escalation" matches).
func ParseLabel(s string) (Label, bool) {
	key := labelKey(s)
	for _, l := range Labels {
		if labelKey(string(l)) == key {
			return l, true
		}
	}
	return "", false
}

func labelKey(s string) string {
	s = strings.NewReplacer("_", " ", "-", " ").Replace(strings.ToLower(s))
	return strings.Join(strings.Fields(s), " ")
}

func validSeverity(s string) bool {
	switch models.Severity(strings.ToUpper(strings.TrimSpace(s))) {
	case models.SeverityP1, models.SeverityP2, models.SeverityP3, models.SeverityP4, models.SeverityP5:
		return true
	}
	return false
}

// validate reports every way a decoded LLM answer breaks the output contract.
// It must run before normalise, which would hide a bad severity.
func validate(a models.LLMAnalysis) []string {
	var issues []string
	if _, ok := ParseLabel(a.AttackType); !ok {
		issues = append(issues, IssueUnknownLabel)
	}
	if !validSeverity(a.Severity) {
		issues = append(issues, IssueBadSeverity)
	}
	if a.Confidence < 0 || a.Confidence > 1 {
		issues = append(issues, IssueConfidenceRange)
	}
	return issues
}

func metricLabel(attackType string) string {
	if l, ok := ParseLabel(attackType); ok {
		return string(l)
	}
	return labelOther
}
