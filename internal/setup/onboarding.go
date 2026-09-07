package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// runCommunity enables Community mode by pointing the guild's rules and
// announcements channels at channels created by [[channels]] and raising the
// verification level where requested.
func (r *Runner) runCommunity(ctx context.Context, guildID string, cfg *Config, channels map[string]string, steps *[]Step) {
	cc := cfg.Community
	if cc.RulesChannel == "" && cc.UpdatesChannel == "" && cc.VerificationLevel == "" && cc.ContentFilter == "" {
		return
	}
	params := &discordgo.GuildParams{}
	if cc.RulesChannel != "" {
		params.RulesChannelID = channels[cc.RulesChannel]
	}
	if cc.UpdatesChannel != "" {
		params.PublicUpdatesChannelID = channels[cc.UpdatesChannel]
	}
	if lv := parseVerificationLevel(cc.VerificationLevel); lv != nil {
		params.VerificationLevel = lv
	}
	if cf := parseContentFilter(cc.ContentFilter); cf != nil {
		params.ExplicitContentFilter = int(*cf)
	}
	if _, err := r.s.GuildEdit(guildID, params); err != nil {
		*steps = append(*steps, Step{Section: "community", Label: "community", Status: "failed", Detail: err.Error()})
		return
	}
	*steps = append(*steps, Step{Section: "community", Label: "community", Status: "ok", Detail: "community mode enabled"})
}

// runOnboarding publishes the onboarding flow (default channels and prompts),
// resolving role/channel references by the names defined in the document.
func (r *Runner) runOnboarding(ctx context.Context, guildID string, cfg *Config, roles, channels map[string]string, steps *[]Step) {
	oc := cfg.Onboarding

	defaultIDs := make([]string, 0, len(oc.DefaultChannels))
	for _, name := range oc.DefaultChannels {
		defaultIDs = append(defaultIDs, channels[name])
	}

	prompts := make([]discordgo.GuildOnboardingPrompt, 0, len(oc.Prompts))
	for _, p := range oc.Prompts {
		options := make([]discordgo.GuildOnboardingPromptOption, 0, len(p.Options))
		for _, o := range p.Options {
			roleIDs := make([]string, 0, len(o.Roles))
			for _, name := range o.Roles {
				if name == "@everyone" {
					roleIDs = append(roleIDs, guildID)
					continue
				}
				roleIDs = append(roleIDs, roles[name])
			}
			chanIDs := make([]string, 0, len(o.Channels))
			for _, name := range o.Channels {
				chanIDs = append(chanIDs, channels[name])
			}
			options = append(options, discordgo.GuildOnboardingPromptOption{
				Title:       o.Title,
				Description: o.Description,
				RoleIDs:     roleIDs,
				ChannelIDs:  chanIDs,
			})
		}
		single := p.SingleSelect
		if strings.EqualFold(p.Type, "single") {
			single = true
		}
		inOnboarding := true
		if !p.InOnboarding {
			inOnboarding = false
		}
		prompts = append(prompts, discordgo.GuildOnboardingPrompt{
			ID:           "0",
			Title:        p.Title,
			Type:         discordgo.GuildOnboardingPromptTypeMultipleChoice,
			SingleSelect: single,
			Required:     p.Required,
			InOnboarding: inOnboarding,
			Options:      options,
		})
	}

	enabled := oc.Enabled
	mode := discordgo.GuildOnboardingModeDefault
	onboarding := &discordgo.GuildOnboarding{
		GuildID:           guildID,
		Enabled:           &enabled,
		Mode:              &mode,
		DefaultChannelIDs: defaultIDs,
		Prompts:           &prompts,
	}
	if _, err := r.s.GuildOnboardingEdit(guildID, onboarding); err != nil {
		*steps = append(*steps, Step{Section: "onboarding", Label: "onboarding", Status: "failed", Detail: err.Error()})
		return
	}
	*steps = append(*steps, Step{Section: "onboarding", Label: "onboarding", Status: "ok", Detail: "onboarding flow updated"})
}

// findRoleID looks up a role by name (case-insensitive) among a guild's
// existing roles.
func findRoleID(s *discordgo.Session, guildID, name string) (*discordgo.Role, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		if strings.EqualFold(r.Name, name) {
			return r, nil
		}
	}
	return nil, fmt.Errorf("role %q not found in this server", name)
}
