# MinjiBot — AI Agent Context

Go Discord bot + REST API + web dashboard.
Module: `github.com/kibetnathan/minjibot`, Go 1.26, pgx/v5 + Echo v5 + discordgo.

## Build & test

```sh
go build ./...        # compile all packages
go vet ./...          # static analysis
gofmt -w .            # format (no CI linter yet)
make test             # go test ./...
make integration-test # integration_tests/db_test.go against TESTING_DB
```

## Database

- **Postgres 15** on host port **5434** (not 5432): `docker compose up -d database`
- All config from `.env` (gitignored): `DB_URL`, `TESTING_DB`, `POSTGRES_*`
- Migrations: goose format in `db/migrations/` (UTC-timestamp filenames, `-- +goose Up/Down`)
- `make goose-migrate-up` / `make goose-migrate-down` (uses exported `.env` vars)

## Codegen

`sqlc generate` — reads `db/queries/*.sql` + `db/migrations/` schema, writes `infrastructure/postgres/` (package `postgres`, pgx/v5, `Querier` interface, JSON tags).
**Never hand-edit** `infrastructure/postgres/`.

## Architecture

```
cmd/
  main.go            Unified entrypoint (bot + API)
  bot/main.go        Bot-only
  api/main.go        API-only

internal/
  bot/               Discord gateway connection, handler registration
    handlers/        Gateway event handlers (message, message-delete, interaction)
  commands/          ~100+ bot commands (prefix + slash), help pagination, tldr
    commands.go      CommandHandler struct, Handle/HandleSlash dispatch, thin wrappers
    moderation.go    Shared mod helpers (perm checks, audit logging, embed builders)
    mod_*.go         Per-feature command implementations (ban, jail, purge, etc.)
    slash_commands.go Slash command definitions (SlashCommands variable)
    helpers.go       Interaction option helpers (OptInt, OptBool, OptUser)
    help.go          Help sections, BuildHelpPageEmbed
    pagination.go    Reaction-based (prefix) + button-based (slash) pagination
    rp.go            Roleplay emotion/action message helpers
  api/               Echo HTTP server
    app.go           App struct, NewApp, registerRoutes, CORS
    auth.go          Discord OAuth flow (/api/auth/discord, callback, logout, me)
    logs.go          Dashboard log endpoints (/api/guilds, /api/logs/*)
    settings.go      Guild settings endpoints (GET/PUT /api/guilds/:id/settings)
    setup.go         Setup provisioning endpoints (run, dry-run, channels)
    diary.go         Diary endpoints
    session.go       resolveSession helper
  setup/             TOML server provisioning (Parse/Validate, Runner, per-section impls)
    toml.go          Config structs + structural validation
    runner.go        Runner struct, Run() sequencing, NewRunner
    commands.go      Prefix-command passthrough with {role}/{channel} ref expansion
    media.go         Emoji/sticker uploads
    onboarding.go    Onboarding publish + default-channel auto-fill
    perms.go         Permission/color parsing tables
    guard.go         Refuses to provision servers that already look live
    rules_logging.go [rules] embed (post+pin) and [logging] settings persistence
  safe/              Panic-recovery helpers for gateway handlers
  config/            env config (caarlos0/env/v11)
  domain/            Domain entities (pure structs, no deps)
  ports/
    dto/             Data transfer objects for repository calls
    repository/      Repository interfaces + SQL implementations
  services/auth/     Discord OAuth2 client + session manager
  logger/            slog JSON logger setup

infrastructure/postgres/  Generated sqlc output (do not edit)
db/migrations/            Goose SQL schema
db/queries/               Named sqlc queries
tests/                    External unit tests for commands package
integration_tests/        DB integration tests
dashboard/minji-bot/      React + Vite + shadcn dashboard
```

## Key files to know

| File | What it does |
|---|---|
| `internal/commands/commands.go` | `CommandHandler` struct, `Handle()` + `HandleSlash()` dispatch, thin wrappers that delegate to `mod_*.go` functions |
| `internal/commands/moderation.go` | Shared mod utilities: `effectiveModPerms`, `requireModerator`, `resolveTargetUser`, `auditAction`, `logModAction`, embed builders, option helpers |
| `internal/commands/slash_commands.go` | `SlashCommands` variable — all ApplicationCommand definitions |
| `internal/commands/helpers.go` | `OptInt`, `OptBool`, `OptUser` — interaction option extractors |
| `internal/bot/app.go` | Bot `App` struct, `NewApp`, `RegisterHandlers`, gateway intents |
| `internal/api/app.go` | API `App` struct, Echo setup, CORS, route wiring |
| `internal/api/logs.go` | Dashboard endpoints: guild list, deleted messages, mod actions |
| `internal/ports/repository/store.go` | `SQLStore` wrapping generated `Querier` |

## Non-command features

These ship outside the command catalog and are easy to miss when grepping for commands.

- **Server setup provisioning** — `internal/setup/` parses + validates a TOML document and `Runner.Run` applies it (roles → channels → rules → logging → media → community → onboarding → commands). Each item appends a `Step` (ok|skipped|failed). `NewRunner(s, commands, settings)`: `settings` persists `[logging]` guild settings; may be nil (logging step then fails). Guard refuses established servers. The dashboard sends the document via `POST /api/guilds/:guildId/setup` (dry-run = `.../dry-run`).
- **Rules embed** — `[rules]` posts and pins a rules embed (`ChannelMessageSendEmbed` + `ChannelMessagePin`); a pin failure is non-fatal. Only runs if the referenced channel was successfully created.
- **Logging settings persistence** — `[logging]` upserts guild settings (log channel + message-content capture) via `GuildSettingsRepository`; preserves existing prefix/language/auto-moderation fields by reading them first.
- **Deleted-message + mod-action logging** — rolling 2000-message cache (`State.MaxMessageCount`) so deleted content is reconstructable; opt-in content capture pruned after 30 days by `startMessageLogPruner` in `internal/bot/app.go` (runs at startup + every 6h).
- **Donation prompt cadence** — `commands.IsModerationCommand(name)` exempts mod actions; `MaybeShowDonatePrompt` (called from message/interaction handlers, not setup passthrough) surfaces the donation card every randomly-chosen 13-15 non-moderation commands. Guarded by a mutex; gateway handlers run concurrently.
- **Guild settings** — prefix, language, auto-moderation, log channel, message logging via `internal/api/settings.go` + dashboard settings page.

## Conventions

- Prefix commands use `-` (e.g. `-ban`, `-help`)
- Slash commands registered globally on `Ready` via `s.ApplicationCommandCreate`
- Each command has both a prefix handler (`foo(s, m, args)`) and slash handler (`fooSlash(s, i)`)
- Slash dispatch goes through `HandleSlash()` in `commands.go`
- Mod actions log to both `audit_logs` table and configured log channel (via `logModAction`)
- Dashboard uses session cookie auth (Discord OAuth2 flow in `services/auth/`)

## Gotchas

- Bot gateway intents include `IntentsGuildMessageReactions` (needed for `-help` reaction pagination)
- `State.MaxMessageCount = 2000` — rolling cache for deleted message content
- `DISCORD_CLIENT_ID` falls back to bot user ID for slash command registration if unset
- `APP_URL` must be the API origin, `FRONTEND_URL` the dashboard origin
- Vite dev proxy: `/api` → `:8080`
- Postgres on port 5434 (not 5432)
- `init-testdb.sh` creates `TESTING_DB` on first docker start
- `go.mod` has some `// indirect` deps — don't move to direct until something imports them

## Dependencies to not touch

- `infrastructure/postgres/` — generated only
- `go.mod` indirect deps — leave as indirect until explicit import
