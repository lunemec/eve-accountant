package accountant

import (
	"context"
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
	"github.com/pkg/errors"
)

type Service interface {
	Balance(ctx context.Context, from, to time.Time) (*aggregate.Balance, error)
	BalanceByDivision(ctx context.Context, from, to time.Time) (*aggregate.BalanceByDivision, error)
	BalanceByType(ctx context.Context, from, to time.Time) (*aggregate.BalanceByType, error)
	BalanceByDayByDivisionByType(ctx context.Context, from, to time.Time) ([]*aggregate.BalanceByDivisionByType, error)
	MonthlyBalanceBelowThreshold(ctx context.Context) (bool, aggregate.MonthlyBalanceNotification, error)
}

type accountantService struct {
	balanceSvc              balance.Service
	monthlyBalanceThreshold entity.Amount
}

func New(balanceSvc balance.Service, monthlyBalanceThreshold entity.Amount) *accountantService {
	return &accountantService{
		balanceSvc:              balanceSvc,
		monthlyBalanceThreshold: monthlyBalanceThreshold,
	}
}

func (s *accountantService) Balance(ctx context.Context, from, to time.Time) (*aggregate.Balance, error) {
	return s.balanceSvc.Balance(ctx, from, to)
}

func (s *accountantService) BalanceByDayByDivisionByType(ctx context.Context, from, to time.Time) ([]*aggregate.BalanceByDivisionByType, error) {
	return s.balanceSvc.BalanceByDayByDivisionByType(ctx, from, to)
}

func (s *accountantService) BalanceByDivision(ctx context.Context, from, to time.Time) (*aggregate.BalanceByDivision, error) {
	return s.balanceSvc.BalanceByDivision(ctx, from, to)
}

func (s *accountantService) BalanceByType(ctx context.Context, from, to time.Time) (*aggregate.BalanceByType, error) {
	return s.balanceSvc.BalanceByType(ctx, from, to)
}

func (s *accountantService) MonthlyBalanceBelowThreshold(ctx context.Context) (bool, aggregate.MonthlyBalanceNotification, error) {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()

	dateStart := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
	dateEnd := dateStart.AddDate(0, 1, -1)

	monthlyBalanceNotification := aggregate.MonthlyBalanceNotification{
		Threshold: s.monthlyBalanceThreshold,
		DateStart: dateStart,
		DateEnd:   dateEnd,
	}

	balance, err := s.Balance(ctx, dateStart, dateEnd)
	if err != nil {
		return false, monthlyBalanceNotification, errors.Wrap(err, "error checking balance")
	}
	monthlyBalanceNotification.Balance = *balance

	if balance.Balance() < monthlyBalanceNotification.Threshold {
		return true, monthlyBalanceNotification, nil
	}
	return false, monthlyBalanceNotification, nil
}
