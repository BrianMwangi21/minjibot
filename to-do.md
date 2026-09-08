# To Do

## High priority

- [ ] Social lookup commands (github, instagram, tiktok, youtube, etc.)
- [ ] Character RP commands (character create/say/act/autosay)
- [ ] HTTP interactions endpoint (`POST /interactions`) for slash commands over HTTP
- [ ] CI pipeline (GitHub Actions: build, vet, test, lint)
- [ ] Dashboard: command management page (enable/disable per guild)

## Medium priority

- [ ] Social lookup expansions (linkedin, pinterest, roblox, etc.) beyond the planned list
- [ ] Dashboard: birthday management UI
- [ ] Dashboard: command management page
- [ ] Rate limiting per user/guild

## Low priority

- [ ] Dashboard: user profile page
- [ ] Dashboard: birthday management UI
- [ ] Dashboard: diary entries UI
- [ ] Embed all help responses with rich formatting
- [ ] Custom prefix per guild
- [ ] Auto-moderation (spam, link filtering)
- [ ] Welcome/goodbye messages
- [ ] Audit log rotation (auto-delete old entries)

## Done

- [x] Create landing page
- [x] Bot entry point (gateway + prefix + slash)
- [x] ~100+ commands (general, moderation, utility, fun, RP)
- [x] Help pagination (reactions + buttons)
- [x] TLDR command
- [x] Discord OAuth2 flow
- [x] Session-based API auth
- [x] Dashboard guild picker + log views
- [x] Deleted message logging (DB + log channel)
- [x] Moderation action logging (DB + log channel)
- [x] Audit log with actor/target names
- [x] Setup command (log channel configuration)
- [x] Birthday system (add, list, channel, role, celebrate)
- [x] Diary system (add, view, delete)
- [x] Polls (reaction-based + quick yes/no)
- [x] Roleplay emotions + actions
- [x] Moderation: ban, kick, timeout, warn, purge, nuke
- [x] Moderation: jail, unjail, staffstrip
- [x] Moderation: role management (add, create, edit, hoist, member)
- [x] Moderation: channel management (hide, reveal, lockdown, nsfw, sfw, slowmode, topic)
- [x] Moderation: denyperm, imute, gifmute
- [x] Moderation: force nickname, nick lock/unlock
- [x] Image commands (caption, compress, img2gif, vid2gif)
- [x] GIF search
- [x] Emoji management (add, enlarge, list, remove, steal)
- [x] Sticker management (add, steal, remove)
- [x] Quote, pin, unpin
- [x] Translate, reminder, reverse image search
- [x] DDG search, chat history search
- [x] Botinfo, channelinfo, guild stats
- [x] Avatar, banner, user info, roles, emojis, stickers, bans
- [x] Weather, timezone, Urban Dictionary

## Done — non-command features

- [x] Server setup provisioning (TOML: server, roles, channels, emojis, stickers, community, onboarding, commands)
- [x] Setup dry-run validation + guard against established servers
- [x] Rules embed section (`[rules]`: post + pin a rules embed)
- [x] Logging section (`[logging]`: mod-log channel + deleted-message capture)
- [x] Onboarding default-channel auto-fill to Discord's minimums
- [x] Deleted-message content logging with 30-day retention + background pruner
- [x] Moderation action + audit logging (DB + log channel)
- [x] Guild settings API (prefix, language, log channel, auto-moderation)
- [x] Dashboard: settings page, setup page (dry-run + run), profile, diary
- [x] Public setup guide with downloadable TOML templates (8 templates)
- [x] Donation prompt cadence (every 13-15 non-moderation commands)
- [x] Avatar history tracking (enable/disable, store past avatars)
