package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kibetnathan/minjibot/internal/commands"
	"github.com/kibetnathan/minjibot/internal/ports/dto"
	"github.com/kibetnathan/minjibot/internal/ports/repository"
	"github.com/kibetnathan/minjibot/internal/safe"
	"log/slog"
)

const DefaultPrefix = "-"

type MessageHandlerDeps struct {
	Logger       *slog.Logger
	GuildRepo    repository.GuildRepository
	SettingsRepo repository.GuildSettingsRepository
	PermRepo     repository.UserPermissionRepository
	AuditRepo    repository.AuditLogRepository
}

func RegisterMessageHandler(s *discordgo.Session, deps MessageHandlerDeps, cmdHandler *commands.CommandHandler) {
	tracker := newSleepTracker()
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		onMessageCreate(s, m, deps, cmdHandler, tracker)
	})
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate, deps MessageHandlerDeps, cmdHandler *commands.CommandHandler, tracker *sleepTracker) {
	// A panic in any command handler must not take down the whole bot.
	defer safe.Recover(deps.Logger, "onMessageCreate")

	if m.Author.Bot {
		return
	}

	ctx := context.Background()

	// Resolve the guild's command prefix (fall back to the default). Looked up
	// early so both the auto-delete guard and command dispatch see the same
	// prefix.
	settings, sErr := deps.SettingsRepo.Get(ctx, m.GuildID)
	prefix := DefaultPrefix
	if sErr == nil && settings.Prefix != "" {
		prefix = settings.Prefix
	}

	// Check for one-off easter eggs (scat, etc.).
	if checkEasterEggs(s, m) {
		return
	}

	// Auto-delete messages from lurkers after 0.5 seconds — but not lurk
	// commands themselves (otherwise they can never stop lurking).
	if !isLurkCommand(m.Content, prefix) {
		if commands.IsLurking(m.GuildID, m.Author.ID) {
			chID, msgID := m.ChannelID, m.ID
			safe.Go(deps.Logger, "lurkAutoDelete", func() {
				time.Sleep(500 * time.Millisecond)
				_ = s.ChannelMessageDelete(chID, msgID)
			})
		}
	}

	// Ensure the guild row exists so the dashboard guild picker sees this
	// server immediately.
	if err := ensureGuildRecord(ctx, deps.GuildRepo, deps.Logger, m.GuildID, "", 0); err != nil {
		return
	}

	// Log the message content only when the guild has explicitly opted in.
	// Message-content logging is off by default: it stores every message and,
	// left unbounded, both bloats the audit table and is a privacy liability.
	// Guilds that enable it are pruned by the retention job (see bot.App).
	if sErr == nil && settings.MessageLoggingEnabled {
		if _, err := deps.AuditRepo.Create(ctx, dto.CreateAuditLogParams{
			GuildID:  m.GuildID,
			Action:   "MESSAGE_CREATE",
			ActorID:  m.Author.ID,
			TargetID: m.ChannelID,
			Metadata: []byte(fmt.Sprintf(`{"message_id":%q,"content":%q,"channel_id":%q}`, m.ID, m.Content, m.ChannelID)),
		}); err != nil {
			deps.Logger.Error("Failed to create audit log", "error", err)
		}
	}

	_ = settings

	// Check for command
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	// Commands can be chained together with "&&", e.g. "-spark && -smoke".
	// Each segment is trimmed and dispatched in order; a failing segment does
	// not stop the remaining ones. A "sleep" segment pauses the chain for its
	// duration, e.g. "-nsfw && -sleep 5s && -sfw".
	key := sleepKey{guildID: m.GuildID, userID: m.Author.ID}

	// A "-sleep" from an earlier message may still be pending for this
	// (guild, user): wait out the remainder before their next command runs.
	if rem := tracker.pop(key); rem > 0 {
		time.Sleep(rem)
	}

	chain := strings.Split(m.Content, "&&")
	for _, segment := range chain {
		if delay, ok := sleepSegmentDelay(segment, prefix); ok {
			if delay > 0 {
				time.Sleep(delay)
				tracker.set(key, delay)
			}
			continue
		}
		dispatchCommand(ctx, s, m, prefix, segment, cmdHandler)
	}
}

// maxChainSleep caps how long a single "-sleep" segment may pause a command
// chain, so a typo like "-sleep 1h" can't pin a handler goroutine for a long
// stretch.
const maxChainSleep = 5 * time.Minute

// sleepSegmentDelay recognizes a "-sleep <duration>" segment within a command
// chain and returns the pause it requests. It returns ok=false for any other
// segment (including a malformed sleep) so normal dispatch handles it.
func sleepSegmentDelay(segment, prefix string) (time.Duration, bool) {
	seg := strings.TrimSpace(segment)
	if !strings.HasPrefix(seg, prefix) {
		return 0, false
	}

	fields := strings.Fields(strings.TrimPrefix(seg, prefix))
	if len(fields) == 0 || !strings.EqualFold(fields[0], "sleep") {
		return 0, false
	}

	var total time.Duration
	for _, raw := range fields[1:] {
		d, err := parseSleepDuration(raw)
		if err != nil {
			return 0, false
		}
		total += d
		if total > maxChainSleep {
			total = maxChainSleep
		}
	}
	if total <= 0 {
		return 0, false
	}
	return total, true
}

// sleepKey identifies the target of a pending sleep: a user in a guild (DMs
// use an empty guild).
type sleepKey struct {
	guildID string
	userID  string
}

// sleepTracker records pending "-sleep" deadlines per (guild, user) so a sleep
// also delays that user's next command message in the same guild, not just the
// rest of the current chain. Handlers run concurrently, so all access is
// mutex-guarded.
type sleepTracker struct {
	mu        sync.Mutex
	deadlines map[sleepKey]time.Time
}

func newSleepTracker() *sleepTracker {
	return &sleepTracker{deadlines: make(map[sleepKey]time.Time)}
}

// pop returns the remaining wait for a key (0 if none or expired) and clears
// it. The deadline is consumed by the first command that comes after it, so a
// stale entry never lingers once that command runs.
func (t *sleepTracker) pop(key sleepKey) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	deadline, ok := t.deadlines[key]
	if !ok {
		return 0
	}
	delete(t.deadlines, key)
	if rem := time.Until(deadline); rem > 0 {
		return rem
	}
	return 0
}

// set records a pending sleep deadline for a key.
func (t *sleepTracker) set(key sleepKey, d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.deadlines[key] = time.Now().Add(d)
	t.pruneLocked()
}

// pruneLocked drops expired entries so users who sleep and then never run
// another command don't leak map entries indefinitely.
func (t *sleepTracker) pruneLocked() {
	now := time.Now()
	for k, deadline := range t.deadlines {
		if !deadline.After(now) {
			delete(t.deadlines, k)
		}
	}
}

// parseSleepDuration parses a sleep argument: a Go duration such as "5s" or
// "1m30s", or a bare number of seconds like "5".
func parseSleepDuration(raw string) (time.Duration, error) {
	if d, err := time.ParseDuration(raw); err == nil {
		return d, nil
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second, nil
	}
	return 0, fmt.Errorf("invalid duration %q", raw)
}

// dispatchCommand runs a single command segment (already extracted from any
// "&&" chain) if it is a valid command invocation.
func dispatchCommand(
	ctx context.Context,
	s *discordgo.Session,
	m *discordgo.MessageCreate,
	prefix string,
	segment string,
	cmdHandler *commands.CommandHandler,
) {
	segment = strings.TrimSpace(segment)
	if !strings.HasPrefix(segment, prefix) {
		return
	}

	args := strings.Fields(strings.TrimPrefix(segment, prefix))
	if len(args) == 0 {
		return
	}

	if err := cmdHandler.Handle(ctx, s, m, args[0], args[1:]); err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %v", err))
		return
	}

	// Non-moderation commands count toward the donation card cadence.
	if !commands.IsModerationCommand(args[0]) {
		cmdHandler.MaybeShowDonatePrompt(s, m.ChannelID)
	}
}

// isLurkCommand reports whether a raw message content is a lurk or lurkers
// command using the given prefix. This lets lurkers toggle their state without
// the auto-delete kicking in.
func isLurkCommand(content, prefix string) bool {
	low := strings.ToLower(strings.TrimSpace(content))
	if prefix != "" && strings.HasPrefix(low, prefix) {
		low = strings.TrimSpace(strings.TrimPrefix(low, prefix))
	}
	return low == "lurk" || strings.HasPrefix(low, "lurk ") ||
		low == "lurkers" || strings.HasPrefix(low, "lurkers ")
}
