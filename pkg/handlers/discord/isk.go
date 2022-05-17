package discord

import (
	"fmt"
	"strings"
	"time"

	balanceDomainAggrgate "github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

// iskHandler will be called every time a new
// message is created on any channel that the autenticated bot has access to.
func (h *discordHandler) iskHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// React before starting the balance calculation (it takes quite few seconds to fetch everything).
	err := h.discord.MessageReactionAdd(m.ChannelID, m.ID, `⏱️`)
	if err != nil {
		h.error(errors.Wrap(err, "error reacting with :stopwatch: emoji"), m.ChannelID)
	}

	dateStart, dateEnd, err := h.parseDateStartDateEnd(args)
	if err != nil {
		h.error(err, m.ChannelID)
		return
	}

	balance, err := h.accountantSvc.Balance(h.ctx, dateStart, dateEnd)
	if err != nil {
		h.error(errors.Wrap(err, "error calculating balance"), m.ChannelID)
		return
	}

	for _, messages := range h.iskMessages(dateStart, dateEnd, balance) {
		_, err = h.discord.ChannelMessageSendEmbed(m.ChannelID, messages)
		if err != nil {
			h.error(errors.Wrap(err, "error sending balance message"), m.ChannelID)
			return
		}
	}
}

func (h *discordHandler) iskMessages(dateStart, dateEnd time.Time, balance *balanceDomainAggrgate.Balance) []*discordgo.MessageEmbed {
	var (
		balanceDescription strings.Builder
	)

	balanceDescription.WriteString(
		fmt.Sprintf(
			"`%s`\n\n%s: `%s`\n%s: `%s`\n\n%s",
			humanize.FormatFloat(floatFormat, float64(balance.Balance())),
			incomeMsg,
			humanize.FormatFloat(floatFormat, float64(balance.Income)),
			expensesMsg,
			humanize.FormatFloat(floatFormat, float64(balance.Expenses)),
			forMoreDetailsMsg,
		),
	)

	title := fmt.Sprintf("%s %s", balanceMsg, titleWithDate(dateStart, dateEnd))
	var messages = []*discordgo.MessageEmbed{
		{
			Title:       title,
			Description: balanceDescription.String(),
			Color:       0xffffff,
		},
	}

	return messages
}
