package setup

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/kibetnathan/minjibot/internal/ports/dto"
)

// runRules posts a rules embed into the configured channel (referenced by name
// against the freshly-created channels map) and pins it so the rules are always
// visible. A failed pin is not fatal — the embed is still posted.
func (r *Runner) runRules(cfg *Config, channels map[string]string, steps *[]Step) {
	rc := cfg.Rules
	chID, ok := channels[rc.Channel]
	if !ok || chID == "" {
		*steps = append(*steps, Step{Section: "rules", Label: rc.Channel, Status: "failed", Detail: "channel was not created"})
		return
	}

	title := rc.Title
	if title == "" {
		title = "Server Rules"
	}
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: rc.Message,
		Color:       0x5865F2, // blurple; overridden by [rules] color below
	}
	if rc.Color != "" {
		if color, err := parseRoleColor(rc.Color); err == nil {
			embed.Color = color
		}
	}

	msg, err := r.s.ChannelMessageSendEmbed(chID, embed)
	if err != nil {
		*steps = append(*steps, Step{Section: "rules", Label: rc.Channel, Status: "failed", Detail: err.Error()})
		return
	}

	detail := fmt.Sprintf("rules embed posted to <#%s>", chID)
	if err := r.s.ChannelMessagePin(chID, msg.ID); err != nil {
		detail += fmt.Sprintf(" (pin skipped: %v)", err)
	} else {
		detail += " and pinned"
	}
	*steps = append(*steps, Step{Section: "rules", Label: rc.Channel, Status: "ok", Detail: detail})
}

// runLogging persists the guild's logging settings: the channel mod actions and
// deleted messages are posted to, and whether message content is captured so
// deletions can be reconstructed. It is a no-op if no settings repo is wired.
func (r *Runner) runLogging(ctx context.Context, guildID string, cfg *Config, channels map[string]string, steps *[]Step) {
	lc := cfg.Logging
	chID, ok := channels[lc.Channel]
	if !ok || chID == "" {
		*steps = append(*steps, Step{Section: "logging", Label: lc.Channel, Status: "failed", Detail: "channel was not created"})
		return
	}
	if r.settings == nil {
		*steps = append(*steps, Step{Section: "logging", Label: lc.Channel, Status: "failed", Detail: "settings repository is unavailable"})
		return
	}

	current, err := r.settings.Get(ctx, guildID)
	if err != nil {
		*steps = append(*steps, Step{Section: "logging", Label: lc.Channel, Status: "failed", Detail: fmt.Sprintf("read current settings: %v", err)})
		return
	}

	_, err = r.settings.Upsert(ctx, dto.UpsertGuildSettingsParams{
		GuildID:               guildID,
		Prefix:                current.Prefix,
		Language:              current.Language,
		AutoModerationEnabled: current.AutoModerationEnabled,
		LoggingChannelID:      chID,
		MessageLoggingEnabled: lc.DeletedMessages,
	})
	if err != nil {
		*steps = append(*steps, Step{Section: "logging", Label: lc.Channel, Status: "failed", Detail: err.Error()})
		return
	}

	deleted := "off"
	if lc.DeletedMessages {
		deleted = "on"
	}
	*steps = append(*steps, Step{Section: "logging", Label: lc.Channel, Status: "ok", Detail: fmt.Sprintf("log channel set to <#%s>; deleted-message content capture %s", chID, deleted)})
}
