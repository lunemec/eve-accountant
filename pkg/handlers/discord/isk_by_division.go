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

// iskByDivision will be called every time a new
// message is created on any channel that the autenticated bot has access to.
func (h *discordHandler) iskByDivisionHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
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
	balance, err := h.accountantSvc.BalanceByDivision(h.ctx, dateStart, dateEnd)
	if err != nil {
		h.error(errors.Wrap(err, "error calculating balance"), m.ChannelID)
		return
	}

	for _, messages := range h.iskByDivisionMessages(dateStart, dateEnd, balance) {
		_, err = h.discord.ChannelMessageSendEmbed(m.ChannelID, messages)
		if err != nil {
			h.error(errors.Wrap(err, "error sending balance message"), m.ChannelID)
			return
		}
	}
}

type balanceByDivisionRow struct {
	Division balanceDomainEntity.DivisionName
	Amount   balanceDomainEntity.Amount
}

func (h *discordHandler) iskByDivisionMessages(dateStart, dateEnd time.Time, balance *balanceDomainAggrgate.BalanceByDivision) []*discordgo.MessageEmbed {
	var descriptionRowData = make([]balanceByDivisionRow, 0, len(balance.IncomeByDivision)+len(balance.ExpensesByDivision))

	for division, amount := range balance.IncomeByDivision {
		descriptionRowData = append(descriptionRowData, balanceByDivisionRow{
			Division: division,
			Amount:   amount,
		})
	}
	for division, amount := range balance.ExpensesByDivision {
		descriptionRowData = append(descriptionRowData, balanceByDivisionRow{
			Division: division,
			Amount:   amount,
		})
	}
	sort.Slice(descriptionRowData, func(i, j int) bool {
		return descriptionRowData[i].Amount > descriptionRowData[j].Amount
	})

	var (
		totalIncome   float64
		totalExpenses float64
		income        strings.Builder
		expenses      strings.Builder
	)

	income.WriteString("```")
	expenses.WriteString("```")
	for _, descriptionRow := range descriptionRowData {
		if descriptionRow.Amount > 0 {
			totalIncome += float64(descriptionRow.Amount)
			income.WriteString(fmt.Sprintf("%s  %s\n", humanize.FormatFloat(floatFormat, float64(descriptionRow.Amount)), string(descriptionRow.Division)))
		}
		if descriptionRow.Amount < 0 {
			totalExpenses += float64(descriptionRow.Amount)
			expenses.WriteString(fmt.Sprintf("%s  %s\n", humanize.FormatFloat(floatFormat, float64(descriptionRow.Amount)), string(descriptionRow.Division)))
		}
	}
	income.WriteString("```")
	expenses.WriteString("```")

	var messages = []*discordgo.MessageEmbed{
		{
			Title:       fmt.Sprintf("%s %s", incomeMsg, titleWithDate(dateStart, dateEnd)),
			Description: income.String(),
			Color:       0x00ff00,
		},
		{
			Title:       fmt.Sprintf("%s %s", expensesMsg, titleWithDate(dateStart, dateEnd)),
			Description: expenses.String(),
			Color:       0xff0000,
		},
	}

	return messages
}
