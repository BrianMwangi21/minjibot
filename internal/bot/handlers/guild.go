package handlers

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/kibetnathan/minjibot/internal/ports/dto"
	"github.com/kibetnathan/minjibot/internal/ports/repository"
	"github.com/kibetnathan/minjibot/internal/safe"
)

type GuildHandlerDeps struct {
	Logger    *slog.Logger
	GuildRepo repository.GuildRepository
}

// RegisterGuildHandler records the guild the moment the bot joins it, so the
// dashboard's guild picker reflects new servers immediately instead of waiting
// for the first message or command.
func RegisterGuildHandler(s *discordgo.Session, deps GuildHandlerDeps) {
	s.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		defer safe.Recover(deps.Logger, "onGuildCreate")
		if g == nil || g.Guild == nil {
			return
		}
		if err := ensureGuildRecord(context.Background(), deps.GuildRepo, deps.Logger, g.ID, g.Name, int32(g.PremiumTier)); err != nil {
			deps.Logger.Error("Failed to record guild join", "error", err, "guild_id", g.ID)
		}
	})
}

// ensureGuildRecord creates the guild row on first sight. It is a no-op when
// the row already exists, so any event path (message, interaction, message
// delete, guild join) can call it safely.
func ensureGuildRecord(ctx context.Context, repo repository.GuildRepository, logger *slog.Logger, guildID, name string, tier int32) error {
	if repo == nil {
		return nil
	}
	if _, err := repo.GetByID(ctx, guildID); err == nil {
		return nil
	}
	_, err := repo.Create(ctx, dto.CreateGuildParams{ID: guildID, Name: name, PremiumTier: tier})
	if err != nil {
		logger.Error("Failed to create guild", "error", err, "guild_id", guildID)
	}
	return err
}
