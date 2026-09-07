package commands

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// channelPerms maps friendly permission names to Discord permission bits, so
// the channel command can accept -channel setperm ... <permission> without the
// caller needing to memorise bit values. Only server-wide channel permissions
// that map cleanly to a per-channel overwrite are listed.
var channelPerms = map[string]int64{
	"view":      discordgo.PermissionViewChannel,
	"send":      discordgo.PermissionSendMessages,
	"history":   discordgo.PermissionReadMessageHistory,
	"attach":    discordgo.PermissionAttachFiles,
	"embed":     discordgo.PermissionEmbedLinks,
	"mention":   discordgo.PermissionMentionEveryone,
	"emoji":     discordgo.PermissionUseExternalEmojis,
	"reactions": discordgo.PermissionAddReactions,
	"manage":    discordgo.PermissionManageMessages,
	"managech":  discordgo.PermissionManageChannels,
	"webhooks":  discordgo.PermissionManageWebhooks,
	"invite":    discordgo.PermissionCreateInstantInvite,
	"connect":   discordgo.PermissionVoiceConnect,
	"speak":     discordgo.PermissionVoiceSpeak,
	"mute":      discordgo.PermissionVoiceMuteMembers,
	"deafen":    discordgo.PermissionVoiceDeafenMembers,
	"priority":  discordgo.PermissionVoicePrioritySpeaker,
	"commands":  discordgo.PermissionUseApplicationCommands,
}

// channelUsageEmbed is shown when the channel command runs without a
// subcommand, listing every supported subcommand.
func channelUsageEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Color: 0x5865F2,
		Title: "Channel management",
		Description: strings.Join([]string{
			"**Create:** `-channel create <name> [text|voice|category] [topic]`",
			"**Edit:** `-channel edit [channel] rename|topic|slowmode|nsfw <value>`",
			"**Permissions:** `-channel setperm [channel] <role> <allow|deny> <permission>`",
			"**Info:** `-channel info [channel]`",
			"",
			"Available permissions for `setperm`: " + fmt.Sprintf("`%s`", strings.Join(sortedChannelPermKeys(), "`, `")),
			"",
			"Moderation permission required.",
		}, "\n"),
		Footer: &discordgo.MessageEmbedFooter{Text: "Tip: use /channel for the same actions with a menu."},
	}
}

func sortedChannelPermKeys() []string {
	out := make([]string, 0, len(channelPerms))
	for k := range channelPerms {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// channelMessageCommandHandler is the prefix handler for -channel. With no
// arguments it posts the usage embed; otherwise it dispatches to a subcommand.
func channelMessageCommandHandler(h *CommandHandler, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if ok, err := requireModForMessage(s, m, "Channel"); !ok {
		return err
	}
	if len(args) == 0 {
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, channelUsageEmbed())
		return err
	}
	switch strings.ToLower(args[0]) {
	case "create":
		return channelCreateMessage(h, s, m, args[1:])
	case "edit":
		return channelEditMessage(h, s, m, args[1:])
	case "setperm":
		return channelSetPermMessage(h, s, m, args[1:])
	case "info":
		return channelInfoMessage(s, m, args[1:])
	default:
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, channelUsageEmbed())
		return err
	}
}

// channelCreateMessage handles `-channel create <name> [type] [topic]`.
func channelCreateMessage(h *CommandHandler, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if len(args) == 0 {
		return sendModError(s, m.ChannelID, "Channel create", "Usage: `-channel create <name> [text|voice|category] [topic]`")
	}
	name := sanitizeChannelName(args[0])
	if name == "" {
		return sendModError(s, m.ChannelID, "Channel create", "Channel name cannot be empty.")
	}
	chType := discordgo.ChannelTypeGuildText
	topic := ""
	if len(args) > 1 {
		t, err := channelTypeFromArg(args[1])
		if err != nil {
			return sendModError(s, m.ChannelID, "Channel create", err.Error())
		}
		chType = t
		if len(args) > 2 {
			topic = strings.Join(args[2:], " ")
		}
	}
	data := discordgo.GuildChannelCreateData{Name: name, Type: chType}
	if chType == discordgo.ChannelTypeGuildText || chType == discordgo.ChannelTypeGuildNews {
		data.Topic = topic
	}
	ch, err := s.GuildChannelCreateComplex(m.GuildID, data)
	if err != nil {
		return sendModError(s, m.ChannelID, "Channel create", fmt.Sprintf("Failed to create channel: %s", err))
	}
	if chType == discordgo.ChannelTypeGuildText {
		auditAction(h, context.Background(), m.GuildID, "CHANNEL_CREATE", m.Author.ID, ch.ID, map[string]any{"name": ch.Name, "type": "text"})
		_, err = s.ChannelMessageSendEmbed(m.ChannelID, modSuccessEmbed("Channel create", fmt.Sprintf("Created <#%s>.", ch.ID)))
	} else {
		auditAction(h, context.Background(), m.GuildID, "CHANNEL_CREATE", m.Author.ID, ch.ID, map[string]any{"name": ch.Name, "type": channelTypeName(ch.Type)})
		_, err = s.ChannelMessageSendEmbed(m.ChannelID, modSuccessEmbed("Channel create", fmt.Sprintf("Created **%s** (`%s`).", ch.Name, channelTypeName(ch.Type))))
	}
	return err
}

// channelEditMessage handles `-channel edit [channel] <rule> <value>`. The
// channel is optional; when the first argument is not a channel mention/ID the
// current channel is used. Supported rules: rename, topic, slowmode, nsfw.
func channelEditMessage(h *CommandHandler, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if len(args) < 2 {
		return sendModError(s, m.ChannelID, "Channel edit", "Usage: `-channel edit [channel] rename|topic|slowmode|nsfw <value>`")
	}
	channelID := m.ChannelID
	rest := args
	if id := ParseChannelMention(args[0]); id != "" {
		channelID = id
		rest = args[1:]
	}
	if len(rest) < 2 {
		return sendModError(s, m.ChannelID, "Channel edit", "Usage: `-channel edit [channel] rename|topic|slowmode|nsfw <value>`")
	}
	rule := strings.ToLower(rest[0])
	value := strings.Join(rest[1:], " ")

	var err error
	switch rule {
	case "rename", "name":
		err = editChannelName(s, channelID, value)
	case "topic":
		err = editChannelTopic(s, channelID, value)
	case "slowmode", "slow":
		err = editChannelSlowmode(s, channelID, value)
	case "nsfw":
		err = editChannelNSFW(s, channelID, value)
	default:
		return sendModError(s, m.ChannelID, "Channel edit", "Unknown rule. Use `rename`, `topic`, `slowmode`, or `nsfw`.")
	}
	if err != nil {
		return sendModError(s, m.ChannelID, "Channel edit", fmt.Sprintf("Failed to update channel: %s", err))
	}
	auditAction(h, context.Background(), m.GuildID, "CHANNEL_EDIT", m.Author.ID, channelID, map[string]any{"rule": rule, "value": value})
	_, err = s.ChannelMessageSendEmbed(m.ChannelID, modSuccessEmbed("Channel edit", fmt.Sprintf("Updated **%s** on <#%s>.", rule, channelID)))
	return err
}

func editChannelName(s *discordgo.Session, channelID, value string) error {
	name := sanitizeChannelName(value)
	if name == "" {
		return fmt.Errorf("channel name cannot be empty")
	}
	if _, err := s.ChannelEditComplex(channelID, &discordgo.ChannelEdit{Name: name}); err != nil {
		return err
	}
	return nil
}

func editChannelTopic(s *discordgo.Session, channelID, value string) error {
	topic := strings.TrimSpace(value)
	if len(topic) > 1024 {
		topic = topic[:1024]
	}
	_, err := s.ChannelEditComplex(channelID, &discordgo.ChannelEdit{Topic: topic})
	return err
}

func editChannelSlowmode(s *discordgo.Session, channelID, value string) error {
	secs, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || secs < 0 {
		return fmt.Errorf("slowmode must be a whole number of seconds >= 0")
	}
	if secs > 21600 {
		secs = 21600
	}
	_, err = s.ChannelEditComplex(channelID, &discordgo.ChannelEdit{RateLimitPerUser: &secs})
	return err
}

func editChannelNSFW(s *discordgo.Session, channelID, value string) error {
	on, err := parseBoolArg(value)
	if err != nil {
		return err
	}
	_, err = s.ChannelEditComplex(channelID, &discordgo.ChannelEdit{NSFW: &on})
	return err
}

// channelSetPermMessage handles
// `-channel setperm [channel] <role> <allow|deny> <permission>`. When the
// first argument is a channel mention/ID it selects that channel, otherwise
// the current channel is used and the first argument must be the role.
func channelSetPermMessage(h *CommandHandler, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	channelID := m.ChannelID
	rest := args
	if len(rest) > 0 {
		if id := ParseChannelMention(rest[0]); id != "" {
			channelID = id
			rest = rest[1:]
		}
	}
	if len(rest) < 3 {
		return sendModError(s, m.ChannelID, "Channel setperm", "Usage: `-channel setperm [channel] <role> <allow|deny> <permission>`")
	}
	role, err := resolveRoleArg(s, m.GuildID, rest[0])
	if err != nil {
		return sendModError(s, m.ChannelID, "Channel setperm", err.Error())
	}
	allow, err := parseAllowDeny(rest[1])
	if err != nil {
		return sendModError(s, m.ChannelID, "Channel setperm", err.Error())
	}
	bit, ok := channelPerms[strings.ToLower(rest[2])]
	if !ok {
		return sendModError(s, m.ChannelID, "Channel setperm", fmt.Sprintf("Unknown permission %q. Valid: %s", rest[2], strings.Join(sortedChannelPermKeys(), ", ")))
	}
	var allowBits, denyBits int64
	if allow {
		allowBits = bit
	} else {
		denyBits = bit
	}
	if err := s.ChannelPermissionSet(channelID, role.ID, discordgo.PermissionOverwriteTypeRole, allowBits, denyBits); err != nil {
		return sendModError(s, m.ChannelID, "Channel setperm", fmt.Sprintf("Failed to update permissions: %s", err))
	}
	verb := "allowed"
	if !allow {
		verb = "denied"
	}
	auditAction(h, context.Background(), m.GuildID, "CHANNEL_PERM", m.Author.ID, channelID, map[string]any{"role": role.Name, "permission": rest[2], "allow": allow})
	_, err = s.ChannelMessageSendEmbed(m.ChannelID, modSuccessEmbed("Channel setperm", fmt.Sprintf("%s is now %s **%s** in <#%s>.", role.Name, verb, rest[2], channelID)))
	return err
}

// channelInfoMessage handles `-channel info [channel]`.
func channelInfoMessage(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	channelID := m.ChannelID
	for _, arg := range args {
		if id := ParseChannelMention(arg); id != "" {
			channelID = id
			break
		}
	}
	embed, err := buildChannelInfoEmbed(s, channelID)
	if err != nil {
		return sendModError(s, m.ChannelID, "Channel info", fmt.Sprintf("Could not load channel: %s", err))
	}
	_, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
	return err
}

// channelSlashCommandHandler is the slash handler for /channel. It responds to
// the interaction for every subcommand.
func channelSlashCommandHandler(h *CommandHandler, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	ok, msg := requireModerator(i.Member)
	if !ok {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{modErrorEmbed("Channel", msg)}},
		})
	}
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{channelUsageEmbed()}},
		})
	}
	sub := data.Options[0]
	opts := OptionMap(sub.Options)
	channelID := optChannel(opts, "channel", i.ChannelID)

	switch sub.Name {
	case "create":
		return channelCreateSlash(h, s, i, opts)
	case "edit":
		return channelEditSlash(h, s, i, opts, channelID)
	case "setperm":
		return channelSetPermSlash(h, s, i, opts, channelID)
	case "info":
		embed, err := buildChannelInfoEmbed(s, channelID)
		if err != nil {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{modErrorEmbed("Channel info", "Could not load channel.")}},
			})
		}
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
	default:
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{channelUsageEmbed()}},
		})
	}
}

func channelCreateSlash(h *CommandHandler, s *discordgo.Session, i *discordgo.InteractionCreate, opts map[string]*discordgo.ApplicationCommandInteractionDataOption) error {
	name := OptString(opts, "name")
	if name == "" {
		return respondChannelSlashError(s, i, "Channel create", "Channel name is required.")
	}
	name = sanitizeChannelName(name)
	chType := discordgo.ChannelTypeGuildText
	if raw := OptString(opts, "type"); raw != "" {
		t, err := channelTypeFromArg(raw)
		if err != nil {
			return respondChannelSlashError(s, i, "Channel create", err.Error())
		}
		chType = t
	}
	topic := OptString(opts, "topic")
	data := discordgo.GuildChannelCreateData{Name: name, Type: chType}
	if (chType == discordgo.ChannelTypeGuildText || chType == discordgo.ChannelTypeGuildNews) && topic != "" {
		data.Topic = topic
	}
	ch, err := s.GuildChannelCreateComplex(i.GuildID, data)
	if err != nil {
		return respondChannelSlashError(s, i, "Channel create", fmt.Sprintf("Failed to create channel: %s", err))
	}
	auditAction(h, context.Background(), i.GuildID, "CHANNEL_CREATE", i.Member.User.ID, ch.ID, map[string]any{"name": ch.Name, "type": channelTypeName(ch.Type)})
	msg := fmt.Sprintf("Created <#%s>.", ch.ID)
	if chType != discordgo.ChannelTypeGuildText {
		msg = fmt.Sprintf("Created **%s** (`%s`).", ch.Name, channelTypeName(ch.Type))
	}
	return respondChannelSlashSuccess(s, i, "Channel create", msg)
}

func channelEditSlash(h *CommandHandler, s *discordgo.Session, i *discordgo.InteractionCreate, opts map[string]*discordgo.ApplicationCommandInteractionDataOption, channelID string) error {
	rule := strings.ToLower(OptString(opts, "property"))
	value := OptString(opts, "value")
	if rule == "" {
		return respondChannelSlashError(s, i, "Channel edit", "A property is required (rename, topic, slowmode, nsfw).")
	}

	var err error
	switch rule {
	case "rename", "name":
		err = editChannelName(s, channelID, value)
	case "topic":
		err = editChannelTopic(s, channelID, value)
	case "slowmode", "slow":
		err = editChannelSlowmode(s, channelID, value)
	case "nsfw":
		err = editChannelNSFW(s, channelID, value)
	default:
		return respondChannelSlashError(s, i, "Channel edit", "Unknown property. Use `rename`, `topic`, `slowmode`, or `nsfw`.")
	}
	if err != nil {
		return respondChannelSlashError(s, i, "Channel edit", fmt.Sprintf("Failed to update channel: %s", err))
	}
	auditAction(h, context.Background(), i.GuildID, "CHANNEL_EDIT", i.Member.User.ID, channelID, map[string]any{"rule": rule, "value": value})
	return respondChannelSlashSuccess(s, i, "Channel edit", fmt.Sprintf("Updated **%s** on <#%s>.", rule, channelID))
}

func channelSetPermSlash(h *CommandHandler, s *discordgo.Session, i *discordgo.InteractionCreate, opts map[string]*discordgo.ApplicationCommandInteractionDataOption, channelID string) error {
	roleID := OptRole(opts, "role")
	if roleID == "" {
		return respondChannelSlashError(s, i, "Channel setperm", "A role is required.")
	}
	role, err := resolveRoleArg(s, i.GuildID, roleID)
	if err != nil {
		return respondChannelSlashError(s, i, "Channel setperm", err.Error())
	}
	allow := OptBool(opts, "allow")
	permRaw := strings.ToLower(OptString(opts, "permission"))
	bit, ok := channelPerms[permRaw]
	if !ok {
		return respondChannelSlashError(s, i, "Channel setperm", fmt.Sprintf("Unknown permission %q. Valid: %s", permRaw, strings.Join(sortedChannelPermKeys(), ", ")))
	}
	var allowBits, denyBits int64
	if allow {
		allowBits = bit
	} else {
		denyBits = bit
	}
	if err := s.ChannelPermissionSet(channelID, role.ID, discordgo.PermissionOverwriteTypeRole, allowBits, denyBits); err != nil {
		return respondChannelSlashError(s, i, "Channel setperm", fmt.Sprintf("Failed to update permissions: %s", err))
	}
	verb := "allowed"
	if !allow {
		verb = "denied"
	}
	auditAction(h, context.Background(), i.GuildID, "CHANNEL_PERM", i.Member.User.ID, channelID, map[string]any{"role": role.Name, "permission": permRaw, "allow": allow})
	return respondChannelSlashSuccess(s, i, "Channel setperm", fmt.Sprintf("%s is now %s **%s** in <#%s>.", role.Name, verb, permRaw, channelID))
}

func respondChannelSlashError(s *discordgo.Session, i *discordgo.InteractionCreate, title, msg string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{modErrorEmbed(title, msg)}},
	})
}

func respondChannelSlashSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, title, msg string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{modSuccessEmbed(title, msg)}},
	})
}

// optChannel returns a channel-valued option's ID, or def when absent.
func optChannel(opts map[string]*discordgo.ApplicationCommandInteractionDataOption, name, def string) string {
	if o, ok := opts[name]; ok && o != nil {
		if v, ok := o.Value.(string); ok && v != "" {
			return v
		}
	}
	return def
}

// OptRole returns a role-valued option's ID.
func OptRole(opts map[string]*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	if o, ok := opts[name]; ok && o != nil {
		if v, ok := o.Value.(string); ok {
			return parseMentionID(v)
		}
	}
	return ""
}

// resolveRoleArg resolves a role from a mention (<@&id>), a plain ID, or a
// case-insensitive role name within the guild.
func resolveRoleArg(s *discordgo.Session, guildID, raw string) (*discordgo.Role, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return nil, fmt.Errorf("could not list roles: %s", err)
	}
	id := parseMentionID(strings.TrimSpace(raw))
	if id != "" {
		for _, r := range roles {
			if r.ID == id {
				return r, nil
			}
		}
	}
	want := strings.TrimSpace(raw)
	for _, r := range roles {
		if strings.EqualFold(r.Name, want) {
			return r, nil
		}
	}
	if id != "" {
		return nil, fmt.Errorf("role not found in this server")
	}
	return nil, fmt.Errorf("role %q not found in this server", raw)
}

// channelTypeFromArg maps a friendly type name to a Discord channel type.
func channelTypeFromArg(raw string) (discordgo.ChannelType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "text", "txt":
		return discordgo.ChannelTypeGuildText, nil
	case "voice", "vc":
		return discordgo.ChannelTypeGuildVoice, nil
	case "category", "cat":
		return discordgo.ChannelTypeGuildCategory, nil
	case "announcement", "news":
		return discordgo.ChannelTypeGuildNews, nil
	case "forum":
		return discordgo.ChannelTypeGuildForum, nil
	default:
		return 0, fmt.Errorf("unknown channel type %q (try text, voice, or category)", raw)
	}
}

// sanitizeChannelName trims a channel name and clamps it to Discord's limit,
// stripping invalid characters where possible.
func sanitizeChannelName(raw string) string {
	name := strings.TrimSpace(raw)
	for _, h := range []string{" ", "\t"} {
		name = strings.ReplaceAll(name, h, "-")
	}
	lower := strings.ToLower(name)
	var b strings.Builder
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-':
			b.WriteRune(r)
		}
	}
	name = strings.Trim(b.String(), "-")
	if len(name) > 100 {
		name = name[:100]
	}
	return name
}

// parseBoolArg parses on/off, true/false, yes/no, 1/0.
func parseBoolArg(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "true", "yes", "1", "enable", "enabled":
		return true, nil
	case "off", "false", "no", "0", "disable", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("expected on/off, got %q", raw)
	}
}

// parseAllowDeny parses the allow/deny verb used by the prefix command.
func parseAllowDeny(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "allow", "grant", "on", "true":
		return true, nil
	case "deny", "denied", "off", "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected allow or deny, got %q", raw)
	}
}
