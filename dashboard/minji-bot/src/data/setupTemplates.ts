export type SetupTemplate = {
  id: string
  name: string
  description: string
  toml: string
}

export const setupTemplates: SetupTemplate[] = [
  {
    id: "community",
    name: "Community server",
    description:
      "Roles for members and moderators, an announcements/rules/chat split, Community mode, onboarding prompts, and a channel permission command.",
    toml: `# Community server template
# Roles, channels, Community mode, onboarding, then a command.

[server]
name = "My Community"

[[roles]]
name = "Member"
color = "#5865f2"
permissions = ["view_channel", "send_messages", "read_message_history", "add_reactions"]

[[roles]]
name = "Moderator"
color = "#ed4245"
permissions = ["view_channel", "manage_messages", "kick_members", "ban_members", "timeout_members"]

[[channels]]
name = "announcements"
type = "text"

[[channels]]
name = "rules"
type = "text"
permission_overwrites = [ { target = "role:Member", allow = ["view"], deny = ["send"] } ]

[[channels]]
name = "chat"
type = "text"

[[channels]]
name = "Voice"
type = "category"

[[channels]]
name = "General"
type = "voice"
parent = "Voice"

# Enabling Community mode points Discord's rules/announcements channels at the
# channels created above.
[community]
rules_channel = "rules"
updates_channel = "announcements"
verification_level = "low"

[onboarding]
enabled = true
default_channels = ["chat"]

[[onboarding.prompts]]
title = "Who are you?"
required = true
single_select = true

[[onboarding.prompts.options]]
title = "Regular member"
roles = ["Member"]

[[onboarding.prompts.options]]
title = "Moderator"
roles = ["Moderator"]

# Commands run last, after channels and roles exist.
[[commands]]
cmd = "channel"
args = "setperm {channel:chat} Member allow send"`,
  },
  {
    id: "gaming",
    name: "Gaming server",
    description:
      "Voice lobbies under a category, text channels with sensible limits, and a hoisted Competitive role.",
    toml: `# Gaming server template

[server]
name = "Gamer Den"

[[roles]]
name = "Member"
color = "#5865f2"
permissions = ["view_channel", "send_messages", "read_message_history", "add_reactions"]

[[roles]]
name = "Competitive"
color = "#f47fff"
hoist = true
permissions = ["view_channel", "send_messages", "read_message_history", "connect", "speak"]

[[roles]]
name = "Moderator"
color = "#ed4245"
permissions = ["view_channel", "manage_messages", "kick_members", "ban_members", "timeout_members", "manage_channels"]

[[channels]]
name = "Lobby"
type = "category"

[[channels]]
name = "General"
type = "voice"
parent = "Lobby"

[[channels]]
name = "Squad 1"
type = "voice"
parent = "Lobby"

[[channels]]
name = "squad-lfg"
type = "text"
parent = "Lobby"

[[channels]]
name = "memes"
type = "text"

[[channels]]
name = "Comms"
type = "category"

[[channels]]
name = "Announcements"
type = "text"
parent = "Comms"`,
  },
  {
    id: "minimal",
    name: "Minimal",
    description:
      "The smallest useful document: a server name, one role, and two channels.",
    toml: `# Minimal template

[server]
name = "My Server"
description = "A fresh server provisioned with MinjiBot."

[[roles]]
name = "Member"
color = "#99aab5"
permissions = ["view_channel", "send_messages", "read_message_history"]

[[channels]]
name = "general"
type = "text"
topic = "Say hi!"

[[channels]]
name = "chill"
type = "voice"`,
  },
]