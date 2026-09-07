package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
)

// CommandRunner is the subset of the bot's command handler the runner needs:
// run a prefix command as if it had been typed into a Discord channel.
type CommandRunner interface {
	Handle(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, cmd string, args []string) error
}

// Notify describes where command passthrough results are reported and whose
// permissions are evaluated.
type Notify struct {
	ChannelID  string
	AuthorID   string
	AuthorName string
}

// Step is one provisioning action and its outcome.
type Step struct {
	Section string `json:"section"`
	Label   string `json:"label"`
	Status  string `json:"status"` // ok | skipped | failed
	Detail  string `json:"detail"`
}

// Runner provisions a guild from a setup Config.
type Runner struct {
	s        *discordgo.Session
	commands CommandRunner
}

// NewRunner builds a runner. commands may be nil, in which case the
// [[commands]] section is rejected with a clear error.
func NewRunner(s *discordgo.Session, commands CommandRunner) *Runner {
	return &Runner{s: s, commands: commands}
}

// Parse decodes and validates a setup document. Parse errors carry the TOML
// line so the dashboard can point at the offending entry.
func Parse(doc []byte) (*Config, error) {
	var cfg Config
	if _, err := toml.Decode(string(doc), &cfg); err != nil {
		return nil, fmt.Errorf("invalid TOML: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid setup document: %w", err)
	}
	return &cfg, nil
}

// Validate runs the same structural checks as Parse without requiring Discord.
// It is used by the dry-run endpoint.
func Validate(doc []byte) error {
	_, err := Parse(doc)
	return err
}

// Run executes each section in config order, appending one Step per item.
// Failures are recorded in the returned steps rather than aborting the whole
// run, so a bad role doesn't hide a good channel.
func (r *Runner) Run(ctx context.Context, guildID string, cfg *Config, notify Notify) []Step {
	steps := make([]Step, 0, 4+len(cfg.Roles)+len(cfg.Channels)+len(cfg.Emojis)+len(cfg.Stickers)+len(cfg.Commands))

	r.runServer(ctx, guildID, cfg, &steps)

	roles := r.runRoles(ctx, guildID, cfg, &steps)
	channels := r.runChannels(ctx, guildID, cfg, roles, &steps)

	if len(cfg.Emojis) > 0 {
		r.runEmojis(ctx, guildID, cfg, &steps)
	}
	if len(cfg.Stickers) > 0 {
		r.runStickers(ctx, guildID, cfg, &steps)
	}
	if cfg.Community != nil {
		r.runCommunity(ctx, guildID, cfg, channels, &steps)
	}
	if cfg.Onboarding != nil {
		r.runOnboarding(ctx, guildID, cfg, roles, channels, &steps)
	}
	if len(cfg.Commands) > 0 {
		r.runCommands(ctx, guildID, cfg, roles, channels, notify, &steps)
	}

	return steps
}

func (r *Runner) runServer(ctx context.Context, guildID string, cfg *Config, steps *[]Step) {
	if cfg.Server.Name == "" && cfg.Server.Description == "" {
		return
	}
	params := &discordgo.GuildParams{Name: cfg.Server.Name, Description: cfg.Server.Description}
	if _, err := r.s.GuildEdit(guildID, params); err == nil {
		*steps = append(*steps, Step{Section: "server", Label: cfg.Server.Name, Status: "ok", Detail: "server updated"})
	} else {
		*steps = append(*steps, Step{Section: "server", Label: cfg.Server.Name, Status: "failed", Detail: err.Error()})
	}
}

func (r *Runner) runRoles(ctx context.Context, guildID string, cfg *Config, steps *[]Step) map[string]string {
	out := make(map[string]string, len(cfg.Roles))
	for _, rc := range cfg.Roles {
		color, err := parseRoleColor(rc.Color)
		if err != nil {
			*steps = append(*steps, Step{Section: "roles", Label: rc.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		perms := int64(0)
		for _, p := range rc.Permissions {
			perms |= rolePermBits[p]
		}
		hoist := rc.Hoist
		mentionable := rc.Mentionable
		role, err := r.s.GuildRoleCreate(guildID, &discordgo.RoleParams{
			Name:        rc.Name,
			Color:       &color,
			Hoist:       &hoist,
			Mentionable: &mentionable,
			Permissions: &perms,
		})
		if err != nil {
			*steps = append(*steps, Step{Section: "roles", Label: rc.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		out[rc.Name] = role.ID
		*steps = append(*steps, Step{Section: "roles", Label: rc.Name, Status: "ok", Detail: role.ID})
	}
	return out
}

func (r *Runner) runChannels(ctx context.Context, guildID string, cfg *Config, roles map[string]string, steps *[]Step) map[string]string {
	out := make(map[string]string, len(cfg.Channels))
	parentIDs := map[string]string{} // channel name → parent channel ID

	for _, cc := range cfg.Channels {
		chType, err := parseChannelType(cc.Type)
		if err != nil {
			*steps = append(*steps, Step{Section: "channels", Label: cc.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		data := discordgo.GuildChannelCreateData{
			Name:     cc.Name,
			Type:     chType,
			Topic:    cc.Topic,
			NSFW:     cc.NSFW,
			ParentID: parentIDs[cc.Parent],
		}
		skip := false
		for _, o := range cc.PermissionOverwrites {
			po, err := r.overwrite(ctx, guildID, roles, o)
			if err != nil {
				*steps = append(*steps, Step{Section: "channels", Label: cc.Name, Status: "failed", Detail: err.Error()})
				skip = true
				break
			}
			data.PermissionOverwrites = append(data.PermissionOverwrites, po)
		}
		if skip {
			continue
		}
		ch, err := r.s.GuildChannelCreateComplex(guildID, data)
		if err != nil {
			*steps = append(*steps, Step{Section: "channels", Label: cc.Name, Status: "failed", Detail: err.Error()})
			continue
		}
		out[cc.Name] = ch.ID
		parentIDs[cc.Name] = ch.ID
		*steps = append(*steps, Step{Section: "channels", Label: cc.Name, Status: "ok", Detail: ch.ID})
	}
	return out
}

// overwrite converts a permission_overwrites entry into a Discord overwrite by
// resolving "role:Name" targets against the freshly-created roles map (falling
// back to a server-wide role name lookup) and "user:ID" targets directly.
func (r *Runner) overwrite(ctx context.Context, guildID string, roles map[string]string, oc PermissionOverwriteConfig) (*discordgo.PermissionOverwrite, error) {
	target := strings.TrimSpace(oc.Target)
	var po discordgo.PermissionOverwrite
	switch {
	case strings.HasPrefix(target, "role:"):
		name := strings.TrimSpace(strings.TrimPrefix(target, "role:"))
		po = discordgo.PermissionOverwrite{Type: discordgo.PermissionOverwriteTypeRole}
		if id, ok := roles[name]; ok {
			po.ID = id
		} else {
			role, err := findRoleID(r.s, guildID, name)
			if err != nil {
				return nil, fmt.Errorf("overwrite: %w", err)
			}
			po.ID = role.ID
		}
	case strings.HasPrefix(target, "user:"):
		po = discordgo.PermissionOverwrite{ID: strings.TrimSpace(strings.TrimPrefix(target, "user:")), Type: discordgo.PermissionOverwriteTypeMember}
	default:
		return nil, fmt.Errorf("overwrite target %q must be \"role:Name\" or \"user:ID\"", target)
	}
	for _, p := range oc.Allow {
		po.Allow |= channelPermBits[p]
	}
	for _, p := range oc.Deny {
		po.Deny |= channelPermBits[p]
	}
	return &po, nil
}
