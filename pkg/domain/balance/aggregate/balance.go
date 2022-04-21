package aggregate

import "github.com/lunemec/eve-accountant/pkg/domain/balance/entity"

type AmountByType map[entity.RefType]entity.Amount

type Balance struct {
	AmountByType AmountByType
	Income       entity.Amount
	Expenses     entity.Amount
}

func NewBalance() *Balance {
	return &Balance{
		AmountByType: make(AmountByType),
	}
}

func (b *Balance) Sum(other *Balance) {
	b.Income += other.Income
	b.Expenses += other.Expenses
	b.AmountByType.sum(other.AmountByType)
}

func (abt AmountByType) sum(other AmountByType) {
	for refType, amount := range other {
		abt[refType] += amount
	}
}
