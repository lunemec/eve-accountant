package discord

import (
	"fmt"
	"sort"
	"strings"
	"time"

	balanceDomainAggrgate "github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	balanceDomainEntity "github.com/lunemec/eve-accountant/pkg/domain/balance/entity"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

// iskHandler will be called every time a new
// message is created on any channel that the autenticated bot has access to.
func (h *discordHandler) iskHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content != "!isk" && m.Content != "!accountant" {
		return
	}
	// React before starting the balance calculation (it takes quite few seconds to fetch everything).
	err := h.discord.MessageReactionAdd(m.ChannelID, m.ID, `⏱️`)
	if err != nil {
		h.error(errors.Wrap(err, "error reacting with :stopwatch: emoji"), m)
	}
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()

	dateStart := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
	dateEnd := dateStart.AddDate(0, 1, -1)

	balance, err := h.accountantSvc.Balance(dateStart, dateEnd)
	if err != nil {
		h.error(errors.Wrap(err, "error calculating balance"), m)
		return
	}

	for _, messages := range h.iskMessages(balance) {
		_, err = h.discord.ChannelMessageSendEmbed(m.ChannelID, messages)
		if err != nil {
			h.error(errors.Wrap(err, "error sending balance message"), m)
			return
		}
	}
	return
}

type balanceRow struct {
	Type   balanceDomainEntity.RefType
	Amount balanceDomainEntity.Amount
}

func (h *discordHandler) iskMessages(balance *balanceDomainAggrgate.Balance) []*discordgo.MessageEmbed {
	var descriptionRowData = make([]balanceRow, 0, len(balance.AmountByType))

	for refType, amount := range balance.AmountByType {
		descriptionRowData = append(descriptionRowData, balanceRow{
			Type:   refType,
			Amount: amount,
		})
	}
	sort.Slice(descriptionRowData, func(i, j int) bool {
		return descriptionRowData[i].Amount > descriptionRowData[j].Amount
	})

	var (
		balanceDescription strings.Builder
		totalIncome        float64
		totalExpenses      float64
		income             strings.Builder
		expenses           strings.Builder

		format = "#\u202F###."
	)

	income.WriteString("```")
	expenses.WriteString("```")
	for _, descriptionRow := range descriptionRowData {
		if descriptionRow.Amount > 0 {
			totalIncome += float64(descriptionRow.Amount)
			income.WriteString(fmt.Sprintf("%s  %s\n", humanize.FormatFloat(format, float64(descriptionRow.Amount)), string(descriptionRow.Type)))
		}
		if descriptionRow.Amount < 0 {
			totalExpenses += float64(descriptionRow.Amount)
			expenses.WriteString(fmt.Sprintf("%s  %s\n", humanize.FormatFloat(format, float64(descriptionRow.Amount)), string(descriptionRow.Type)))
		}
	}
	income.WriteString("```")
	expenses.WriteString("```")

	balanceDescription.WriteString(
		fmt.Sprintf(
			"Income: `%s`\nExpenses: `%s`\n\n Balance: `%s`\n\nIncome Raw: `%s`\nExpenses Raw: `%s`\n\nBalance Raw: `%s`",
			humanize.FormatFloat(format, totalIncome),
			humanize.FormatFloat(format, totalExpenses),
			humanize.FormatFloat(format, totalIncome+totalExpenses), // We must add because totalExpense is negative.
			humanize.FormatFloat(format, float64(balance.Income)),
			humanize.FormatFloat(format, float64(balance.Expenses)),
			humanize.FormatFloat(format, float64(balance.Income)+float64(balance.Expenses)), // We must add because Expenses is negative.
		),
	)

	var messages = []*discordgo.MessageEmbed{
		{
			Title:       "Balance",
			Description: balanceDescription.String(),
			Color:       0x0000ff,
		},
		{
			Title:       "Income",
			Description: income.String(),
			Color:       0x00ff00,
		},
		{
			Title:       "Expenses",
			Description: expenses.String(),
			Color:       0xff0000,
		},
	}

	return messages
}
