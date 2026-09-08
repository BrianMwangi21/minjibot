package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kibetnathan/minjibot/internal/setup"
)

// TestSetupTemplatesValidate parses every template in the dashboard's
// setupTemplates.ts with the real TOML decoder and structural validator, so a
// template can't drift out of sync with what the runner accepts.
func TestSetupTemplatesValidate(t *testing.T) {
	path := filepath.Join("..", "dashboard", "minji-bot", "src", "data", "setupTemplates.ts")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read templates: %v", err)
	}

	var templates []string
	var buf []string
	inTemplate := false
	for _, line := range strings.Split(string(src), "\n") {
		if idx := strings.Index(line, "toml: `"); idx >= 0 {
			rest := line[idx+len("toml: `"):]
			inTemplate = true
			buf = []string{}
			if c := strings.LastIndex(rest, "`,"); c >= 0 {
				if c > 0 {
					templates = append(templates, rest[:c])
				}
				inTemplate = false
				continue
			}
			if rest != "" {
				buf = append(buf, rest)
			}
			continue
		}
		if !inTemplate {
			continue
		}
		if c := strings.LastIndex(line, "`,"); c >= 0 {
			if c > 0 {
				buf = append(buf, line[:c])
			}
			templates = append(templates, strings.Join(buf, "\n"))
			inTemplate = false
			continue
		}
		buf = append(buf, line)
	}

	if len(templates) < 6 {
		t.Fatalf("expected at least 6 templates, found %d", len(templates))
	}

	for i, doc := range templates {
		if err := setup.Validate([]byte(doc)); err != nil {
			t.Fatalf("template %d failed validation: %v\n%s", i, err, doc)
		}
	}
}
