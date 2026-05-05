package conformance

type AcigFinding struct {
	Critic     string `json:"critic"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Detail     string `json:"detail,omitempty"`
	CharterRef string `json:"charter_ref,omitempty"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
}

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