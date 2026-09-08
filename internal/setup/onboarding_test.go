package setup

import (
	"testing"
)

func TestChannelEveryoneAccess(t *testing.T) {
	cases := []struct {
		name     string
		cc       ChannelConfig
		wantView bool
		wantSend bool
	}{
		{"open text", ChannelConfig{Name: "chat", Type: "text"}, true, true},
		{"open announcement", ChannelConfig{Name: "news", Type: "announcement"}, true, true},
		{"open forum", ChannelConfig{Name: "help", Type: "forum"}, true, true},
		{"voice is not sendable", ChannelConfig{Name: "vc", Type: "voice"}, true, false},
		{"category is not sendable", ChannelConfig{Name: "cat", Type: "category"}, true, false},
		{
			"view-only rules",
			ChannelConfig{Name: "rules", Type: "text", PermissionOverwrites: []PermissionOverwriteConfig{
				{Target: "role:@everyone", Allow: []string{"view", "history"}, Deny: []string{"send"}},
			}},
			true, false,
		},
		{
			"hidden channel",
			ChannelConfig{Name: "staff", Type: "text", PermissionOverwrites: []PermissionOverwriteConfig{
				{Target: "role:@everyone", Deny: []string{"view"}},
				{Target: "role:Staff", Allow: []string{"view", "send"}},
			}},
			false, false,
		},
		{
			"non-everyone overwrite ignored",
			ChannelConfig{Name: "lounge", Type: "text", PermissionOverwrites: []PermissionOverwriteConfig{
				{Target: "role:Member", Allow: []string{"send"}},
			}},
			true, true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view, send := channelEveryoneAccess(tc.cc)
			if view != tc.wantView {
				t.Fatalf("view = %v, want %v", view, tc.wantView)
			}
			if send != tc.wantSend {
				t.Fatalf("send = %v, want %v", send, tc.wantSend)
			}
		})
	}
}

// TestFillOnboardingDefaultsPadsFromExisting covers the branch that reaches the
// minimums using only channels the document created, so no Discord calls happen
// and a nil session is safe.
func TestFillOnboardingDefaultsPadsFromExisting(t *testing.T) {
	r := &Runner{} // nil session is fine: nothing needs to be created

	channels := map[string]string{}
	var cc []ChannelConfig
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	for _, n := range names {
		cc = append(cc, ChannelConfig{Name: n, Type: "text"})
		channels[n] = "id-" + n
	}

	cfg := &Config{Channels: cc}
	got := r.fillOnboardingDefaults(nil, "guild", cfg, channels, []string{"id-a"}, &[]Step{})

	if len(got) != 7 {
		t.Fatalf("len = %d, want 7", len(got))
	}
	if got[0] != "id-a" {
		t.Fatalf("first default = %q, want doc default id-a", got[0])
	}
	seen := map[string]bool{}
	for _, id := range got {
		if seen[id] {
			t.Fatalf("duplicate default %q", id)
		}
		seen[id] = true
	}
}

// TestFillOnboardingDefaultsKeepsOversizedList keeps an oversized doc default
// list intact.
func TestFillOnboardingDefaultsKeepsOversizedList(t *testing.T) {
	r := &Runner{}
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	channels := map[string]string{}
	var cc []ChannelConfig
	for _, n := range names {
		channels[n] = "id-" + n
		cc = append(cc, ChannelConfig{Name: n, Type: "text"})
	}
	cfg := &Config{Channels: cc}
	defaults := make([]string, 0, len(names))
	for _, n := range names {
		defaults = append(defaults, channels[n])
	}
	got := r.fillOnboardingDefaults(nil, "guild", cfg, channels, defaults, &[]Step{})
	if len(got) != 8 {
		t.Fatalf("len = %d, want 8 (unchanged)", len(got))
	}
}
