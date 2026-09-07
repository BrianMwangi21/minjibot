// Dashboard log endpoints: guild picker, deleted messages, and moderation
// actions. All handlers require a valid session cookie (Discord OAuth) AND that
// the session's Discord user is on the dashboard admin allowlist.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/kibetnathan/minjibot/internal/domain/auditlog"
	"github.com/kibetnathan/minjibot/internal/domain/deletedmessage"
	"github.com/kibetnathan/minjibot/internal/ports/repository"
	authsvc "github.com/kibetnathan/minjibot/internal/services/auth"
	"github.com/labstack/echo/v5"
)

// logHandlers bundles the dependencies shared by the dashboard log endpoints.
type logHandlers struct {
	sess    *authsvc.SessionManager
	authz   *guildAuthz
	guilds  repository.GuildRepository
	audits  repository.AuditLogRepository
	deletes repository.DeletedMessageRepository
}

func (a *App) registerLogRoutes(group *echo.Group, h *logHandlers) {
	group.GET("/guilds", h.listGuilds)
	group.GET("/logs/deleted", h.listDeletedMessages)
	group.GET("/logs/actions", h.listModActions)
}

// requireGuildPerm gates a dashboard data endpoint for a single guild: 401
// without a valid session, and 403 when the session's Discord user holds no
// moderation permission in that guild.
func (h *logHandlers) requireGuildPerm(c *echo.Context, guildID string) (string, bool) {
	sess, ok := resolveSession(c, h.sess)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return "", false
	}
	if sess.AccessToken == "" {
		// Session predates guild-based authorization; requires re-login.
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "reauthenticate"})
		return "", false
	}
	allowed, err := h.authz.canView(c.Request().Context(), sess.UserID, sess.AccessToken, guildID)
	if err != nil {
		// Discord rejected/revoked the token (or is unreachable): force a
		// fresh login rather than guessing at permissions.
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "reauthenticate"})
		return "", false
	}
	if !allowed {
		c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		return "", false
	}
	return sess.UserID, true
}

type guildSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PremiumTier     int32  `json:"premium_tier"`
	DeletedMessages int64  `json:"deleted_messages"`
	ModActions      int64  `json:"mod_actions"`
}

// listGuilds returns every guild the bot has data for that the session user
// holds moderation permissions in, plus per-guild deleted-message and
// moderation-action counts, so the dashboard can offer a picker.
func (h *logHandlers) listGuilds(c *echo.Context) error {
	sess, ok := resolveSession(c, h.sess)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	if sess.AccessToken == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "reauthenticate"})
	}
	allowed, err := h.authz.authorizableGuilds(c.Request().Context(), sess.UserID, sess.AccessToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "reauthenticate"})
	}
	ctx := c.Request().Context()

	guilds, err := h.guilds.List(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	deletedCounts, _ := h.deletes.CountForAllGuilds(ctx)
	actionCounts, _ := h.audits.CountForAllGuilds(ctx)

	out := make([]guildSummary, 0, len(guilds))
	for _, g := range guilds {
		if _, ok := allowed[g.ID]; !ok {
			continue
		}
		out = append(out, guildSummary{
			ID:              g.ID,
			Name:            g.Name,
			PremiumTier:     g.PremiumTier,
			DeletedMessages: deletedCounts[g.ID],
			ModActions:      actionCounts[g.ID],
		})
	}
	return c.JSON(http.StatusOK, out)
}

// listDeletedMessages returns deleted messages for a guild (required query
// param guild_id), newest first.
func (h *logHandlers) listDeletedMessages(c *echo.Context) error {
	guildID := c.QueryParam("guild_id")
	if guildID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "guild_id required"})
	}
	if _, ok := h.requireGuildPerm(c, guildID); !ok {
		return nil
	}
	ctx := c.Request().Context()

	limit, offset := pageParams(c)

	msgs, err := h.deletes.ListForGuild(ctx, guildID, int32(limit), int32(offset))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	total, _ := h.deletes.CountForGuild(ctx, guildID)

	out := make([]deletedMessageView, 0, len(msgs))
	for _, dm := range msgs {
		out = append(out, deletedMessageViewFrom(dm))
	}
	return c.JSON(http.StatusOK, map[string]any{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"items":  out,
	})
}

// listModActions returns moderation actions (audit logs) for a guild (required
// query param guild_id), newest first, excluding message-created noise.
func (h *logHandlers) listModActions(c *echo.Context) error {
	guildID := c.QueryParam("guild_id")
	if guildID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "guild_id required"})
	}
	if _, ok := h.requireGuildPerm(c, guildID); !ok {
		return nil
	}
	ctx := c.Request().Context()

	limit, offset := pageParams(c)

	logs, err := h.audits.ListForGuild(ctx, guildID, int32(limit), int32(offset))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	total, _ := h.audits.CountForGuild(ctx, guildID)

	out := make([]modActionView, 0, len(logs))
	for _, l := range logs {
		if strings.HasPrefix(l.Action, "MESSAGE_") {
			continue
		}
		out = append(out, modActionViewFrom(l))
	}
	return c.JSON(http.StatusOK, map[string]any{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"items":  out,
	})
}

func pageParams(c *echo.Context) (limit, offset int) {
	limit = 50
	offset = 0
	if v, err := strconv.Atoi(c.QueryParam("limit")); err == nil && v > 0 && v <= 200 {
		limit = v
	}
	if v, err := strconv.Atoi(c.QueryParam("offset")); err == nil && v >= 0 {
		offset = v
	}
	return limit, offset
}

type deletedMessageView struct {
	ID            int64           `json:"id"`
	ChannelID     string          `json:"channel_id"`
	MessageID     string          `json:"message_id"`
	AuthorID      string          `json:"author_id"`
	AuthorName    string          `json:"author_name"`
	Content       string          `json:"content"`
	Attachments   json.RawMessage `json:"attachments"`
	DeletedBy     string          `json:"deleted_by"`
	DeletedByName string          `json:"deleted_by_name"`
	CreatedAt     string          `json:"created_at"`
}

func deletedMessageViewFrom(dm deletedmessage.DeletedMessage) deletedMessageView {
	attachments := dm.Attachments
	if attachments == nil {
		attachments = []byte("null")
	}
	return deletedMessageView{
		ID:            dm.ID,
		ChannelID:     dm.ChannelID,
		MessageID:     dm.MessageID,
		AuthorID:      dm.AuthorID,
		AuthorName:    dm.AuthorName,
		Content:       dm.Content,
		Attachments:   json.RawMessage(attachments),
		DeletedBy:     dm.DeletedBy,
		DeletedByName: dm.DeletedByName,
		CreatedAt:     dm.CreatedAt.UTC().Format(timeFormat),
	}
}

type modActionView struct {
	ID         int64           `json:"id"`
	Action     string          `json:"action"`
	ActorID    string          `json:"actor_id"`
	ActorName  string          `json:"actor_name"`
	TargetID   string          `json:"target_id"`
	TargetName string          `json:"target_name"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  string          `json:"created_at"`
}

const timeFormat = "2006-01-02T15:04:05Z"

func modActionViewFrom(l auditlog.AuditLog) modActionView {
	meta := l.Metadata
	if meta == nil {
		meta = []byte("null")
	}
	return modActionView{
		ID:         l.ID,
		Action:     l.Action,
		ActorID:    l.ActorID,
		ActorName:  l.ActorName,
		TargetID:   l.TargetID,
		TargetName: l.TargetName,
		Metadata:   json.RawMessage(meta),
		CreatedAt:  l.CreatedAt.UTC().Format(timeFormat),
	}
}
