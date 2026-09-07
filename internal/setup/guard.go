package setup

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// maxProvisionChannels is the highest channel count a target server is expected
// to have before provisioning is treated as destructive. Discord creates a few
// channels by default, and provisioned documents stay comfortably below this.
const maxProvisionChannels = 12

// maxProvisionMembers is the highest non-bot member count a fresh server may
// have. Beyond it the server is clearly live and an unattended provisioning
// run would be disruptive.
const maxProvisionMembers = 20

// Guard inspects a target guild for signs that it is already in use before the
// runner provisions it. Replaying a setup document over an established server
// creates duplicate roles, channels, and settings, so the run endpoint refuses
// to touch servers that look live.
type Guard struct {
	s      *discordgo.Session
	selfID string // the bot's own user ID, excluded from the bot-partner count
}

// NewGuard builds a guard around the bot's session. The bot's own user ID is
// resolved lazily from the session state (or REST) so it is never counted as a
// third-party bot.
func NewGuard(s *discordgo.Session) *Guard {
	return &Guard{s: s}
}

// Reasons describes why a guild snapshot should not be provisioned. It also
// implements error so a non-empty guard result reads naturally as a rejection
// message.
type Reasons []string

// Error joins the individual signs into a readable message.
func (r Reasons) Error() string {
	out := ""
	for i, reason := range r {
		if i > 0 {
			out += "; "
		}
		out += reason
	}
	return out
}

// Check inspects the target guild and returns the signs that it is already set
// up, or nil if it looks like a fresh server. A non-nil error means the bot
// could not assess the guild at all.
func (g *Guard) Check(ctx context.Context, guildID string) (Reasons, error) {
	if g.s == nil {
		return nil, fmt.Errorf("bot session unavailable")
	}

	channels, err := g.s.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("loading channels: %w", err)
	}

	members, err := g.s.GuildMembers(guildID, "", 1000)
	if err != nil {
		return nil, fmt.Errorf("loading members: %w", err)
	}

	nonBots, otherBots := 0, 0
	selfID := g.selfUserID()
	for _, m := range members {
		if m.User == nil {
			continue
		}
		if m.User.Bot {
			if selfID == "" || m.User.ID != selfID {
				otherBots++
			}
			continue
		}
		nonBots++
	}

	return guardReasons(len(channels), otherBots, nonBots), nil
}

// selfUserID resolves the bot's own user ID, preferring the gateway state and
// falling back to REST.
func (g *Guard) selfUserID() string {
	if g.selfID != "" {
		return g.selfID
	}
	if g.s.State != nil && g.s.State.User != nil {
		g.selfID = g.s.State.User.ID
		return g.selfID
	}
	if u, err := g.s.User("@me"); err == nil {
		g.selfID = u.ID
		return g.selfID
	}
	return ""
}

// guardReasons is the pure sign logic: a server is already set up when it has
// a crowd of third-party bots alongside built-out channels, or more than
// maxProvisionMembers humans. Purely many-channel servers that the owner is
// still shaping are left alone so a real setup run keeps working.
func guardReasons(channels, otherBots, nonBots int) Reasons {
	var reasons Reasons
	if channels > maxProvisionChannels && otherBots >= 1 {
		reasons = append(reasons, fmt.Sprintf("already has %d channels and %d other bot(s)", channels, otherBots))
	}
	if nonBots > maxProvisionMembers {
		reasons = append(reasons, fmt.Sprintf("already has %d members", nonBots))
	}
	return reasons
}

// Err returns a summary of the reasons as an error, or nil if there are none.
func (r Reasons) err() error {
	if len(r) == 0 {
		return nil
	}
	return fmt.Errorf("server looks already set up: %s", r.Error())
}
