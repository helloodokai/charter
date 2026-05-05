package charter

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewID() string {
	now := time.Now().UTC()
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("ch-%s-%s", now.Format("2006-01-02"), hex.EncodeToString(b))
}

func ParseID(s string) (string, error) {
	if len(s) < 3 || s[:3] != "ch-" {
		return "", fmt.Errorf("invalid charter ID format: %q", s)
	}
	return s, nil
}