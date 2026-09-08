// Package setup implements a TOML-driven server provisioning runner. A server
// admin pastes a TOML document into the dashboard describing roles, channels,
// emojis, stickers, community/rules configuration, onboarding prompts, and
// arbitrary bot commands to run in order. The runner applies these against
// Discord through the bot's own session so existing permission checks, audit
// logging, and command handlers behave exactly as if a moderator ran them in
// the server.
package setup

import "fmt"

// Config is the parsed root document for a server setup file.
type Config struct {
	Server     ServerConfig      `toml:"server"`
	Rules      *RulesConfig      `toml:"rules"`
	Roles      []RoleConfig      `toml:"roles"`
	Channels   []ChannelConfig   `toml:"channels"`
	Emojis     []EmojiConfig     `toml:"emojis"`
	Stickers   []StickerConfig   `toml:"stickers"`
	Logging    *LoggingConfig    `toml:"logging"`
	Community  *CommunityConfig  `toml:"community"`
	Onboarding *OnboardingConfig `toml:"onboarding"`
	Commands   []CommandConfig   `toml:"commands"`
}

// RulesConfig posts (and pins) a rules embed into a channel defined by the
// document.
type RulesConfig struct {
	Channel string `toml:"channel"`
	Title   string `toml:"title"`
	Color   string `toml:"color"`
	Message string `toml:"message"`
}

// LoggingConfig wires the guild's logging settings: the channel mod actions and
// deleted messages are posted to, and whether message content is stored so
// deletions can be reconstructed.
type LoggingConfig struct {
	Channel         string `toml:"channel"`
	DeletedMessages bool   `toml:"deleted_messages"`
}

// ServerConfig renames the server. All fields are optional.
type ServerConfig struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
}

// RoleConfig describes a role to create.
type RoleConfig struct {
	Name        string   `toml:"name"`
	Color       string   `toml:"color"`
	Hoist       bool     `toml:"hoist"`
	Mentionable bool     `toml:"mentionable"`
	Permissions []string `toml:"permissions"`
}

// ChannelConfig describes a channel (or category) to create.
type ChannelConfig struct {
	Name                 string                      `toml:"name"`
	Type                 string                      `toml:"type"`
	Topic                string                      `toml:"topic"`
	NSFW                 bool                        `toml:"nsfw"`
	Parent               string                      `toml:"parent"`
	PermissionOverwrites []PermissionOverwriteConfig `toml:"permission_overwrites"`
}

// PermissionOverwriteConfig grants or denies channel permissions for a role
// (target "role:Name") or a single member (target "user:DiscordID").
type PermissionOverwriteConfig struct {
	Target string   `toml:"target"`
	Allow  []string `toml:"allow"`
	Deny   []string `toml:"deny"`
}

// EmojiConfig uploads an emoji from a public URL.
type EmojiConfig struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}

// StickerConfig uploads a sticker from a public URL.
type StickerConfig struct {
	Name        string `toml:"name"`
	URL         string `toml:"url"`
	Description string `toml:"description"`
	Tags        string `toml:"tags"`
}

// CommunityConfig enables the server's Community mode by pointing it at the
// rules and announcements channels created by [[channels]].
type CommunityConfig struct {
	RulesChannel      string `toml:"rules_channel"`
	UpdatesChannel    string `toml:"updates_channel"`
	VerificationLevel string `toml:"verification_level"`
	ContentFilter     string `toml:"content_filter"`
}

// OnboardingConfig configures the onboarding flow. Prompt options may select
// roles and channels by the names defined in [[roles]] / [[channels]].
type OnboardingConfig struct {
	Enabled         bool           `toml:"enabled"`
	DefaultChannels []string       `toml:"default_channels"`
	Prompts         []PromptConfig `toml:"prompts"`
}

// PromptConfig is a single onboarding prompt.
type PromptConfig struct {
	Title        string               `toml:"title"`
	Type         string               `toml:"type"`
	Required     bool                 `toml:"required"`
	SingleSelect bool                 `toml:"single_select"`
	InOnboarding bool                 `toml:"in_onboarding"`
	Options      []PromptOptionConfig `toml:"options"`
}

// PromptOptionConfig is one selectable option in a prompt.
type PromptOptionConfig struct {
	Title       string   `toml:"title"`
	Description string   `toml:"description"`
	Roles       []string `toml:"roles"`
	Channels    []string `toml:"channels"`
}

// CommandConfig runs a bot command via the existing prefix handler, e.g.
// cmd = "channel", args = "setperm #rules Member deny send_messages".
type CommandConfig struct {
	Cmd  string `toml:"cmd"`
	Args string `toml:"args"`
}

// validate runs structural checks that need no Discord access: required names,
// permission spellings, reference availability, and duplicate declarations.
func (c *Config) validate() error {
	counts := map[string]int{}
	roleNames := map[string]struct{}{}
	for _, r := range c.Roles {
		counts["role:"+r.Name]++
		roleNames[r.Name] = struct{}{}
		if r.Name == "" {
			return fmt.Errorf("roles: name is required")
		}
		for _, p := range r.Permissions {
			if _, ok := rolePermBits[p]; !ok {
				return fmt.Errorf("role %q: unknown permission %q", r.Name, p)
			}
		}
		if _, err := parseRoleColor(r.Color); err != nil {
			return fmt.Errorf("role %q: %w", r.Name, err)
		}
	}

	chanNames := map[string]struct{}{}
	for _, ch := range c.Channels {
		counts["channel:"+ch.Name]++
		chanNames[ch.Name] = struct{}{}
		if ch.Name == "" {
			return fmt.Errorf("channels: name is required")
		}
		if ch.Type != "" {
			if _, err := parseChannelType(ch.Type); err != nil {
				return fmt.Errorf("channel %q: %w", ch.Name, err)
			}
		}
		if ch.Parent != "" {
			if _, ok := chanNames[ch.Parent]; !ok {
				return fmt.Errorf("channel %q: parent %q must be a channel defined earlier (or itself)", ch.Name, ch.Parent)
			}
		}
		for _, o := range ch.PermissionOverwrites {
			for _, p := range append(append([]string{}, o.Allow...), o.Deny...) {
				if _, ok := channelPermBits[p]; !ok {
					return fmt.Errorf("channel %q overwrite: unknown permission %q", ch.Name, p)
				}
			}
		}
	}

	for _, e := range c.Emojis {
		counts["emoji:"+e.Name]++
		if e.Name == "" || e.URL == "" {
			return fmt.Errorf("emojis: name and url are required")
		}
	}
	for _, st := range c.Stickers {
		counts["sticker:"+st.Name]++
		if st.Name == "" || st.URL == "" {
			return fmt.Errorf("stickers: name and url are required")
		}
	}

	if c.Community != nil {
		if c.Community.RulesChannel != "" {
			if _, ok := chanNames[c.Community.RulesChannel]; !ok {
				return fmt.Errorf("community: rules_channel %q is not a defined channel", c.Community.RulesChannel)
			}
		}
		if c.Community.UpdatesChannel != "" {
			if _, ok := chanNames[c.Community.UpdatesChannel]; !ok {
				return fmt.Errorf("community: updates_channel %q is not a defined channel", c.Community.UpdatesChannel)
			}
		}
		if c.Community.VerificationLevel != "" && parseVerificationLevel(c.Community.VerificationLevel) == nil {
			return fmt.Errorf("community: unknown verification_level %q", c.Community.VerificationLevel)
		}
		if c.Community.ContentFilter != "" && parseContentFilter(c.Community.ContentFilter) == nil {
			return fmt.Errorf("community: unknown content_filter %q", c.Community.ContentFilter)
		}
	}

	if c.Rules != nil {
		if c.Rules.Channel == "" {
			return fmt.Errorf("rules: channel is required")
		}
		if _, ok := chanNames[c.Rules.Channel]; !ok {
			return fmt.Errorf("rules: channel %q is not a defined channel", c.Rules.Channel)
		}
		if c.Rules.Color != "" {
			if _, err := parseRoleColor(c.Rules.Color); err != nil {
				return fmt.Errorf("rules: %w", err)
			}
		}
	}

	if c.Logging != nil {
		if c.Logging.Channel == "" {
			return fmt.Errorf("logging: channel is required")
		}
		if _, ok := chanNames[c.Logging.Channel]; !ok {
			return fmt.Errorf("logging: channel %q is not a defined channel", c.Logging.Channel)
		}
	}

	if c.Onboarding != nil {
		for _, d := range c.Onboarding.DefaultChannels {
			if _, ok := chanNames[d]; !ok {
				return fmt.Errorf("onboarding: default channel %q is not a defined channel", d)
			}
		}
		for i, p := range c.Onboarding.Prompts {
			if p.Title == "" {
				return fmt.Errorf("onboarding: prompt %d requires a title", i)
			}
			for _, r := range p.RolesReferenced() {
				if _, ok := roleNames[r]; !ok {
					return fmt.Errorf("onboarding prompt %q: role %q is not a defined role", p.Title, r)
				}
			}
			for _, ch := range p.ChannelsReferenced() {
				if _, ok := chanNames[ch]; !ok {
					return fmt.Errorf("onboarding prompt %q: channel %q is not a defined channel", p.Title, ch)
				}
			}
		}
	}

	for name, n := range counts {
		if n > 1 {
			return fmt.Errorf("duplicate %s declared %d times", name, n)
		}
	}
	return nil
}

// RolesReferenced returns the names of roles referenced by a prompt's options.
func (p *PromptConfig) RolesReferenced() []string {
	var out []string
	for _, o := range p.Options {
		for _, r := range o.Roles {
			if r != "@everyone" {
				out = append(out, r)
			}
		}
	}
	return out
}

// ChannelsReferenced returns the names of channels referenced by a prompt's
// options.
func (p *PromptConfig) ChannelsReferenced() []string {
	var out []string
	for _, o := range p.Options {
		out = append(out, o.Channels...)
	}
	return out
}
