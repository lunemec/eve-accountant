package notifier

import (
	"context"
	"time"

	"github.com/lunemec/eve-accountant/pkg/services/accountant"

	"go.uber.org/zap"
)

type notifierHandler struct {
	ctx            context.Context
	log            *zap.Logger
	checkInterval  time.Duration
	notifyInterval time.Duration
	accountantSvc  accountant.Service
}

type doctrineContractsService interface {
}

func New(
	ctx context.Context,
	log *zap.Logger,
	checkInterval, notifyInterval time.Duration,
	accountantSvc accountant.Service,
) *notifierHandler {
	notifier := notifierHandler{
		ctx:            ctx,
		log:            log,
		checkInterval:  checkInterval,
		notifyInterval: notifyInterval,
		accountantSvc:  accountantSvc,
	}
	return &notifier
}

func (n *notifierHandler) Start() {
	n.log.Info("Notifier handler started.")
	ticker := time.NewTicker(n.checkInterval)
	for {
		select {
		case <-ticker.C:
			err := n.tick()
			if err != nil {
				n.log.Error("notifier error", zap.Error(err))
			}
		case <-n.ctx.Done():
			return
		}
	}
}

// tick is called every ticker interval.
func (n *notifierHandler) tick() error {
	return nil
}
