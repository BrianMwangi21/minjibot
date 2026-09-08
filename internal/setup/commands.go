package setup

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// runCommands executes each [[commands]] entry through the bot's normal
// CommandHandler, so permission checks, audit logging, and reply embeds behave
// exactly as if a moderator typed the command in the notify channel. `{role:Name}`
// and `{channel:Name}` tokens in args are replaced with their mentions.
func (r *Runner) runCommands(ctx context.Context, guildID string, cfg *Config, roles, channels map[string]string, notify Notify, steps *[]Step) {
	if r.commands == nil {
		*steps = append(*steps, Step{Section: "commands", Label: "commands", Status: "failed", Detail: "command passthrough unavailable (bot not connected)"})
		return
	}
	for _, cc := range cfg.Commands {
		cmd := strings.ToLower(strings.TrimSpace(cc.Cmd))
		if cmd == "" {
			*steps = append(*steps, Step{Section: "commands", Label: cc.Args, Status: "failed", Detail: "cmd is required"})
			continue
		}
		args := strings.Fields(expandRefs(cc.Args, roles, channels))

		realMsg := &discordgo.Message{
			GuildID:   guildID,
			ChannelID: notify.ChannelID,
			Author:    &discordgo.User{ID: notify.AuthorID, Username: notify.AuthorName},
		}
		m := &discordgo.MessageCreate{Message: realMsg}
		if err := r.commands.Handle(ctx, r.s, m, cmd, args); err != nil {
			*steps = append(*steps, Step{Section: "commands", Label: cmd, Status: "failed", Detail: err.Error()})
			continue
		}
		*steps = append(*steps, Step{Section: "commands", Label: cmd, Status: "ok", Detail: cc.Args})
	}
}

// expandRefs replaces {role:Name} and {channel:Name} tokens with Discord
// mentions. Unresolved names pass through untouched so the command handler can
// produce its own "not found" message.
func expandRefs(raw string, roles, channels map[string]string) string {
	for name, id := range roles {
		raw = strings.ReplaceAll(raw, "{role:"+name+"}", "<@&"+id+">")
	}
	for name, id := range channels {
		raw = strings.ReplaceAll(raw, "{channel:"+name+"}", "<#"+id+">")
	}
	return raw
}
