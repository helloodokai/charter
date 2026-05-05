package conformance

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/helloodokai/charter/internal/charter"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityBlocking Severity = "blocking"
)

type Finding struct {
	Critic     string   `json:"critic" yaml:"critic"`
	Severity   Severity `json:"severity" yaml:"severity"`
	Message    string   `json:"message" yaml:"message"`
	Detail     string   `json:"detail,omitempty" yaml:"detail,omitempty"`
	CharterRef string   `json:"charter_ref,omitempty" yaml:"charter_ref,omitempty"`
	File       string   `json:"file,omitempty" yaml:"file,omitempty"`
	Line       int      `json:"line,omitempty" yaml:"line,omitempty"`
}

type Verdict struct {
	CharterID string    `json:"charter_id" yaml:"charter_id"`
	Goal      string    `json:"goal" yaml:"goal"`
	Status    string    `json:"status" yaml:"status"`
	Findings  []Finding `json:"findings" yaml:"findings"`
	Score     float64   `json:"score" yaml:"score"`
}

type Grader interface {
	Name() string
	Grade(ch *charter.Charter, diff string) []Finding
}

type BlastRadiusGrader struct{}

func (g *BlastRadiusGrader) Name() string { return "blast_radius" }

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

type UnknownGatingGrader struct{}

func (g *UnknownGatingGrader) Name() string { return "unknown_gating" }

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

type NonGoalViolationGrader struct{}

func (g *NonGoalViolationGrader) Name() string { return "non_goal_violation" }

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
		if f.Severity == SeverityBlocking {
			status = "fail"
			score = 0.0
			break
		}
		if f.Severity == SeverityError {
			score -= 0.2
		} else if f.Severity == SeverityWarning {
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