package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lunemec/eve-accountant/pkg/services/accountant"
	"github.com/pkg/errors"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var (
	floatFormat                   = "#\u202F###."
	balanceMsg                    = ":euro: Balance"
	incomeMsg                     = ":chart_with_upwards_trend: Income"
	expensesMsg                   = ":chart_with_downwards_trend: Expenses"
	monthlyBalanceNotificationMsg = ":exclamation: Monthly Balance Low"
	forMoreDetailsMsg             = "For more details run:\n\n`!isk by division`\n`!isk by type`\n`!isk YYYY-MM-DD YYYY-MM-DD`\n`!isk by division YYYY-MM-DD YYYY-MM-DD`\n`!isk by type YYYY-MM-DD YYYY-MM-DD`"
)

type discordHandler struct {
	ctx       context.Context
	log       *zap.Logger
	discord   *discordgo.Session
	channelID string

	accountantSvc accountant.Service
}

func New(
	ctx context.Context,
	log *zap.Logger,
	discord *discordgo.Session,
	channelID string,
	accountantSvc accountant.Service,
) *discordHandler {
	return &discordHandler{
		ctx:           ctx,
		log:           log,
		discord:       discord,
		channelID:     channelID,
		accountantSvc: accountantSvc,
	}
}

func (h *discordHandler) Start() {
	h.log.Info("Discord handler started.")
	h.discord.AddHandler(h.router)
	<-h.ctx.Done()
}

func (h *discordHandler) router(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself.
	if m.Author.ID == s.State.User.ID {
		return
	}
	if ok, _ := h.command("!help", m.Content); ok {
		h.helpHandler(s, m, nil)
	}
	if ok, args := h.command("!isk by division", m.Content); ok {
		h.iskByDivisionHandler(s, m, args)
		return
	}
	if ok, args := h.command("!isk by type", m.Content); ok {
		h.iskByTypeHandler(s, m, args)
		return
	}
	if ok, args := h.command("!isk", m.Content); ok {
		h.iskHandler(s, m, args)
		return
	}
}

func (h *discordHandler) error(errIn error, channelID string) {
	h.log.Error("error in discord handler call", zap.Error(errIn))
	msg := fmt.Sprintf("Sorry, some error happened: %s", errIn.Error())
	_, err := h.discord.ChannelMessageSend(channelID, msg)
	if err != nil {
		h.log.Error("error responding with error", zap.Error(err), zap.NamedError("original_error", errIn))
	}
}

func (h *discordHandler) command(command string, messageContent string) (bool, []string) {
	if !strings.HasPrefix(messageContent, command) {
		return false, nil
	}
	paramsStr := strings.TrimPrefix(messageContent, command)
	paramsStr = strings.TrimSpace(paramsStr)
	params := strings.Split(paramsStr, " ")

	return true, params
}

func (h *discordHandler) parseDateStartDateEnd(params []string) (time.Time, time.Time, error) {
	var (
		err                error
		dateStart, dateEnd time.Time
	)
	if len(params) == 2 {
		format := "2006-01-02"
		dateStart, err = time.Parse(format, params[0])
		if err != nil {
			return dateStart, dateEnd, errors.Wrap(err, "unknown date format, use YYYY-MM-DD")
		}
		dateEnd, err = time.Parse(format, params[1])
		if err != nil {
			return dateStart, dateEnd, errors.Wrap(err, "unknown date format, use YYYY-MM-DD")
		}
	} else {
		now := time.Now()
		currentYear, currentMonth, _ := now.Date()

		dateStart = time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
		dateEnd = dateStart.AddDate(0, 1, -1)
	}
	return dateStart, dateEnd, nil
}

func titleWithDate(dateStart, dateEnd time.Time) string {
	var title string
	if dateStart.Month() == dateEnd.Month() {
		currentYear, currentMonth, _ := dateStart.Date()
		title = fmt.Sprintf("for %s %d", currentMonth.String(), currentYear)
	} else {
		startYear, startMonth, _ := dateStart.Date()
		endYear, endMonth, _ := dateEnd.Date()
		title = fmt.Sprintf("for %s %d - %s %d", startMonth.String(), startYear, endMonth.String(), endYear)
	}

	return title
}
