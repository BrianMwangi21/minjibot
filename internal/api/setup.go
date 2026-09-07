// Server-setup endpoints: run a TOML provisioning document against a guild via
// the attached bot session. All handlers require a session cookie AND that the
// session's Discord user holds Administrator in the target guild.
package api

import (
	"context"
	"net/http"

	"github.com/bwmarrin/discordgo"
	authsvc "github.com/kibetnathan/minjibot/internal/services/auth"
	"github.com/kibetnathan/minjibot/internal/setup"
	"github.com/labstack/echo/v5"
)

// setupRequest is the JSON body accepted by the setup endpoints.
type setupRequest struct {
	Toml            string `json:"toml"`
	NotifyChannelID string `json:"notify_channel_id"`
}

// setupHandlers implements the provisioning endpoints.
type setupHandlers struct {
	sess  *authsvc.SessionManager
	authz *guildAuthz
	app   *App
}

func (a *App) registerSetupRoutes(group *echo.Group, h *setupHandlers) {
	group.POST("/guilds/:guildId/setup", h.runSetup)
	group.POST("/guilds/:guildId/setup/dry-run", h.dryRunSetup)
	group.GET("/guilds/:guildId/channels", h.listChannels)
}

// requireGuildAdmin is requireGuildPerm but for the Administrator bit.
func (h *setupHandlers) requireGuildAdmin(c *echo.Context, guildID string) (string, bool) {
	sess, ok := resolveSession(c, h.sess)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return "", false
	}
	if sess.AccessToken == "" {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "reauthenticate"})
		return "", false
	}
	allowed, err := h.authz.canAdmin(c.Request().Context(), sess.UserID, sess.AccessToken, guildID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "reauthenticate"})
		return "", false
	}
	if !allowed {
		c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		return "", false
	}
	return sess.UserID, true
}

func (h *setupHandlers) botReady() bool {
	return h.app.Session != nil && h.app.SetupRunner != nil
}

// runSetup provisions a guild from the posted document. Runs admin-gated in the
// acting user's name so command-permission checks and audit logging attribute
// correctly.
func (h *setupHandlers) runSetup(c *echo.Context) error {
	guildID := c.Param("guildId")
	if guildID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "guildId required"})
	}
	userID, ok := h.requireGuildAdmin(c, guildID)
	if !ok {
		return nil
	}
	if !h.botReady() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "bot not connected"})
	}

	var req setupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	cfg, err := setup.Parse([]byte(req.Toml))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	notifyChannelID := req.NotifyChannelID
	if notifyChannelID == "" {
		notifyChannelID = defaultSystemChannel(guildID, h.app.Session)
	}
	if notifyChannelID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "notify_channel_id is required when the guild has no accessible system channel"})
	}

	notify := setup.Notify{ChannelID: notifyChannelID, AuthorID: userID, AuthorName: h.username(c.Request().Context(), userID)}
	steps := h.app.SetupRunner.Run(c.Request().Context(), guildID, cfg, notify)
	return c.JSON(http.StatusOK, map[string]any{"steps": steps})
}

// dryRunSetup parses and validates the document without applying anything.
func (h *setupHandlers) dryRunSetup(c *echo.Context) error {
	guildID := c.Param("guildId")
	if guildID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "guildId required"})
	}
	if _, ok := h.requireGuildAdmin(c, guildID); !ok {
		return nil
	}

	var req setupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	if err := setup.Validate([]byte(req.Toml)); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"valid": true})
}

// username best-effort resolves a Discord username for the acting user. Empty
// on failure; senders fall back to the ID.
func (h *setupHandlers) username(ctx context.Context, userID string) string {
	if !h.botReady() {
		return ""
	}
	u, err := h.app.Session.User(userID)
	if err != nil {
		return ""
	}
	return u.Username
}

// channelView is what a guild channel looks like to the frontend picker.
type channelView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	ParentID string `json:"parent_id"`
}

// listChannels returns the guild's channels for the notify-channel picker.
func (h *setupHandlers) listChannels(c *echo.Context) error {
	guildID := c.Param("guildId")
	if guildID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "guildId required"})
	}
	if _, ok := h.requireGuildAdmin(c, guildID); !ok {
		return nil
	}
	if !h.botReady() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "bot not connected"})
	}
	channels, err := h.app.Session.GuildChannels(guildID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "could not load channels"})
	}
	out := make([]channelView, 0, len(channels))
	for _, ch := range channels {
		if !isManageableChannelType(ch.Type) {
			continue
		}
		out = append(out, channelView{ID: ch.ID, Name: ch.Name, Type: int(ch.Type), ParentID: ch.ParentID})
	}
	return c.JSON(http.StatusOK, out)
}

// defaultSystemChannel picks a sensible notify channel when the request didn't
// specify one: the guild system channel, else the first text channel.
func defaultSystemChannel(guildID string, s *discordgo.Session) string {
	if g, err := s.Guild(guildID); err == nil && g.SystemChannelID != "" {
		return g.SystemChannelID
	}
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return ""
	}
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildText {
			return ch.ID
		}
	}
	return ""
}

func isManageableChannelType(t discordgo.ChannelType) bool {
	switch t {
	case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildVoice,
		discordgo.ChannelTypeGuildCategory, discordgo.ChannelTypeGuildNews,
		discordgo.ChannelTypeGuildForum:
		return true
	}
	return false
}
