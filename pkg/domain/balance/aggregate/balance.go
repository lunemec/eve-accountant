package aggregate

import (
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
)

type AmountByType map[entity.RefType]entity.Amount
type AmountByDivision map[entity.DivisionName]entity.Amount

type Balance struct {
	Income   entity.Amount
	Expenses entity.Amount
}

func NewBalance() *Balance {
	return &Balance{}
}

func (b *Balance) Balance() entity.Amount {
	return b.Income + b.Expenses // We must add because Expenses is negative.
}

func (b *Balance) Sum(other *Balance) {
	b.Income += other.Income
	b.Expenses += other.Expenses
}

type BalanceByType struct {
	IncomeByType   AmountByType
	ExpensesByType AmountByType
}

func NewBalanceByType() *BalanceByType {
	return &BalanceByType{
		IncomeByType:   make(AmountByType),
		ExpensesByType: make(AmountByType),
	}
}

func (b *BalanceByType) Sum(other *BalanceByType) {
	b.IncomeByType.sum(other.IncomeByType)
	b.ExpensesByType.sum(other.ExpensesByType)
}

type BalanceByDivision struct {
	IncomeByDivision   AmountByDivision
	ExpensesByDivision AmountByDivision
}

func NewBalanceByDivision() *BalanceByDivision {
	return &BalanceByDivision{
		IncomeByDivision:   make(AmountByDivision),
		ExpensesByDivision: make(AmountByDivision),
	}
}

func (b *BalanceByDivision) Sum(other *BalanceByDivision) {
	b.IncomeByDivision.sum(other.IncomeByDivision)
	b.ExpensesByDivision.sum(other.ExpensesByDivision)
}

func (abt AmountByType) sum(other AmountByType) {
	for refType, amount := range other {
		abt[refType] += amount
	}
}

func (abt AmountByDivision) sum(other AmountByDivision) {
	for divisionName, amount := range other {
		abt[divisionName] += amount
	}
}

type MonthlyBalanceNotification struct {
	Threshold          entity.Amount
	DateStart, DateEnd time.Time
	Balance            Balance
}
