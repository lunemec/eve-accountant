package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
)

func (h *discordHandler) MonthlyBalanceBelowThresholdMessage(ctx context.Context, notification aggregate.MonthlyBalanceNotification) {
	notificationMsg := fmt.Sprintf(
		"`%s` < `%s`\n\n%s: `%s`\n%s: `%s`\n\nFor more details run:\n`!isk by division`\n`!isk by type`",
		humanize.FormatFloat(floatFormat, float64(notification.Balance.Balance())),
		humanize.FormatFloat(floatFormat, float64(notification.Threshold)),
		incomeMsg,
		humanize.FormatFloat(floatFormat, float64(notification.Balance.Income)),
		expensesMsg,
		humanize.FormatFloat(floatFormat, float64(notification.Balance.Expenses)),
	)
	_, err := h.discord.ChannelMessageSendEmbed(h.channelID, &discordgo.MessageEmbed{
		Title: fmt.Sprintf(
			"%s %s",
			monthlyBalanceNotificationMsg,
			titleWithDate(notification.DateStart, notification.DateEnd),
		),
		Description: notificationMsg,
		Color:       0xff0000,
	})
	if err != nil {
		h.error(err, h.channelID)
		return
	}
}
