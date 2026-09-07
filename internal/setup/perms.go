package setup

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// rolePermBits maps friendly permission names (used in [[roles]] and
// [[channels]] permission_overwrites) to Discord permission bits. Only
// server-wide role permissions are listed.
var rolePermBits = map[string]int64{
	"administrator":            discordgo.PermissionAdministrator,
	"view_channel":             discordgo.PermissionViewChannel,
	"manage_channels":          discordgo.PermissionManageChannels,
	"manage_roles":             discordgo.PermissionManageRoles,
	"manage_guild":             discordgo.PermissionManageGuild,
	"create_instant_invite":    discordgo.PermissionCreateInstantInvite,
	"manage_messages":          discordgo.PermissionManageMessages,
	"manage_webhooks":          discordgo.PermissionManageWebhooks,
	"manage_threads":           discordgo.PermissionManageThreads,
	"create_public_threads":    discordgo.PermissionCreatePublicThreads,
	"create_private_threads":   discordgo.PermissionCreatePrivateThreads,
	"send_messages_in_threads": discordgo.PermissionSendMessagesInThreads,
	"use_external_stickers":    discordgo.PermissionUseExternalStickers,
	"use_application_commands": discordgo.PermissionUseApplicationCommands,
	"send_messages":            discordgo.PermissionSendMessages,
	"send_tts_messages":        discordgo.PermissionSendTTSMessages,
	"send_voice_messages":      discordgo.PermissionSendVoiceMessages,
	"read_message_history":     discordgo.PermissionReadMessageHistory,
	"mention_everyone":         discordgo.PermissionMentionEveryone,
	"manage_expressions":       discordgo.PermissionManageGuildExpressions,
	"create_expressions":       discordgo.PermissionCreateGuildExpressions,
	"use_external_emojis":      discordgo.PermissionUseExternalEmojis,
	"embed_links":              discordgo.PermissionEmbedLinks,
	"attach_files":             discordgo.PermissionAttachFiles,
	"add_reactions":            discordgo.PermissionAddReactions,
	"timeout_members":          discordgo.PermissionModerateMembers,
	"kick_members":             discordgo.PermissionKickMembers,
	"ban_members":              discordgo.PermissionBanMembers,
	"view_audit_log":           discordgo.PermissionViewAuditLogs,
	"manage_events":            discordgo.PermissionManageEvents,
	"connect":                  discordgo.PermissionVoiceConnect,
	"speak":                    discordgo.PermissionVoiceSpeak,
	"mute_members":             discordgo.PermissionVoiceMuteMembers,
	"deafen_members":           discordgo.PermissionVoiceDeafenMembers,
	"move_members":             discordgo.PermissionVoiceMoveMembers,
	"priority_speaker":         discordgo.PermissionVoicePrioritySpeaker,
}

// channelPermBits mirrors the bot's -channel setperm permission names so a
// setup file and the live command accept the same vocabulary.
var channelPermBits = map[string]int64{
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

var colorRe = regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`)

// parseRoleColor turns "#RRGGBB" (or a bare 6-digit hex or integer) into the
// 3-byte Role.Color value Discord expects.
func parseRoleColor(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	if m := colorRe.FindStringSubmatch(raw); m != nil {
		v, _ := strconv.ParseUint(m[1], 16, 32)
		return int(v), nil
	}
	if n, err := strconv.Atoi(raw); err == nil && n >= 0 && n <= 16777215 {
		return n, nil
	}
	return 0, fmt.Errorf("invalid color %q (use #RRGGBB)", raw)
}

// parseChannelType maps a friendly type name to a Discord channel type.
func parseChannelType(raw string) (discordgo.ChannelType, error) {
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
		return 0, fmt.Errorf("unknown channel type %q (try text, voice, category, announcement, or forum)", raw)
	}
}

// parseVerificationLevel maps a friendly name to a Discord verification level.
// An empty value defaults to Low — Community mode requires at least Low, so
// "none" is only honored when explicitly requested.
func parseVerificationLevel(raw string) *discordgo.VerificationLevel {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		v := discordgo.VerificationLevelLow
		return &v
	case "none":
		v := discordgo.VerificationLevelNone
		return &v
	case "low":
		v := discordgo.VerificationLevelLow
		return &v
	case "medium":
		v := discordgo.VerificationLevelMedium
		return &v
	case "high":
		v := discordgo.VerificationLevelHigh
		return &v
	case "highest":
		v := discordgo.VerificationLevelVeryHigh
		return &v
	}
	return nil
}

// parseContentFilter maps a friendly name to an explicit-content filter level.
func parseContentFilter(raw string) *discordgo.ExplicitContentFilterLevel {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "disabled":
		v := discordgo.ExplicitContentFilterDisabled
		return &v
	case "members_without_roles":
		v := discordgo.ExplicitContentFilterMembersWithoutRoles
		return &v
	case "all_members":
		v := discordgo.ExplicitContentFilterAllMembers
		return &v
	}
	return nil
}

// sanitizeName lowercases a role/emoji/sticker name and strips characters the
// Discord API rejects.
func sanitizeName(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-', r == ' ':
			b.WriteRune(r)
		}
		if b.Len() >= 100 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}
