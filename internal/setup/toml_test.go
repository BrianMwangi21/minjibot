package setup

import (
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	doc := `
[server]
name = "Neroville"

[[roles]]
name = "Member"
color = "#7289da"
permissions = ["send_messages", "view_channel"]

[[roles]]
name = "Mod"
permissions = ["manage_messages", "kick_members"]

[[channels]]
name = "info"
type = "text"
topic = "Welcome"

[[channels]]
name = "rules"
parent = "info"

[[channels]]
name = "general"

[[channels]]
name = "locked"
permission_overwrites = [ { target = "role:Member", allow = ["send"], deny = ["invite"] } ]

[[emojis]]
name = "pepe"
url = "https://example.com/pepe.png"

[[stickers]]
name = "wave"
url = "https://example.com/wave.png"

[community]
rules_channel = "rules"
updates_channel = "general"

[onboarding]
enabled = true
default_channels = ["general"]
[[onboarding.prompts]]
title = "Pick a role"
single_select = true
[[onboarding.prompts.options]]
title = "Member"
roles = ["Member"]
[[onboarding.prompts.options]]
title = "General"
channels = ["general"]

[[commands]]
cmd = "channel"
args = "setperm #general Member deny send"
`
	cfg, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Server.Name != "Neroville" {
		t.Errorf("server name = %q", cfg.Server.Name)
	}
	if len(cfg.Roles) != 2 || len(cfg.Channels) != 4 || len(cfg.Emojis) != 1 || len(cfg.Stickers) != 1 {
		t.Fatalf("unexpected section sizes")
	}
	if cfg.Onboarding == nil || len(cfg.Onboarding.Prompts) != 1 {
		t.Fatalf("onboarding not parsed")
	}
}

func TestParseDuplicateName(t *testing.T) {
	doc := `
[[roles]]
name = "Member"
[[roles]]
name = "Member"
`
	if _, err := Parse([]byte(doc)); err == nil {
		t.Fatal("expected duplicate role error")
	} else if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseUnknownPermission(t *testing.T) {
	doc := `
[[roles]]
name = "Member"
permissions = ["does_not_exist"]
`
	if _, err := Parse([]byte(doc)); err == nil {
		t.Fatal("expected unknown permission error")
	}
}

func TestParseUnknownChannelType(t *testing.T) {
	doc := `
[[channels]]
name = "x"
type = "not-a-type"
`
	if _, err := Parse([]byte(doc)); err == nil {
		t.Fatal("expected unknown channel type error")
	}
}

func TestParseBadJsonStyle(t *testing.T) {
	if _, err := Parse([]byte("this is: not toml")); err == nil {
		t.Fatal("expected TOML parse error")
	}
}

func TestParseRulesAndLogging(t *testing.T) {
	doc := `
[[channels]]
name = "rules"

[[channels]]
name = "mod-logs"

[rules]
channel = "rules"
title = "Server Rules"
color = "#ff0000"
message = "Be nice."

[logging]
channel = "mod-logs"
deleted_messages = true
`
	cfg, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Rules == nil || cfg.Rules.Channel != "rules" || cfg.Rules.Message != "Be nice." {
		t.Fatalf("rules not parsed: %+v", cfg.Rules)
	}
	if cfg.Logging == nil || cfg.Logging.Channel != "mod-logs" || !cfg.Logging.DeletedMessages {
		t.Fatalf("logging not parsed: %+v", cfg.Logging)
	}
}

func TestParseRulesBadChannel(t *testing.T) {
	doc := `
[[channels]]
name = "general"

[rules]
channel = "rules"
`
	if _, err := Parse([]byte(doc)); err == nil || !strings.Contains(err.Error(), `rules: channel "rules"`) {
		t.Fatalf("expected undefined rules channel error, got %v", err)
	}
}

func TestParseLoggingBadChannel(t *testing.T) {
	doc := `
[[channels]]
name = "general"

[logging]
channel = "mod-logs"
`
	if _, err := Parse([]byte(doc)); err == nil || !strings.Contains(err.Error(), `logging: channel "mod-logs"`) {
		t.Fatalf("expected undefined logging channel error, got %v", err)
	}
}

func TestParseRulesBadColor(t *testing.T) {
	doc := `
[[channels]]
name = "rules"

[rules]
channel = "rules"
color = "not-a-color"
`
	if _, err := Parse([]byte(doc)); err == nil || !strings.Contains(err.Error(), "rules:") {
		t.Fatalf("expected rules color error, got %v", err)
	}
}

func TestExpander(t *testing.T) {
	got := expandRefs("{role:Member} and {channel:general}", map[string]string{"Member": "r1"}, map[string]string{"general": "c1"})
	if !strings.Contains(got, "<@&r1>") || !strings.Contains(got, "<#c1>") {
		t.Fatalf("expandRefs = %q", got)
	}
}
