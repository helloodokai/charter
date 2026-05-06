package dialogue

import (
	"fmt"
	"strings"

	"github.com/helloodokai/charter/internal/charter"
)

func (d *Dialogue) extractGoalAndContext(answer string) error {
	goal, context, err := parseGoalContext(answer)
	if err != nil {
		return err
	}
	if goal != "" {
		d.charter.Goal = goal
	}
	if context != "" {
		d.charter.Context = context
	}
	return nil
}

func (d *Dialogue) extractNonGoals(answer string) error {
	lines := splitLines(answer)
	if len(lines) == 0 {
		lines = []string{answer}
	}
	d.charter.NonGoals = append(d.charter.NonGoals, lines...)
	return nil
}

func (d *Dialogue) extractAcceptanceCriteria(answer string) error {
	lines := splitLines(answer)
	if len(lines) == 0 {
		d.charter.AcceptanceCriteria = append(d.charter.AcceptanceCriteria, charter.AcceptanceCriterion{
			ID:           fmt.Sprintf("ac-%d", len(d.charter.AcceptanceCriteria)+1),
			Statement:    strings.TrimSpace(answer),
			Verification: "manual",
		})
		return nil
	}
	for _, line := range lines {
		verification := "test"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "manual") {
			verification = "manual"
		} else if strings.Contains(lower, "metric") {
			verification = "metric"
		}
		d.charter.AcceptanceCriteria = append(d.charter.AcceptanceCriteria, charter.AcceptanceCriterion{
			ID:           fmt.Sprintf("ac-%d", len(d.charter.AcceptanceCriteria)+1),
			Statement:    line,
			Verification: verification,
		})
	}
	return nil
}

func (d *Dialogue) extractEdgeCases(answer string) error {
	lines := splitLines(answer)
	if len(lines) == 0 {
		lines = []string{answer}
	}
	d.charter.EdgeCases = append(d.charter.EdgeCases, lines...)
	return nil
}

func (d *Dialogue) extractBlastRadius(answer string) error {
	br := parseBlastRadius(answer)
	if len(br.Files) == 0 && len(br.Services) == 0 && len(br.Data) == 0 {
		d.charter.BlastRadius.Files = append(d.charter.BlastRadius.Files, splitLines(answer)...)
	} else {
		d.charter.BlastRadius.Files = append(d.charter.BlastRadius.Files, br.Files...)
		d.charter.BlastRadius.Services = append(d.charter.BlastRadius.Services, br.Services...)
		d.charter.BlastRadius.Data = append(d.charter.BlastRadius.Data, br.Data...)
	}
	return nil
}

func (d *Dialogue) extractConstraints(answer string) error {
	c := parseConstraints(answer)
	if len(c.Performance) == 0 && len(c.Security) == 0 && len(c.Compatibility) == 0 && len(c.Style) == 0 && len(c.Dependencies) == 0 {
		d.charter.Constraints.Performance = append(d.charter.Constraints.Performance, splitLines(answer)...)
	} else {
		if len(c.Performance) > 0 {
			d.charter.Constraints.Performance = append(d.charter.Constraints.Performance, c.Performance...)
		}
		if len(c.Security) > 0 {
			d.charter.Constraints.Security = append(d.charter.Constraints.Security, c.Security...)
		}
		if len(c.Compatibility) > 0 {
			d.charter.Constraints.Compatibility = append(d.charter.Constraints.Compatibility, c.Compatibility...)
		}
		if len(c.Style) > 0 {
			d.charter.Constraints.Style = append(d.charter.Constraints.Style, c.Style...)
		}
		if len(c.Dependencies) > 0 {
			d.charter.Constraints.Dependencies = append(d.charter.Constraints.Dependencies, c.Dependencies...)
		}
	}
	return nil
}

func (d *Dialogue) extractUnknowns(answer string) error {
	lines := splitLines(answer)
	if len(lines) == 0 {
		lines = []string{answer}
	}
	for _, item := range lines {
		d.charter.Unknowns = append(d.charter.Unknowns, charter.Unknown{
			ID:       fmt.Sprintf("unk-%d", len(d.charter.Unknowns)+1),
			Question: item,
			Blocking: true,
		})
	}
	return nil
}

func (d *Dialogue) extractRisk(answer string) error {
	lower := strings.ToLower(answer)
	switch {
	case strings.Contains(lower, "critical"):
		d.charter.Risk = charter.RiskCritical
	case strings.Contains(lower, "high"):
		d.charter.Risk = charter.RiskHigh
	case strings.Contains(lower, "medium"):
		d.charter.Risk = charter.RiskMedium
	default:
		d.charter.Risk = charter.RiskLow
	}

	lines := strings.SplitN(answer, "\n", 2)
	if len(lines) > 1 {
		d.charter.RiskRationale = strings.TrimSpace(lines[1])
	} else {
		d.charter.RiskRationale = answer
	}
	return nil
}

func splitLines(text string) []string {
	var items []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "• ")
		if strings.HasPrefix(line, strings.SplitN(line, ".", 2)[0]+". ") {
			line = strings.TrimPrefix(line, strings.SplitN(line, ".", 2)[0]+". ")
		}
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}
	return items
}

func parseGoalContext(text string) (string, string, error) {
	var goal, context string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "GOAL:") {
			goal = strings.TrimSpace(strings.TrimPrefix(line, line[:5]))
			goal = strings.TrimPrefix(goal, ":")
			goal = strings.TrimSpace(goal)
		} else if strings.HasPrefix(strings.ToUpper(line), "CONTEXT:") {
			context = strings.TrimSpace(strings.TrimPrefix(line, line[:8]))
			context = strings.TrimPrefix(context, ":")
			context = strings.TrimSpace(context)
		}
	}
	if goal == "" {
		lines := strings.SplitN(text, "\n", 2)
		goal = strings.TrimSpace(lines[0])
		if len(lines) > 1 {
			context = strings.TrimSpace(lines[1])
		}
	}
	return goal, context, nil
}

func parseBlastRadius(text string) charter.BlastRadius {
	var br charter.BlastRadius
	section := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "FILES:") {
			section = "files"
			continue
		} else if strings.HasPrefix(upper, "SERVICES:") {
			section = "services"
			continue
		} else if strings.HasPrefix(upper, "DATA:") {
			section = "data"
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch section {
		case "files":
			br.Files = append(br.Files, line)
		case "services":
			br.Services = append(br.Services, line)
		case "data":
			br.Data = append(br.Data, line)
		}
	}
	return br
}

func parseConstraints(text string) charter.Constraints {
	var c charter.Constraints
	section := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "performance:") {
			section = "performance"
			val := strings.TrimSpace(strings.TrimPrefix(line, line[:len("performance:")]))
			val = strings.TrimPrefix(val, ":")
			if val != "" && !strings.Contains(strings.ToLower(val), "none") {
				c.Performance = append(c.Performance, strings.TrimSpace(val))
			}
			continue
		} else if strings.HasPrefix(lower, "security:") {
			section = "security"
			val := strings.TrimSpace(strings.TrimPrefix(line, line[:len("security:")]))
			val = strings.TrimPrefix(val, ":")
			if val != "" && !strings.Contains(strings.ToLower(val), "none") {
				c.Security = append(c.Security, strings.TrimSpace(val))
			}
			continue
		} else if strings.HasPrefix(lower, "compatibility:") {
			section = "compatibility"
			val := strings.TrimSpace(strings.TrimPrefix(line, line[:len("compatibility:")]))
			val = strings.TrimPrefix(val, ":")
			if val != "" && !strings.Contains(strings.ToLower(val), "none") {
				c.Compatibility = append(c.Compatibility, strings.TrimSpace(val))
			}
			continue
		} else if strings.HasPrefix(lower, "style:") {
			section = "style"
			val := strings.TrimSpace(strings.TrimPrefix(line, line[:len("style:")]))
			val = strings.TrimPrefix(val, ":")
			if val != "" && !strings.Contains(strings.ToLower(val), "none") {
				c.Style = append(c.Style, strings.TrimSpace(val))
			}
			continue
		} else if strings.HasPrefix(lower, "dependencies:") {
			section = "dependencies"
			val := strings.TrimSpace(strings.TrimPrefix(line, line[:len("dependencies:")]))
			val = strings.TrimPrefix(val, ":")
			if val != "" && !strings.Contains(strings.ToLower(val), "none") {
				c.Dependencies = append(c.Dependencies, strings.TrimSpace(val))
			}
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch section {
		case "performance":
			c.Performance = append(c.Performance, line)
		case "security":
			c.Security = append(c.Security, line)
		case "compatibility":
			c.Compatibility = append(c.Compatibility, line)
		case "style":
			c.Style = append(c.Style, line)
		case "dependencies":
			c.Dependencies = append(c.Dependencies, line)
		}
	}
	return c
}

func parseCounterSpec(content string) charter.CounterSpec {
	var cs charter.CounterSpec
	lines := strings.Split(content, "\n")
	var inAmbiguity bool
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MISINTERPRETATION") {
			inAmbiguity = false
			after := strings.SplitN(line, ":", 2)
			if len(after) > 1 {
				cs.Misinterpretations = append(cs.Misinterpretations, strings.TrimSpace(after[1]))
			}
		} else if strings.HasPrefix(line, "AMBIGUITIES") {
			inAmbiguity = true
		} else if inAmbiguity && strings.HasPrefix(line, "-") {
			cs.AmbiguitiesFlagged = append(cs.AmbiguitiesFlagged, strings.TrimPrefix(line, "- "))
		}
	}
	return cs
}