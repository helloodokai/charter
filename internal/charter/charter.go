package charter

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Risk string

const (
	RiskLow      Risk = "low"
	RiskMedium   Risk = "medium"
	RiskHigh     Risk = "high"
	RiskCritical Risk = "critical"
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusReady    Status = "ready"
	StatusApproved Status = "approved"
	StatusArchived Status = "archived"
)

type AcceptanceCriterion struct {
	ID           string `yaml:"id"            json:"id"`
	Statement    string `yaml:"statement"     json:"statement"`
	Verification string `yaml:"verification"  json:"verification"`
	TestHint     string `yaml:"test_hint,omitempty" json:"test_hint,omitempty"`
}

type Unknown struct {
	ID       string `yaml:"id"       json:"id"`
	Question string `yaml:"question" json:"question"`
	Blocking bool   `yaml:"blocking"  json:"blocking"`
}

type BlastRadius struct {
	Files    []string `yaml:"files,omitempty"    json:"files,omitempty"`
	Services []string `yaml:"services,omitempty" json:"services,omitempty"`
	Data     []string `yaml:"data,omitempty"     json:"data,omitempty"`
}

type Constraints struct {
	Performance   []string `yaml:"performance,omitempty"   json:"performance,omitempty"`
	Security      []string `yaml:"security,omitempty"      json:"security,omitempty"`
	Compatibility []string `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	Style         []string `yaml:"style,omitempty"         json:"style,omitempty"`
	Dependencies  []string `yaml:"dependencies,omitempty"  json:"dependencies,omitempty"`
}

type CounterSpec struct {
	Misinterpretations []string `yaml:"misinterpretations" json:"misinterpretations"`
	AmbiguitiesFlagged []string `yaml:"ambiguities_flagged,omitempty" json:"ambiguities_flagged,omitempty"`
}

type Source struct {
	Type string `yaml:"type" json:"type"`
	URL  string `yaml:"url,omitempty"  json:"url,omitempty"`
	Raw  string `yaml:"raw,omitempty"  json:"raw,omitempty"`
}

type TranscriptTurn struct {
	Role    string    `yaml:"role"    json:"role"`
	At      time.Time `yaml:"at"      json:"at"`
	Content string    `yaml:"content" json:"content"`
}

type Charter struct {
	SchemaVersion      string                `yaml:"schema_version" json:"schema_version"`
	ID                  string                `yaml:"id"             json:"id"`
	CreatedAt           time.Time             `yaml:"created_at"     json:"created_at"`
	UpdatedAt           time.Time             `yaml:"updated_at"     json:"updated_at"`
	Authors             []string              `yaml:"authors"        json:"authors"`
	Source              Source                `yaml:"source"         json:"source"`
	Goal                string                `yaml:"goal"           json:"goal"`
	Context             string                `yaml:"context"        json:"context"`
	NonGoals            []string              `yaml:"non_goals"      json:"non_goals"`
	AcceptanceCriteria  []AcceptanceCriterion `yaml:"acceptance_criteria" json:"acceptance_criteria"`
	EdgeCases           []string              `yaml:"edge_cases"     json:"edge_cases"`
	Constraints         Constraints           `yaml:"constraints"    json:"constraints"`
	Unknowns            []Unknown             `yaml:"unknowns"       json:"unknowns"`
	BlastRadius         BlastRadius           `yaml:"blast_radius"   json:"blast_radius"`
	VerificationPlan    []string              `yaml:"verification_plan" json:"verification_plan"`
	RollbackPlan        string                `yaml:"rollback_plan,omitempty" json:"rollback_plan,omitempty"`
	CounterSpec         CounterSpec           `yaml:"counter_spec"   json:"counter_spec"`
	Risk                Risk                  `yaml:"risk"           json:"risk"`
	RiskRationale       string                `yaml:"risk_rationale" json:"risk_rationale"`
	Status              Status                `yaml:"status"         json:"status"`
	Transcript           []TranscriptTurn      `yaml:"transcript,omitempty" json:"transcript,omitempty"`
}

func New(goal string, source Source, author string) *Charter {
	now := time.Now().UTC()
	return &Charter{
		SchemaVersion: "1",
		ID:            NewID(),
		CreatedAt:     now,
		UpdatedAt:      now,
		Authors:       []string{author},
		Source:        source,
		Goal:          goal,
		Status:        StatusDraft,
	}
}

func Load(path string) (*Charter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading charter %s: %w", path, err)
	}
	var c Charter
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing charter %s: %w", path, err)
	}
	return &c, nil
}

func (c *Charter) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating charters dir: %w", err)
	}
	path := filepath.Join(dir, c.ID+".yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling charter: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing charter %s: %w", path, err)
	}
	return nil
}

func (c *Charter) FilePath(dir string) string {
	return filepath.Join(dir, c.ID+".yaml")
}