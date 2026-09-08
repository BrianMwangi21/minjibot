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

	// Discord refuses to turn onboarding on unless at least 7 default
	// channels are selected and at least 5 of them let @everyone view and
	// send. Pad the selection with other channels the document created, or
	// auto-create extras so a sparse document can still enable onboarding.
	if oc.Enabled {
		defaultIDs = r.fillOnboardingDefaults(ctx, guildID, cfg, channels, defaultIDs, steps)
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

// minOnboardingDefaultsTotal and minOnboardingDefaultsSendable mirror Discord's
// prerequisites for enabling onboarding: at least 7 default channels selected,
// of which at least 5 must let @everyone view and send messages.
const (
	minOnboardingDefaultsTotal    = 7
	minOnboardingDefaultsSendable = 5
)

// fillOnboardingDefaults grows the onboarding default-channel selection to
// Discord's minimums. It first prefers channels the document created (those
// where @everyone can send first, then view-only ones), and only if the
// document still comes up short does it create extra text channels — matching
// Discord's own onboarding wizard, which offers to create the required
// channels. Every auto-created channel is reported as a Step.
func (r *Runner) fillOnboardingDefaults(ctx context.Context, guildID string, cfg *Config, channels map[string]string, defaultIDs []string, steps *[]Step) []string {
	final := append([]string{}, defaultIDs...)
	inList := make(map[string]bool, len(final))
	for _, id := range final {
		inList[id] = true
	}

	sendable := make(map[string]bool, len(cfg.Channels))
	var sendPend, viewPend []string
	for _, cc := range cfg.Channels {
		id, ok := channels[cc.Name]
		if !ok {
			continue
		}
		view, send := channelEveryoneAccess(cc)
		if !view {
			continue
		}
		sendable[id] = send
		if inList[id] {
			continue
		}
		if send {
			sendPend = append(sendPend, id)
		} else {
			viewPend = append(viewPend, id)
		}
	}

	sendInList := 0
	for _, id := range final {
		if sendable[id] {
			sendInList++
		}
	}

	add := func(id string) {
		if inList[id] {
			return
		}
		inList[id] = true
		final = append(final, id)
		if sendable[id] {
			sendInList++
		}
	}

	for _, id := range sendPend {
		if len(final) >= minOnboardingDefaultsTotal && sendInList >= minOnboardingDefaultsSendable {
			break
		}
		add(id)
	}
	for _, id := range viewPend {
		if len(final) >= minOnboardingDefaultsTotal {
			break
		}
		add(id)
	}

	// Still short? Create text channels until the requirement is met.
	usedNames := make(map[string]bool, len(cfg.Channels))
	for _, cc := range cfg.Channels {
		usedNames[cc.Name] = true
	}
	suffix := 0
	for len(final) < minOnboardingDefaultsTotal || sendInList < minOnboardingDefaultsSendable {
		name := nextOnboardingFillName(usedNames, suffix)
		if name == "" {
			break
		}
		suffix++
		ch, err := r.s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
			Name: name,
			Type: discordgo.ChannelTypeGuildText,
		})
		if err != nil {
			*steps = append(*steps, Step{
				Section: "onboarding",
				Label:   name,
				Status:  "failed",
				Detail:  fmt.Sprintf("auto-create default channel: %v", err),
			})
			break
		}
		usedNames[name] = true
		final = append(final, ch.ID)
		sendInList++
		*steps = append(*steps, Step{
			Section: "onboarding",
			Label:   name,
			Status:  "ok",
			Detail:  "auto-created to meet onboarding default-channel requirements",
		})
	}
	return final
}

// fillNames are the channel names used when onboarding needs extra defaults
// the document did not declare.
var fillNames = []string{"introductions", "chat", "general", "off-topic", "memes", "showcase", "random", "lounge", "community", "welcome"}

// nextOnboardingFillName picks the next unused fill name, falling back to
// "general-N" when the pool is exhausted.
func nextOnboardingFillName(used map[string]bool, suffix int) string {
	for _, name := range fillNames {
		if !used[name] {
			return name
		}
	}
	name := fmt.Sprintf("general-%d", suffix+1)
	if used[name] {
		return ""
	}
	return name
}

// channelEveryoneAccess classifies a document channel for the onboarding
// default-channel requirements: whether @everyone can view it and whether they
// can send messages in it. Voice channels and categories are viewable but never
// count as sendable.
func channelEveryoneAccess(cc ChannelConfig) (view, send bool) {
	view, send = true, true
	switch strings.ToLower(cc.Type) {
	case "voice", "vc", "category", "cat":
		send = false
	}
	for _, o := range cc.PermissionOverwrites {
		if !strings.EqualFold(strings.TrimSpace(o.Target), "role:@everyone") {
			continue
		}
		for _, p := range o.Deny {
			switch strings.ToLower(p) {
			case "view":
				view = false
			case "send":
				send = false
			}
		}
	}
	if !view {
		send = false
	}
	return view, send
}
