package conformance

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/helloodokai/charter/internal/charter"
)

// Severity represents the severity level of a conformance finding.
type Severity string

// SeverityInfo is an informational finding that does not affect the score.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityBlocking Severity = "blocking"
)

// Finding represents a single conformance issue detected by a grader.
type Finding struct {
	Critic     string   `json:"critic" yaml:"critic"`
	Severity   Severity `json:"severity" yaml:"severity"`
	Message    string   `json:"message" yaml:"message"`
	Detail     string   `json:"detail,omitempty" yaml:"detail,omitempty"`
	CharterRef string   `json:"charter_ref,omitempty" yaml:"charter_ref,omitempty"`
	File       string   `json:"file,omitempty" yaml:"file,omitempty"`
	Line       int      `json:"line,omitempty" yaml:"line,omitempty"`
}

// Verdict summarises the conformance assessment for a charter.
type Verdict struct {
	CharterID string    `json:"charter_id" yaml:"charter_id"`
	Goal      string    `json:"goal" yaml:"goal"`
	Status    string    `json:"status" yaml:"status"`
	Findings  []Finding `json:"findings" yaml:"findings"`
	Score     float64   `json:"score" yaml:"score"`
}

// Grader assesses a charter against a diff and returns findings.
type Grader interface {
	Name() string
	Grade(ch *charter.Charter, diff string) []Finding
}

// BlastRadiusGrader flags files that fall outside the charter's declared blast radius.
type BlastRadiusGrader struct{}

// Name returns the identifier for the blast radius grader.
func (g *BlastRadiusGrader) Name() string { return "blast_radius" }

// Grade evaluates blast radius conformance for the given charter and diff.
func (g *BlastRadiusGrader) Grade(ch *charter.Charter, diff string) []Finding {
	if len(ch.BlastRadius.Files) == 0 {
		return nil
	}

	var filesInDiff []string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			filesInDiff = append(filesInDiff, strings.TrimPrefix(line, "+++ b/"))
		} else if strings.HasPrefix(line, "--- a/") {
			filesInDiff = append(filesInDiff, strings.TrimPrefix(line, "--- a/"))
		}
	}

	var findings []Finding
	for _, f := range filesInDiff {
		matched := false
		for _, pattern := range ch.BlastRadius.Files {
			if match, _ := filepath.Match(pattern, f); match {
				matched = true
				break
			}
			if strings.HasPrefix(f, strings.TrimSuffix(pattern, "**")) {
				matched = true
				break
			}
		}
		if !matched {
			findings = append(findings, Finding{
				Critic:     "charter:blast_radius",
				Severity:   SeverityWarning,
				Message:    fmt.Sprintf("file %s is outside the declared blast radius", f),
				CharterRef: "blast_radius.files",
				File:       f,
			})
		}
	}

	return findings
}

// UnknownGatingGrader flags blocking unknowns that have not been resolved.
type UnknownGatingGrader struct{}

// Name returns the identifier for the unknown gating grader.
func (g *UnknownGatingGrader) Name() string { return "unknown_gating" }

// Grade evaluates unknown gating conformance for the given charter and diff.
func (g *UnknownGatingGrader) Grade(ch *charter.Charter, diff string) []Finding {
	var findings []Finding
	for _, u := range ch.Unknowns {
		if u.Blocking {
			findings = append(findings, Finding{
				Critic:     "charter:unknown_gating",
				Severity:   SeverityBlocking,
				Message:    fmt.Sprintf("blocking unknown: %s", u.Question),
				Detail:     u.Question,
				CharterRef: fmt.Sprintf("unknowns[%s]", u.ID),
			})
		}
	}
	return findings
}

// NonGoalViolationGrader flags diffs that touch areas declared as non-goals.
type NonGoalViolationGrader struct{}

// Name returns the identifier for the non-goal violation grader.
func (g *NonGoalViolationGrader) Name() string { return "non_goal_violation" }

// Grade evaluates non-goal violations for the given charter and diff.
func (g *NonGoalViolationGrader) Grade(ch *charter.Charter, diff string) []Finding {
	if len(ch.NonGoals) == 0 {
		return nil
	}

	keywords := extractKeywords(ch.NonGoals)
	var findings []Finding
	for _, kw := range keywords {
		if strings.Contains(strings.ToLower(diff), strings.ToLower(kw)) {
			findings = append(findings, Finding{
				Critic:     "charter:non_goal_violation",
				Severity:   SeverityError,
				Message:    fmt.Sprintf("diff may touch a non-goal area: %s", kw),
				Detail:     kw,
				CharterRef: "non_goals",
			})
		}
	}
	return findings
}

func extractKeywords(nonGoals []string) []string {
	var keywords []string
	for _, ng := range nonGoals {
		words := strings.Fields(ng)
		for _, w := range words {
			w = strings.ToLower(strings.Trim(w, ".,;:"))
			if len(w) > 4 {
				keywords = append(keywords, w)
			}
		}
	}
	return keywords
}

// Grade runs all graders against the charter and diff, returning a Verdict.
func Grade(ch *charter.Charter, diff string) *Verdict {
	graders := []Grader{
		&BlastRadiusGrader{},
		&UnknownGatingGrader{},
		&NonGoalViolationGrader{},
	}

	var allFindings []Finding
	for _, g := range graders {
		findings := g.Grade(ch, diff)
		allFindings = append(allFindings, findings...)
	}

	status := "pass"
	score := 1.0
	for _, f := range allFindings {
		switch f.Severity {
		case SeverityBlocking:
			status = "fail"
			score = 0.0
		case SeverityError:
			score -= 0.2
		case SeverityWarning:
			score -= 0.1
		}
	}
	if score < 0 {
		score = 0
	}
	if score < 0.6 && status != "fail" {
		status = "fail"
	}

	return &Verdict{
		CharterID: ch.ID,
		Goal:      ch.Goal,
		Status:    status,
		Findings:  allFindings,
		Score:     score,
	}
}