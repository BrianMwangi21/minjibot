export type SetupTemplate = {
  id: string
  name: string
  description: string
  features: string[]
  toml: string
}

export const setupTemplates: SetupTemplate[] = [
  {
    id: "community",
    name: "Community server",
    description:
      "Roles for members and moderators, an announcements/rules/chat split, Community mode, onboarding prompts, and a channel permission command.",
    features: ["Roles", "Permission locks", "Voice", "Announcements", "Community mode", "Onboarding", "Commands"],
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
type = "announcement"

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
      "Voice lobbies under a category, text channels with sensible limits, a hoisted Competitive role, and a permission command.",
    features: ["Roles", "Permission locks", "Voice", "Commands"],
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
permission_overwrites = [ { target = "role:Member", allow = ["view", "send", "reactions"], deny = [] } ]

[[channels]]
name = "Comms"
type = "category"

[[channels]]
name = "Announcements"
type = "text"
parent = "Comms"

[[commands]]
cmd = "channel"
args = "setperm {channel:memes} Member allow send"`,
  },
  {
    id: "creator",
    name: "Creator & brand",
    description:
      "An announcements channel, a subscriber tier, a fan-art channel, a forum feed, Community mode, and voice hangouts for watch parties.",
    features: ["Roles", "Permission locks", "Voice", "Forum", "Announcements", "Community mode", "Onboarding"],
    toml: `# Creator & brand server template

[server]
name = "Creator Studio"

[[roles]]
name = "Viewer"
color = "#99aab5"
permissions = ["view_channel", "read_message_history", "add_reactions"]

[[roles]]
name = "Subscriber"
color = "#f47fff"
hoist = true
permissions = ["view_channel", "send_messages", "read_message_history", "add_reactions", "embed_links", "attach_files"]

[[roles]]
name = "Moderator"
color = "#ed4245"
permissions = ["view_channel", "manage_messages", "kick_members", "ban_members", "timeout_members", "manage_roles", "view_audit_log"]

[[channels]]
name = "announcements"
type = "announcement"

[[channels]]
name = "rules"
type = "text"
permission_overwrites = [ { target = "role:Viewer", allow = ["view", "history"], deny = ["send"] } ]

[[channels]]
name = "chat"
type = "text"

[[channels]]
name = "fan-art"
type = "text"
permission_overwrites = [ { target = "role:Viewer", allow = ["view", "send", "attach", "reactions"], deny = [] } ]

[[channels]]
name = "content-feed"
type = "forum"
permission_overwrites = [ { target = "role:Viewer", allow = ["view", "history", "send"], deny = [] } ]

[[channels]]
name = "Hangout"
type = "category"

[[channels]]
name = "Live Watch"
type = "voice"
parent = "Hangout"

[[channels]]
name = "Meet & Greet"
type = "voice"
parent = "Hangout"

[community]
rules_channel = "rules"
updates_channel = "announcements"
verification_level = "medium"
content_filter = "members_without_roles"

[onboarding]
enabled = true
default_channels = ["chat"]

[[onboarding.prompts]]
title = "How do you watch?"
single_select = true

[[onboarding.prompts.options]]
title = "Regular viewer"
roles = ["Viewer"]

[[onboarding.prompts.options]]
title = "Subscriber"
roles = ["Subscriber"]`,
  },
  {
    id: "study",
    name: "Study group",
    description:
      "Student/teacher/TA roles, a private staff room hidden from students, a Q&A forum, and voice classrooms.",
    features: ["Roles", "Permission locks", "Voice", "Forum", "Onboarding"],
    toml: `# Study group template

[server]
name = "Study Hub"

[[roles]]
name = "Student"
color = "#9b59b6"
permissions = ["view_channel", "send_messages", "read_message_history", "add_reactions", "attach_files"]

[[roles]]
name = "Teacher"
color = "#e67e22"
hoist = true
permissions = ["view_channel", "send_messages", "manage_messages", "manage_channels", "manage_roles", "mute_members", "deafen_members"]

[[roles]]
name = "TA"
color = "#1abc9c"
hoist = true
permissions = ["view_channel", "send_messages", "manage_messages", "manage_channels", "mute_members", "deafen_members"]

[[channels]]
name = "main-hall"
type = "text"
topic = "Introduce yourself and share links."

[[channels]]
name = "homework"
type = "text"
permission_overwrites = [
  { target = "role:Student", allow = ["view", "send", "attach"], deny = [] },
  { target = "role:TA", allow = ["view", "manage"], deny = [] },
]

[[channels]]
name = "questions"
type = "forum"

# Hidden from students: view is denied for Student, allowed for the team.
[[channels]]
name = "staff-room"
type = "text"
permission_overwrites = [
  { target = "role:Student", deny = ["view"] },
  { target = "role:Teacher", allow = ["view", "send"] },
  { target = "role:TA", allow = ["view", "send"] },
]

[[channels]]
name = "Classes"
type = "category"

[[channels]]
name = "Lecture Hall"
type = "voice"
parent = "Classes"

[[channels]]
name = "Study Room"
type = "voice"
parent = "Classes"

[onboarding]
enabled = true
default_channels = ["main-hall"]

[[onboarding.prompts]]
title = "What best describes you?"
required = true
single_select = true

[[onboarding.prompts.options]]
title = "Student"
roles = ["Student"]

[[onboarding.prompts.options]]
title = "Teacher"
roles = ["Teacher"]

[[onboarding.prompts.options]]
title = "TA"
roles = ["TA"]`,
  },
  {
    id: "tech",
    name: "Dev community",
    description:
      "A help forum, code-review channel, announcements, a private moderator-only staff channel, community mode, and a stack-picker onboarding prompt.",
    features: ["Roles", "Permission locks", "Voice", "Forum", "Announcements", "Community mode", "Onboarding", "Commands"],
    toml: `# Dev / tech community template

[server]
name = "Dev Lounge"

[[roles]]
name = "Member"
color = "#5865f2"
permissions = ["view_channel", "send_messages", "read_message_history", "add_reactions", "embed_links", "attach_files"]

[[roles]]
name = "Developer"
color = "#00b0f4"
hoist = true
permissions = ["view_channel", "send_messages", "read_message_history", "attach_files", "connect", "speak"]

[[roles]]
name = "Moderator"
color = "#ed4245"
permissions = ["view_channel", "manage_messages", "manage_roles", "kick_members", "ban_members", "timeout_members", "view_audit_log", "manage_guild"]

[[channels]]
name = "announcements"
type = "announcement"

[[channels]]
name = "rules"
type = "text"
permission_overwrites = [ { target = "role:Member", allow = ["view", "history"], deny = ["send"] } ]

[[channels]]
name = "general"
type = "text"

[[channels]]
name = "help"
type = "forum"
permission_overwrites = [ { target = "role:Member", allow = ["view", "history", "send"], deny = [] } ]

[[channels]]
name = "code-review"
type = "text"
permission_overwrites = [ { target = "role:Member", allow = ["view", "send", "attach", "embed"], deny = [] } ]

[[channels]]
name = "Voice"
type = "category"

[[channels]]
name = "Dev Chat"
type = "voice"
parent = "Voice"

[[channels]]
name = "Pairing"
type = "voice"
parent = "Voice"

[[channels]]
name = "staff"
type = "text"
permission_overwrites = [
  { target = "role:Member", deny = ["view"] },
  { target = "role:Moderator", allow = ["view", "send", "manage"] },
]

[community]
rules_channel = "rules"
updates_channel = "announcements"
verification_level = "low"

[onboarding]
enabled = true
default_channels = ["general"]

[[onboarding.prompts]]
title = "Pick your stack"

[[onboarding.prompts.options]]
title = "Frontend"
roles = ["Member"]

[[onboarding.prompts.options]]
title = "Backend"
roles = ["Developer"]

[[commands]]
cmd = "channel"
args = "setperm {channel:general} Member allow send"`,
  },
  {
    id: "business",
    name: "Business workspace",
    description:
      "Team voice rooms, a standup channel, a leadership-only channel, and an onboarding prompt to pick your team.",
    features: ["Roles", "Permission locks", "Voice", "Announcements", "Onboarding"],
    toml: `# Business / team workspace template

[server]
name = "Acme Team"

[[roles]]
name = "Staff"
color = "#99aab5"
permissions = ["view_channel", "send_messages", "read_message_history"]

[[roles]]
name = "Manager"
color = "#2ecc71"
hoist = true
permissions = ["view_channel", "manage_messages", "manage_channels", "manage_roles", "kick_members", "timeout_members"]

[[roles]]
name = "Leadership"
color = "#e74c3c"
hoist = true
permissions = ["administrator"]

[[channels]]
name = "announcements"
type = "announcement"

[[channels]]
name = "general"
type = "text"

[[channels]]
name = "standup"
type = "text"
permission_overwrites = [
  { target = "role:Staff", allow = ["view", "send"], deny = [] },
  { target = "role:Manager", allow = ["view", "send", "manage"], deny = [] },
]

# Leadership-only: hidden from Staff and Manager alike.
[[channels]]
name = "leadership"
type = "text"
permission_overwrites = [
  { target = "role:Staff", deny = ["view"] },
  { target = "role:Manager", deny = ["view"] },
  { target = "role:Leadership", allow = ["view", "send"] },
]

[[channels]]
name = "Team Rooms"
type = "category"

[[channels]]
name = "Engineering"
type = "voice"
parent = "Team Rooms"

[[channels]]
name = "Design"
type = "voice"
parent = "Team Rooms"

[[channels]]
name = "Marketing"
type = "voice"
parent = "Team Rooms"

[onboarding]
enabled = true
default_channels = ["general"]

[[onboarding.prompts]]
title = "Select your team"
single_select = true

[[onboarding.prompts.options]]
title = "Engineering"
roles = ["Staff"]

[[onboarding.prompts.options]]
title = "Design"
roles = ["Staff"]

[[onboarding.prompts.options]]
title = "Marketing"
roles = ["Staff"]`,
  },
  {
    id: "ultimate",
    name: "Everything (feature showcase)",
    description:
      "Exercises every provisioning feature: tiers and voice, a VIP-only voice lounge, forums, announcements, emojis, stickers, Community mode, multi-step onboarding, and commands.",
    features: [
      "Roles",
      "Permission locks",
      "Voice",
      "Forum",
      "Announcements",
      "Community mode",
      "Onboarding",
      "Emojis",
      "Stickers",
      "Commands",
    ],
    toml: `# Everything template - exercises every provisioning feature

[server]
name = "Ultimate Server"
description = "Provisioned by MinjiBot from a single document."

[[roles]]
name = "Member"
color = "#5865f2"
permissions = ["view_channel", "send_messages", "read_message_history", "add_reactions", "embed_links", "attach_files", "connect", "speak"]

[[roles]]
name = "VIP"
color = "#f1c40f"
hoist = true
mentionable = true
permissions = ["view_channel", "send_messages", "read_message_history", "add_reactions", "embed_links", "attach_files", "connect", "speak", "priority_speaker"]

[[roles]]
name = "Moderator"
color = "#ed4245"
hoist = true
permissions = ["view_channel", "manage_messages", "manage_roles", "kick_members", "ban_members", "timeout_members", "view_audit_log", "manage_guild"]

[[channels]]
name = "announcements"
type = "announcement"

[[channels]]
name = "rules"
type = "text"
permission_overwrites = [
  { target = "role:Member", allow = ["view"], deny = ["send"] },
  { target = "role:VIP", allow = ["view", "send"], deny = [] },
]

[[channels]]
name = "lounge"
type = "text"
topic = "Kick back and chat."

[[channels]]
name = "memes"
type = "text"

[[channels]]
name = "feedback"
type = "forum"

[[channels]]
name = "Audio"
type = "category"

[[channels]]
name = "Main Stage"
type = "voice"
parent = "Audio"

[[channels]]
name = "vip-lounge"
type = "voice"
parent = "Audio"
permission_overwrites = [
  { target = "role:Member", deny = ["view", "connect"] },
  { target = "role:VIP", allow = ["view", "connect"] },
]

# Emojis/stickers upload from a public image URL - swap the URLs below for
# real image files before running (placeholders will fail that step).
[[emojis]]
name = "wave"
url = "https://example.com/wave.png"

[[stickers]]
name = "wave"
url = "https://example.com/wave.png"
description = "A friendly wave"
tags = "wave"

[community]
rules_channel = "rules"
updates_channel = "announcements"
verification_level = "medium"
content_filter = "members_without_roles"

[onboarding]
enabled = true
default_channels = ["lounge"]

[[onboarding.prompts]]
title = "Where did you find this server?"

[[onboarding.prompts.options]]
title = "Somewhere random"
channels = ["memes"]

[[onboarding.prompts.options]]
title = "A friend invited me"
channels = ["lounge"]

[[onboarding.prompts]]
title = "Level up to VIP"
in_onboarding = true

[[onboarding.prompts.options]]
title = "I'd like VIP"
roles = ["VIP"]

[[commands]]
cmd = "channel"
args = "setperm {channel:memes} Member allow send"

[[commands]]
cmd = "channel"
args = "setperm {channel:vip-lounge} VIP allow connect"`,
  },
  {
    id: "minimal",
    name: "Minimal",
    description:
      "The smallest useful document: a server name, one role, and two channels.",
    features: ["Roles", "Voice"],
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