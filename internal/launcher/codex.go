package launcher

import (
	"fmt"
	"pika/internal/catalog"
	"pika/internal/codexusage"
	"strings"
)

func supportsCodexUsage(c catalog.Candidate) bool {
	if c.Kind != "app" {
		return false
	}
	switch strings.ToLower(c.Target) {
	case "chatgpt.desktop", "codex.desktop", "com.openai.codex.desktop":
		return true
	default:
		return false
	}
}
func (s *Service) CodexUsage(id string, refresh bool) (codexusage.Snapshot, error) {
	c, ok := s.snapshot.Load().ByID[id]
	if !ok || !supportsCodexUsage(c) {
		return codexusage.Snapshot{}, fmt.Errorf("Quota is only available for the Codex application")
	}
	return s.codexQuota.Read(s.ctx, refresh), nil
}
