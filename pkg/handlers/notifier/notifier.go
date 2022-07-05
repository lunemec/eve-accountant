package notifier

import (
	"context"
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	"github.com/lunemec/eve-accountant/pkg/services/accountant"
	"github.com/pkg/errors"

	"go.uber.org/zap"
)

type sendMsgFunc func(context.Context, aggregate.MonthlyBalanceNotification)

type notifierHandler struct {
	ctx            context.Context
	log            *zap.Logger
	checkInterval  time.Duration
	notifyInterval time.Duration
	accountantSvc  accountant.Service

	sendMsgFunc sendMsgFunc

	lastNotify time.Time
}

func New(
	ctx context.Context,
	log *zap.Logger,
	checkInterval, notifyInterval time.Duration,
	accountantSvc accountant.Service,
	sendMsgFunc sendMsgFunc,
) *notifierHandler {
	notifier := notifierHandler{
		ctx:            ctx,
		log:            log,
		checkInterval:  checkInterval,
		notifyInterval: notifyInterval,
		accountantSvc:  accountantSvc,
		sendMsgFunc:    sendMsgFunc,
	}
	return &notifier
}

func (n *notifierHandler) Start() {
	n.log.Info("Notifier handler started.")
	// Tick 1x at startup.
	err := n.tick()
	if err != nil {
		n.log.Error("notifier error", zap.Error(err))
	}

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
	now := time.Now()
	if now.After(n.lastNotify.Add(n.notifyInterval)) {
		err := n.notify()
		if err != nil {
			return err
		}

		n.lastNotify = now
	}
	return nil
}

func (n *notifierHandler) notify() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	notify, balance, err := n.accountantSvc.MonthlyBalanceBelowThreshold(ctx)
	if err != nil {
		return errors.Wrap(err, "error")
	}
	if notify {
		n.sendMsgFunc(ctx, balance)
	}

	return nil
}
