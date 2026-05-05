package dialogue

// Transcript is an ordered sequence of dialogue Turns.
type Transcript []Turn

// Turn represents a single exchange in a dialogue transcript.
type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LastModelTurn returns the most recent turn with role "tool".
func (t Transcript) LastModelTurn() (Turn, bool) {
	for i := len(t) - 1; i >= 0; i-- {
		if t[i].Role == "tool" {
			return t[i], true
		}
	}
	return Turn{}, false
}

// LastHumanTurn returns the most recent turn with role "human".
func (t Transcript) LastHumanTurn() (Turn, bool) {
	for i := len(t) - 1; i >= 0; i-- {
		if t[i].Role == "human" {
			return t[i], true
		}
	}
	return Turn{}, false
}

// FromCharterTranscript converts a slice of generic interface{} turns into a typed Transcript.
func FromCharterTranscript(turns []interface{}) Transcript {
	var t Transcript
	for _, turn := range turns {
		if tt, ok := turn.(Turn); ok {
			t = append(t, tt)
		}
	}
	return t
}