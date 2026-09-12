package launcher

import (
	"pika/internal/catalog"
	"testing"
)

func TestCodexUsageOnlyForDesktopApp(t *testing.T) {
	for _, tc := range []struct {
		kind, target string
		want         bool
	}{
		{"app", "chatgpt.desktop", true}, {"app", "codex.desktop", true},
		{"app", "com.openai.codex.desktop", true}, {"file", "chatgpt.desktop", false},
		{"app", "chatgpt-notes.desktop", false}, {"app", "firefox.desktop", false},
	} {
		if got := supportsCodexUsage(catalog.Candidate{Kind: tc.kind, Target: tc.target}); got != tc.want {
			t.Fatal(tc)
		}
	}
}
