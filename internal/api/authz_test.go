package api

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	authsvc "github.com/kibetnathan/minjibot/internal/services/auth"
)

// fakeGuildFetch returns a fetchGuilds func that always returns guilds.
func fakeGuildFetch(guilds []authsvc.Guild) func(context.Context, string) ([]authsvc.Guild, error) {
	return func(context.Context, string) ([]authsvc.Guild, error) { return guilds, nil }
}

func TestHasModPerms(t *testing.T) {
	cases := []struct {
		name string
		bits int64
		want bool
	}{
		{"manager", discordgo.PermissionManageGuild, true},
		{"administrator dominates", discordgo.PermissionAdministrator, true},
		{"manage messages counts", discordgo.PermissionManageMessages, true},
		{"ban makes you a mod", discordgo.PermissionBanMembers, true},
		{"plain member denied", discordgo.PermissionSendMessages, false},
		{"zero denied", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasModPerms(c.bits); got != c.want {
				t.Errorf("hasModPerms(%d) = %v, want %v", c.bits, got, c.want)
			}
		})
	}
}

func TestGuildAuthz_CanView(t *testing.T) {
	a := &guildAuthz{
		cache: make(map[string]guildPermEntry),
		fetchGuilds: fakeGuildFetch([]authsvc.Guild{
			{ID: "g1", Name: "Mod Hub", Permissions: discordgo.PermissionManageGuild},
			{ID: "g2", Name: "Member Only", Permissions: discordgo.PermissionSendMessages},
		}),
	}

	ok, err := a.canView(context.Background(), "u1", "tok1", "g1")
	if err != nil {
		t.Fatalf("canView: %v", err)
	}
	if !ok {
		t.Error("expected g1 to be viewable (Manage Guild)")
	}

	ok, err = a.canView(context.Background(), "u1", "tok1", "g2")
	if err != nil {
		t.Fatalf("canView: %v", err)
	}
	if ok {
		t.Error("expected g2 to be not viewable (plain member)")
	}

	ok, err = a.canView(context.Background(), "u1", "tok1", "g3")
	if err != nil {
		t.Fatalf("canView: %v", err)
	}
	if ok {
		t.Error("expected a guild the user is not in to be not viewable")
	}
}

func TestGuildAuthz_AuthorizableGuilds_FiltersByPerm(t *testing.T) {
	a := &guildAuthz{
		cache: make(map[string]guildPermEntry),
		fetchGuilds: fakeGuildFetch([]authsvc.Guild{
			{ID: "a", Permissions: discordgo.PermissionAdministrator},
			{ID: "b", Permissions: discordgo.PermissionKickMembers},
			{ID: "c", Permissions: discordgo.PermissionViewChannel},
		}),
	}

	got, err := a.authorizableGuilds(context.Background(), "u1", "tok1")
	if err != nil {
		t.Fatalf("authorizableGuilds: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 authorizable guilds, got %d", len(got))
	}
	for _, id := range []string{"a", "b"} {
		if _, ok := got[id]; !ok {
			t.Errorf("expected %s to be authorizable", id)
		}
	}
	if _, ok := got["c"]; ok {
		t.Error("expected c (plain member) to be excluded")
	}
}

func TestGuildAuthz_RejectsMissingInputs(t *testing.T) {
	a := &guildAuthz{
		cache:       make(map[string]guildPermEntry),
		fetchGuilds: fakeGuildFetch(nil),
	}
	ok, err := a.canView(context.Background(), "", "", "g1")
	if err != nil {
		t.Fatalf("canView: %v", err)
	}
	if ok {
		t.Error("expected empty inputs to be denied")
	}

	ok, err = a.hasAny(context.Background(), "u1", "")
	if err != nil {
		t.Fatalf("hasAny: %v", err)
	}
	if ok {
		t.Error("expected blank token to deny hasAny")
	}
}

func TestGuildAuthz_CacheHitsBeforeRefetch(t *testing.T) {
	// The fake fetch counts calls: a second canView for the same user within
	// the TTL must reuse the cached snapshot instead of calling Discord again.
	calls := 0
	a := &guildAuthz{
		cache: make(map[string]guildPermEntry),
		fetchGuilds: func(context.Context, string) ([]authsvc.Guild, error) {
			calls++
			return []authsvc.Guild{{ID: "g1", Permissions: discordgo.PermissionAdministrator}}, nil
		},
	}

	for i := 0; i < 2; i++ {
		if _, err := a.canView(context.Background(), "u1", "tok1", "g1"); err != nil {
			t.Fatalf("canView: %v", err)
		}
	}
	if calls != 1 {
		t.Errorf("expected fetchGuilds to be called once (cached), got %d calls", calls)
	}
}
