package discord

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// helpHandler will be called every time a new
// message is created on any channel that the autenticated bot has access to.
func (h *discordHandler) helpHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	msg := "I'll keep track of your transactions and help with corporation ISK. \n\n" +
		"Here is the list of commands you can use:\n" +
		"`!help` - shows this help message\n" +
		"`!isk` - top level balance overview\n" +
		"`!isk by division` - balance overview grouped by each division\n" +
		"`!isk by type` - balance overview grouped by transaction type"

	_, err := h.discord.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		Title: "Hello, I'm your accountant.",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://i.imgur.com/ZwUn8DI.jpg",
		},
		Color:       0x00ff00,
		Description: msg,
		Timestamp:   time.Now().Format(time.RFC3339), // Discord wants ISO8601; RFC3339 is an extension of ISO8601 and should be completely compatible.
	})
	if err != nil {
		if err != nil {
			h.log.Error("error sending message for !help", zap.Error(err))
			return
		}
	}
}
