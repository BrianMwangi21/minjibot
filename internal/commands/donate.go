package commands

import (
	"fmt"
	"math/rand/v2"

	"github.com/bwmarrin/discordgo"
)

// DonateURL is where users can support the bot.
const DonateURL = "https://buymeacoffee.com/kruegenn"

// WebsiteURL is the bot's public dashboard/website.
const WebsiteURL = "https://minji-bot.netlify.app/"

// DonateEveryNMin / DonateEveryNMax bound the donation prompt cadence.
const (
	DonateEveryNMin = 13
	DonateEveryNMax = 15
)

// IsModerationCommand reports whether cmd name is a moderation/admin action.
// These never surface the donation prompt.
func IsModerationCommand(cmd string) bool {
	switch cmd {
	case "ban", "hardban", "softban", "kick", "purge", "nuke", "timeout", "warn",
		"history", "audit", "role", "fn", "nick", "jail", "unjail", "staffstrip",
		"hide", "reveal", "lockdown", "nsfw", "sfw", "slowmode", "topic", "channel",
		"denyperm", "imute", "gifmute":
		return true
	default:
		return false
	}
}

// DonatePromptDue advances the non-moderation command counter and reports
// whether the donation card should be shown now: once every randomly-chosen
// 13-15 commands. It is safe for concurrent callers.
func (h *CommandHandler) DonatePromptDue() bool {
	h.donateMu.Lock()
	defer h.donateMu.Unlock()
	if h.donateTarget == 0 {
		h.donateTarget = donateTargetN()
	}
	h.donateCount++
	if h.donateCount < h.donateTarget {
		return false
	}
	h.donateCount = 0
	h.donateTarget = donateTargetN()
	return true
}

// MaybeShowDonatePrompt posts the donation card into channelID once the
// 13-15 command cadence is hit. Send failures are ignored: they are not worth a
// user-facing error, and the counter has already reset.
func (h *CommandHandler) MaybeShowDonatePrompt(s *discordgo.Session, channelID string) {
	if !h.DonatePromptDue() {
		return
	}
	_, _ = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{donateEmbed()},
		Components: donateComponents(),
	})
}

// donateTargetN returns a random cadence in [DonateEveryNMin, DonateEveryNMax].
func donateTargetN() int {
	return DonateEveryNMin + rand.IntN(DonateEveryNMax-DonateEveryNMin+1)
}

func donateMessageCommandHandler(s *discordgo.Session, m *discordgo.MessageCreate, _ []string) error {
	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{donateEmbed()},
		Components: donateComponents(),
	})
	return err
}

func donateSlashCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{donateEmbed()},
			Components: donateComponents(),
		},
	})
}

func donateEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Color:       0xFFDD00,
		Title:       "Support MinjiBot",
		Description: fmt.Sprintf("If MinjiBot has made your server better, consider buying me a coffee to keep it running. Every bit helps!\n\n[Buy me a coffee](%s)", DonateURL),
	}
}

func donateComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "Buy me a coffee",
					Style: discordgo.LinkButton,
					URL:   DonateURL,
				},
			},
		},
	}
}
