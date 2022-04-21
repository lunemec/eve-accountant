package discord

import (
	"context"
	"fmt"

	"github.com/lunemec/eve-accountant/pkg/services/accountant"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type discordHandler struct {
	ctx     context.Context
	log     *zap.Logger
	discord *discordgo.Session

	accountantSvc accountant.Service
}

func New(ctx context.Context, log *zap.Logger, discord *discordgo.Session, accountantSvc accountant.Service) *discordHandler {
	return &discordHandler{
		ctx:           ctx,
		log:           log,
		discord:       discord,
		accountantSvc: accountantSvc,
	}
}

func (h *discordHandler) Start() {
	h.log.Info("Discord handler started.")
	// Add handler to listen for "!help" messages as help message.
	h.discord.AddHandler(h.helpHandler)
	// Add handler to listen for "!isk" messages for isk overview.
	h.discord.AddHandler(h.iskHandler)

	<-h.ctx.Done()
}

func (h *discordHandler) error(errIn error, m *discordgo.MessageCreate) {
	h.log.Error("error in discord handler call", zap.Error(errIn))
	msg := fmt.Sprintf("Sorry, some error happened: %s", errIn.Error())
	_, err := h.discord.ChannelMessageSend(m.ChannelID, msg)
	if err != nil {
		h.log.Error("error responding with error", zap.Error(err), zap.NamedError("original_error", errIn))
	}
}
