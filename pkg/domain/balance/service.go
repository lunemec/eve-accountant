package balance

import (
	"context"
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
	"github.com/pkg/errors"
)

type Service interface {
	Balance(ctx context.Context, from, to time.Time) (*aggregate.Balance, error)
	BalanceByDivision(ctx context.Context, from, to time.Time) (*aggregate.BalanceByDivision, error)
	BalanceByType(ctx context.Context, from, to time.Time) (*aggregate.BalanceByType, error)
	BalanceByDayByDivisionByType(ctx context.Context, from, to time.Time) ([]*aggregate.BalanceByDivisionByType, error)
}

type balanceService struct {
	repositories []Repository
}

func NewService(repositories ...Repository) *balanceService {
	return &balanceService{
		repositories: repositories,
	}
}

func (s *balanceService) Balance(ctx context.Context, from, to time.Time) (*aggregate.Balance, error) {
	totalBalance := aggregate.NewBalance()

	for _, repository := range s.repositories {
		balance, err := s.balanceForRepository(ctx, repository, from, to)
		if err != nil {
			return totalBalance, errors.Wrapf(err, "error loading balance for corporation: %d", repository.CorporationID())
		}
		totalBalance.Sum(balance)
	}
	return totalBalance, nil
}

func (s *balanceService) balanceForRepository(ctx context.Context, repository Repository, from, to time.Time) (*aggregate.Balance, error) {
	balance := aggregate.NewBalance()

	divisions, err := repository.WalletDivisions(ctx)
	if err != nil {
		return balance, errors.Wrapf(err, "error listing divisions for corporation: %d", repository.CorporationID())
	}
	for _, division := range divisions {
		journalRecords, err := repository.WalletJournal(ctx, division, from, to)
		if err != nil {
			return balance, errors.Wrap(err, "unable to list journal records")
		}

		for journalRecord := range journalRecords {
			if journalRecord.Amount > 0 {
				balance.Income += journalRecord.Amount
			}
			if journalRecord.Amount < 0 {
				balance.Expenses += journalRecord.Amount
			}
		}
	}

	return balance, nil
}

func (s *balanceService) BalanceByDayByDivisionByType(ctx context.Context, from, to time.Time) ([]*aggregate.BalanceByDivisionByType, error) {
	var dailyBalance []*aggregate.BalanceByDivisionByType

	for d := from; d.After(to) == false; d = d.AddDate(0, 0, 1) {
		fromTime := d
		toTime := d.Add(24*time.Hour - 1*time.Nanosecond)
		dayBalance := aggregate.NewBalanceByDivisionByType(fromTime)
		for _, repository := range s.repositories {
			balance, err := s.balanceByDayByDivisionByType(ctx, repository, fromTime, toTime)
			if err != nil {
				return nil, errors.Wrapf(err, "error loading balance for corporation: %d", repository.CorporationID())
			}
			dayBalance.Sum(balance)
		}
		dailyBalance = append(dailyBalance, dayBalance)
	}

	return dailyBalance, nil
}

func (s *balanceService) balanceByDayByDivisionByType(ctx context.Context, repository Repository, from, to time.Time) (*aggregate.BalanceByDivisionByType, error) {
	balanceByDivisionByType := aggregate.NewBalanceByDivisionByType(from)

	divisions, err := repository.WalletDivisions(ctx)
	if err != nil {
		return balanceByDivisionByType, errors.Wrapf(err, "error listing divisions for corporation: %d", repository.CorporationID())
	}
	for _, division := range divisions {
		journalRecords, err := repository.WalletJournal(ctx, division, from, to)
		if err != nil {
			return balanceByDivisionByType, errors.Wrap(err, "unable to list journal records")
		}

		divisionName := division.Name
		if divisionName == "" {
			divisionName = "Main"
		}

		for journalRecord := range journalRecords {
			typeOut, ok := refTypeGroups[journalRecord.RefType]
			if !ok {
				typeOut = journalRecord.RefType
			}

			if journalRecord.Amount > 0 {
				balanceByDivisionByType.Income.Record(divisionName, typeOut, journalRecord.Amount)
			}
			if journalRecord.Amount < 0 {
				balanceByDivisionByType.Expenses.Record(divisionName, typeOut, journalRecord.Amount)
			}
		}
	}

	return balanceByDivisionByType, nil
}

func (s *balanceService) BalanceByDivision(ctx context.Context, from, to time.Time) (*aggregate.BalanceByDivision, error) {
	totalBalance := aggregate.NewBalanceByDivision()

	for _, repository := range s.repositories {
		balance, err := s.balanceByDivisionForRepository(ctx, repository, from, to)
		if err != nil {
			return totalBalance, errors.Wrapf(err, "error loading balance for corporation: %d", repository.CorporationID())
		}
		totalBalance.Sum(balance)
	}
	return totalBalance, nil
}

func (s *balanceService) balanceByDivisionForRepository(ctx context.Context, repository Repository, from, to time.Time) (*aggregate.BalanceByDivision, error) {
	balance := aggregate.NewBalanceByDivision()

	divisions, err := repository.WalletDivisions(ctx)
	if err != nil {
		return balance, errors.Wrapf(err, "error listing divisions for corporation: %d", repository.CorporationID())
	}
	for _, division := range divisions {
		divisionName := division.Name
		if divisionName == "" {
			divisionName = "Main"
		}
		journalRecords, err := repository.WalletJournal(ctx, division, from, to)
		if err != nil {
			return balance, errors.Wrap(err, "unable to list journal records")
		}

		for journalRecord := range journalRecords {
			if journalRecord.Amount > 0 {
				balance.IncomeByDivision[divisionName] += journalRecord.Amount
			}
			if journalRecord.Amount < 0 {
				balance.ExpensesByDivision[divisionName] += journalRecord.Amount
			}
		}
	}

	return balance, nil
}

func (s *balanceService) BalanceByType(ctx context.Context, from, to time.Time) (*aggregate.BalanceByType, error) {
	totalBalance := aggregate.NewBalanceByType()

	for _, repository := range s.repositories {
		balance, err := s.balanceByTypeForRepository(ctx, repository, from, to)
		if err != nil {
			return totalBalance, errors.Wrapf(err, "error loading balance for corporation: %d", repository.CorporationID())
		}
		totalBalance.Sum(balance)
	}
	return s.groupTypes(totalBalance), nil
}

func (s *balanceService) balanceByTypeForRepository(ctx context.Context, repository Repository, from, to time.Time) (*aggregate.BalanceByType, error) {
	balance := aggregate.NewBalanceByType()

	divisions, err := repository.WalletDivisions(ctx)
	if err != nil {
		return balance, errors.Wrapf(err, "error listing divisions for corporation: %d", repository.CorporationID())
	}
	for _, division := range divisions {
		journalRecords, err := repository.WalletJournal(ctx, division, from, to)
		if err != nil {
			return balance, errors.Wrap(err, "unable to list journal records")
		}

		for journalRecord := range journalRecords {
			if journalRecord.Amount > 0 {
				balance.IncomeByType[journalRecord.RefType] += journalRecord.Amount
			}
			if journalRecord.Amount < 0 {
				balance.ExpensesByType[journalRecord.RefType] += journalRecord.Amount
			}
		}
	}

	return balance, nil
}

var (
	marketTransactionType = entity.RefType("Market Transaction")
	contractPriceType     = entity.RefType("Contracts")
	playerWalletAction    = entity.RefType("Player Wallet Action")
	cloneTaxType          = entity.RefType("Clone Tax")
	industryTaxType       = entity.RefType("Industry Tax")
	piTaxType             = entity.RefType("PI Tax")
	rewardType            = entity.RefType("Krab Tax")
	jobCostType           = entity.RefType("Job Costs")
	feeType               = entity.RefType("Fee")

	refTypeGroups = map[entity.RefType]entity.RefType{
		entity.RefType("market_transaction"):                marketTransactionType,
		entity.RefType("contract_price"):                    contractPriceType,
		entity.RefType("player_donation"):                   playerWalletAction,
		entity.RefType("corporation_account_withdrawal"):    playerWalletAction,
		entity.RefType("jump_clone_activation_fee"):         cloneTaxType,
		entity.RefType("jump_clone_installation_fee"):       cloneTaxType,
		entity.RefType("industry_job_tax"):                  industryTaxType,
		entity.RefType("reprocessing_tax"):                  industryTaxType,
		entity.RefType("planetary_export_tax"):              piTaxType,
		entity.RefType("planetary_import_tax"):              piTaxType,
		entity.RefType("contract_deposit_refund"):           contractPriceType,
		entity.RefType("contract_auction_bid_refund"):       contractPriceType,
		entity.RefType("contract_auction_sold"):             contractPriceType,
		entity.RefType("insurance"):                         rewardType,
		entity.RefType("bounty_prizes"):                     rewardType,
		entity.RefType("corporate_reward_payout"):           rewardType,
		entity.RefType("project_discovery_reward"):          rewardType,
		entity.RefType("agent_mission_reward"):              rewardType,
		entity.RefType("agent_mission_time_bonus_reward"):   rewardType,
		entity.RefType("ess_escrow_transfer"):               rewardType,
		entity.RefType("researching_technology"):            jobCostType,
		entity.RefType("researching_time_productivity"):     jobCostType,
		entity.RefType("copying"):                           jobCostType,
		entity.RefType("researching_material_productivity"): jobCostType,
		entity.RefType("reaction"):                          jobCostType,
		entity.RefType("manufacturing"):                     jobCostType,
		entity.RefType("alliance_maintainance_fee"):         feeType,
		entity.RefType("office_rental_fee"):                 feeType,
		entity.RefType("contract_sales_tax"):                contractPriceType,
		entity.RefType("contract_brokers_fee_corp"):         contractPriceType,
		entity.RefType("contract_auction_bid_corp"):         contractPriceType,
		entity.RefType("contract_deposit_corp"):             contractPriceType,
		entity.RefType("contract_reward_deposited_corp"):    contractPriceType,
		entity.RefType("contract_price_payment_corp"):       contractPriceType,
		entity.RefType("contract_reward_refund"):            contractPriceType,
		entity.RefType("brokers_fee"):                       marketTransactionType,
		entity.RefType("transaction_tax"):                   marketTransactionType,
		entity.RefType("market_escrow"):                     marketTransactionType,
	}
)

func (s *balanceService) groupTypes(balance *aggregate.BalanceByType) *aggregate.BalanceByType {
	balanceOut := aggregate.NewBalanceByType()

	for typeIn, amountIn := range balance.IncomeByType {
		typeOut, ok := refTypeGroups[typeIn]
		if !ok {
			typeOut = typeIn
		}
		balanceOut.IncomeByType[typeOut] += amountIn
	}
	for typeIn, amountIn := range balance.ExpensesByType {
		typeOut, ok := refTypeGroups[typeIn]
		if !ok {
			typeOut = typeIn
		}
		balanceOut.ExpensesByType[typeOut] += amountIn
	}

	return balanceOut
}
