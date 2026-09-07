package api

import (
	"context"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	authsvc "github.com/kibetnathan/minjibot/internal/services/auth"
)

// modPermBits are the server-wide permissions that grant moderation power.
// It mirrors the bot commands' defaultModPerm plus ManageGuild: viewing a
// guild's dashboard data is gated on the same abilities that let someone
// moderate or manage that guild.
const modPermBits = discordgo.PermissionManageGuild |
	discordgo.PermissionManageMessages |
	discordgo.PermissionKickMembers |
	discordgo.PermissionBanMembers |
	discordgo.PermissionManageRoles |
	discordgo.PermissionManageChannels |
	discordgo.PermissionAdministrator

// guildAuthzTTL is how long a user's per-guild permission snapshot is reused
// before it is re-fetched from Discord.
const guildAuthzTTL = 60 * time.Second

// guildAuthz grants dashboard access per guild: a user may view a guild's data
// only if they hold moderation permissions in that guild on Discord. The
// permissions come from the user's own OAuth session (GET /users/@me/guilds),
// never from a global allowlist — which is what lets the dashboard serve a
// bot installed in a large number of independent guilds.
type guildAuthz struct {
	mu    sync.Mutex
	cache map[string]guildPermEntry

	// fetchGuilds calls Discord /users/@me/guilds; overridden in tests.
	fetchGuilds func(ctx context.Context, accessToken string) ([]authsvc.Guild, error)
}

// guildPermEntry caches one user's per-guild permission bits and guild names.
type guildPermEntry struct {
	perms map[string]int64  // guildID → guild-level permission bits
	names map[string]string // guildID → guild name
	at    time.Time
}

// newGuildAuthz builds a per-guild authorizer that resolves permissions via
// the given Discord OAuth client.
func newGuildAuthz(oauth *authsvc.DiscordOAuth) *guildAuthz {
	return &guildAuthz{
		cache: make(map[string]guildPermEntry),
		fetchGuilds: func(ctx context.Context, accessToken string) ([]authsvc.Guild, error) {
			return oauth.UserGuilds(ctx, accessToken)
		},
	}
}

// hasModPerms reports whether guild-level permission bits include any
// moderation permission. Administrator is included in the mask, so holding it
// always authorizes.
func hasModPerms(bits int64) bool {
	return bits&modPermBits != 0
}

// guildPerms returns the user's per-guild permission bits, memoized for
// guildAuthzTTL to avoid hammering Discord on every dashboard request.
func (g *guildAuthz) guildPerms(ctx context.Context, userID, accessToken string) (map[string]int64, error) {
	g.mu.Lock()
	if e, ok := g.cache[userID]; ok && time.Since(e.at) < guildAuthzTTL {
		g.mu.Unlock()
		return e.perms, nil
	}
	g.mu.Unlock()

	guilds, err := g.fetchGuilds(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	perms := make(map[string]int64, len(guilds))
	for _, gd := range guilds {
		perms[gd.ID] = int64(gd.Permissions)
	}

	names := make(map[string]string, len(guilds))
	for _, gd := range guilds {
		names[gd.ID] = gd.Name
	}

	g.mu.Lock()
	g.cache[userID] = guildPermEntry{perms: perms, names: names, at: time.Now()}
	g.mu.Unlock()
	return perms, nil
}

// guildNames returns each guild's display name for the session user, using the
// same (cached) Discord guild fetch as guildPerms.
func (g *guildAuthz) guildNames(ctx context.Context, userID, accessToken string) (map[string]string, error) {
	if _, err := g.guildPerms(ctx, userID, accessToken); err != nil {
		return nil, err
	}
	g.mu.Lock()
	e, ok := g.cache[userID]
	g.mu.Unlock()
	if !ok {
		return nil, nil
	}
	return e.names, nil
}

// canView reports whether the session user holds moderation permissions in a
// single guild.
func (g *guildAuthz) canView(ctx context.Context, userID, accessToken, guildID string) (bool, error) {
	if userID == "" || accessToken == "" || guildID == "" {
		return false, nil
	}
	perms, err := g.guildPerms(ctx, userID, accessToken)
	if err != nil {
		return false, err
	}
	if bits, ok := perms[guildID]; ok {
		return hasModPerms(bits), nil
	}
	return false, nil
}

// authorizableGuilds returns the set of guilds the user holds moderation
// permissions in.
func (g *guildAuthz) authorizableGuilds(ctx context.Context, userID, accessToken string) (map[string]struct{}, error) {
	perms, err := g.guildPerms(ctx, userID, accessToken)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{})
	for id, bits := range perms {
		if hasModPerms(bits) {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

// hasAny reports whether the user holds moderation permissions in at least one
// guild. Used to answer "is this account a dashboard moderator (somewhere)".
func (g *guildAuthz) hasAny(ctx context.Context, userID, accessToken string) (bool, error) {
	auth, err := g.authorizableGuilds(ctx, userID, accessToken)
	if err != nil {
		return false, err
	}
	return len(auth) > 0, nil
}
