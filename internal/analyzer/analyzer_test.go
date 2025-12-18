package analyzer

import (
	"testing"
	"time"

	"github.com/rgrouls/terminal-wrapped/internal/parser"
)

func TestAnalyzeCDNormalization(t *testing.T) {
	aliases := map[string]string{
		"gc": "cd ~/code",
		"gp": "cd ~/code/projects",
	}

	commands := []parser.Command{
		{Command: "cd foo", Timestamp: time.Now()},                 // foo=1
		{Command: "cd foo/bar", Timestamp: time.Now()},             // foo=1, bar=1
		{Command: "cd ./foo", Timestamp: time.Now()},               // foo=1
		{Command: "gc", Timestamp: time.Now()},                     // (cd ~/code) -> ~/code=1
		{Command: "gp", Timestamp: time.Now()},                     // (cd ~/code/projects) -> ~/code/projects=1, ~/code=1
		{Command: "cd /Users/rgrouls/docs", Timestamp: time.Now()}, // /Users/rgrouls/docs=1 (Users/rgrouls skipped)
	}

	stats := Analyze(commands, aliases)

	// Expected counts:
	// foo: 3 (from "cd foo", "cd foo/bar", "cd ./foo")
	// foo/bar: 1
	// ~/code: 2 (from "gc", "gp")
	// ~/code/projects: 1 (from "gp")
	// /Users/rgrouls/docs: 1

	expected := map[string]int{
		"foo":                 3,
		"foo/bar":             1,
		"~/code":              2,
		"~/code/projects":     1,
		"/Users/rgrouls/docs": 1,
	}

	for name, want := range expected {
		found := false
		for _, dir := range stats.TopDirectories {
			if dir.Name == name {
				found = true
				if dir.Count != want {
					t.Errorf("Dir '%s': expected %d, got %d", name, want, dir.Count)
				}
			}
		}
		if !found {
			t.Errorf("Dir '%s' not found in TopDirectories", name)
		}
	}
}
