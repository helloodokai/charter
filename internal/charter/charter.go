package charter

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Risk represents the risk level of a charter.
type Risk string

// Risk levels for charter risk assessment.
const (
	RiskLow      Risk = "low"
	RiskMedium   Risk = "medium"
	RiskHigh     Risk = "high"
	RiskCritical Risk = "critical"
)

// Status represents the lifecycle stage of a charter.
type Status string

// Status values for charter lifecycle stages.
const (
	StatusDraft    Status = "draft"
	StatusReady    Status = "ready"
	StatusApproved Status = "approved"
	StatusArchived Status = "archived"
)

// AcceptanceCriterion defines a measurable criterion that must be met for the charter to be considered complete.
type AcceptanceCriterion struct {
	ID           string `yaml:"id"            json:"id"`
	Statement    string `yaml:"statement"     json:"statement"`
	Verification string `yaml:"verification"  json:"verification"`
	TestHint     string `yaml:"test_hint,omitempty" json:"test_hint,omitempty"`
}

// Unknown represents an open question that may block progress on the charter.
type Unknown struct {
	ID       string `yaml:"id"       json:"id"`
	Question string `yaml:"question" json:"question"`
	Blocking bool   `yaml:"blocking"  json:"blocking"`
}

// BlastRadius describes the files, services, and data affected by the charter's change.
type BlastRadius struct {
	Files    []string `yaml:"files,omitempty"    json:"files,omitempty"`
	Services []string `yaml:"services,omitempty" json:"services,omitempty"`
	Data     []string `yaml:"data,omitempty"     json:"data,omitempty"`
}

// Constraints captures non-functional requirements and restrictions for the charter.
type Constraints struct {
	Performance   []string `yaml:"performance,omitempty"   json:"performance,omitempty"`
	Security      []string `yaml:"security,omitempty"      json:"security,omitempty"`
	Compatibility []string `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	Style         []string `yaml:"style,omitempty"         json:"style,omitempty"`
	Dependencies  []string `yaml:"dependencies,omitempty"  json:"dependencies,omitempty"`
}

// CounterSpec holds counter-speculative analysis results that identify potential misinterpretations of the charter.
type CounterSpec struct {
	Misinterpretations []string `yaml:"misinterpretations" json:"misinterpretations"`
	AmbiguitiesFlagged []string `yaml:"ambiguities_flagged,omitempty" json:"ambiguities_flagged,omitempty"`
}

// Source describes where the charter's input material originated.
type Source struct {
	Type string `yaml:"type" json:"type"`
	URL  string `yaml:"url,omitempty"  json:"url,omitempty"`
	Raw  string `yaml:"raw,omitempty"  json:"raw,omitempty"`
}

// TranscriptTurn records a single exchange in the charter dialogue.
type TranscriptTurn struct {
	Role    string    `yaml:"role"    json:"role"`
	At      time.Time `yaml:"at"      json:"at"`
	Content string    `yaml:"content" json:"content"`
}

// Charter is the top-level document that captures the intent, constraints, and verification plan for a body of work.
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

// New creates a new Charter in draft status with the given goal, source, and author.
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

// Load reads and parses a Charter from a YAML file at the given path.
func Load(path string) (*Charter, error) {
	data, err := os.ReadFile(path) //nolint:gosec // expected: user-specified path
	if err != nil {
		return nil, fmt.Errorf("reading charter %s: %w", path, err)
	}
	var c Charter
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing charter %s: %w", path, err)
	}
	return &c, nil
}

// Save writes the Charter as a YAML file in the given directory, using the charter ID as the filename.
func (c *Charter) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
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

// FilePath returns the full path where the Charter YAML file would be stored in the given directory.
func (c *Charter) FilePath(dir string) string {
	return filepath.Join(dir, c.ID+".yaml")
}