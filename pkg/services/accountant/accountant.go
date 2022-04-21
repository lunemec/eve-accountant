package accountant

import (
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
)

type Service interface {
	Balance(from, to time.Time) (*aggregate.Balance, error)
}

type accountantService struct {
	balanceSvc balance.Service
}

func New(balanceSvc balance.Service) *accountantService {
	return &accountantService{
		balanceSvc: balanceSvc,
	}
}

func (s *accountantService) Balance(from, to time.Time) (*aggregate.Balance, error) {
	return s.balanceSvc.Balance(from, to)
}

func (s *accountantService) LowFundsNotification() {}
