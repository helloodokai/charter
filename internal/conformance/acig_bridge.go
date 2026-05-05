package conformance

// AcigFinding represents a finding in the ACIG interchange format.
type AcigFinding struct {
	Critic     string `json:"critic"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Detail     string `json:"detail,omitempty"`
	CharterRef string `json:"charter_ref,omitempty"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
}

// ToAcig converts a Finding to the ACIG interchange format.
func (f Finding) ToAcig() AcigFinding {
	return AcigFinding{
		Critic:     f.Critic,
		Severity:   string(f.Severity),
		Message:    f.Message,
		Detail:     f.Detail,
		CharterRef: f.CharterRef,
		File:       f.File,
		Line:       f.Line,
	}
}

// ToAcigVerdict converts a Verdict to the ACIG interchange format.
func (v Verdict) ToAcigVerdict() map[string]interface{} {
	findings := make([]AcigFinding, len(v.Findings))
	for i, f := range v.Findings {
		findings[i] = f.ToAcig()
	}
	return map[string]interface{}{
		"charter_id": v.CharterID,
		"goal":       v.Goal,
		"status":     v.Status,
		"score":      v.Score,
		"findings":   findings,
	}
}