package dialogue

type Transcript []Turn

type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (t Transcript) LastModelTurn() (Turn, bool) {
	for i := len(t) - 1; i >= 0; i-- {
		if t[i].Role == "tool" {
			return t[i], true
		}
	}
	return Turn{}, false
}

func (t Transcript) LastHumanTurn() (Turn, bool) {
	for i := len(t) - 1; i >= 0; i-- {
		if t[i].Role == "human" {
			return t[i], true
		}
	}
	return Turn{}, false
}

func FromCharterTranscript(turns []interface{}) Transcript {
	var t Transcript
	for _, turn := range turns {
		if tt, ok := turn.(Turn); ok {
			t = append(t, tt)
		}
	}
	return t
}